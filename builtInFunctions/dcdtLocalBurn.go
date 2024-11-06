package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/DharitriOne/drt-chain-core-go/core"
	"github.com/DharitriOne/drt-chain-core-go/core/check"
	vmcommon "github.com/DharitriOne/drt-chain-vm-common-go"
)

type dcdtLocalBurn struct {
	baseAlwaysActiveHandler
	keyPrefix             []byte
	marshaller            vmcommon.Marshalizer
	globalSettingsHandler vmcommon.ExtendedDCDTGlobalSettingsHandler
	rolesHandler          vmcommon.DCDTRoleHandler
	enableEpochsHandler   vmcommon.EnableEpochsHandler
	funcGasCost           uint64
	mutExecution          sync.RWMutex
}

// NewDCDTLocalBurnFunc returns the dcdt local burn built-in function component
func NewDCDTLocalBurnFunc(
	funcGasCost uint64,
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ExtendedDCDTGlobalSettingsHandler,
	rolesHandler vmcommon.DCDTRoleHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*dcdtLocalBurn, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	e := &dcdtLocalBurn{
		keyPrefix:             []byte(baseDCDTKeyPrefix),
		marshaller:            marshaller,
		globalSettingsHandler: globalSettingsHandler,
		rolesHandler:          rolesHandler,
		funcGasCost:           funcGasCost,
		enableEpochsHandler:   enableEpochsHandler,
		mutExecution:          sync.RWMutex{},
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dcdtLocalBurn) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.DCDTLocalBurn
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves DCDT local burn function call
func (e *dcdtLocalBurn) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkInputArgumentsForLocalAction(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}

	tokenID := vmInput.Arguments[0]
	err = e.isAllowedToBurn(acntSnd, tokenID)
	if err != nil {
		return nil, err
	}

	if e.enableEpochsHandler.IsFlagEnabled(ConsistentTokensValuesLengthCheckFlag) {
		// TODO: core.MaxLenForDCDTIssueMint should be renamed to something more general, such as MaxLenForDCDTValues
		if len(vmInput.Arguments[1]) > core.MaxLenForDCDTIssueMint {
			return nil, fmt.Errorf("%w: max length for dcdt local burn value is %d", ErrInvalidArguments, core.MaxLenForDCDTIssueMint)
		}
	}
	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	dcdtTokenKey := append(e.keyPrefix, tokenID...)
	err = addToDCDTBalance(acntSnd, dcdtTokenKey, big.NewInt(0).Neg(value), e.marshaller, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok, GasRemaining: vmInput.GasProvided - e.funcGasCost}

	addDCDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionDCDTLocalBurn), vmInput.Arguments[0], 0, value, vmInput.CallerAddr)

	return vmOutput, nil
}

func (e *dcdtLocalBurn) isAllowedToBurn(acntSnd vmcommon.UserAccountHandler, tokenID []byte) error {
	dcdtTokenKey := append(e.keyPrefix, tokenID...)
	isBurnForAll := e.globalSettingsHandler.IsBurnForAll(dcdtTokenKey)
	if isBurnForAll {
		return nil
	}

	return e.rolesHandler.CheckAllowedToExecute(acntSnd, tokenID, []byte(core.DCDTRoleLocalBurn))
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dcdtLocalBurn) IsInterfaceNil() bool {
	return e == nil
}

func checkBasicDCDTArguments(vmInput *vmcommon.ContractCallInput) error {
	if vmInput == nil {
		return ErrNilVmInput
	}
	if vmInput.CallValue == nil {
		return ErrNilValue
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) < core.MinLenArgumentsDCDTTransfer {
		return ErrInvalidArguments
	}
	return nil
}

func checkInputArgumentsForLocalAction(
	acntSnd vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
	funcGasCost uint64,
) error {
	err := checkBasicDCDTArguments(vmInput)
	if err != nil {
		return err
	}
	if !bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return ErrInvalidRcvAddr
	}
	if check.IfNil(acntSnd) {
		return ErrNilUserAccount
	}
	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	if value.Cmp(zero) <= 0 {
		return ErrNegativeValue
	}
	if vmInput.GasProvided < funcGasCost {
		return ErrNotEnoughGas
	}

	return nil
}
