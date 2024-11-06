package builtInFunctions

import (
	"bytes"
	"math/big"
	"sync"

	"github.com/DharitriOne/drt-chain-core-go/core"
	"github.com/DharitriOne/drt-chain-core-go/core/check"
	vmcommon "github.com/DharitriOne/drt-chain-vm-common-go"
)

type dcdtBurn struct {
	baseActiveHandler
	funcGasCost           uint64
	marshaller            vmcommon.Marshalizer
	keyPrefix             []byte
	globalSettingsHandler vmcommon.DCDTGlobalSettingsHandler
	mutExecution          sync.RWMutex
}

// NewDCDTBurnFunc returns the dcdt burn built-in function component
func NewDCDTBurnFunc(
	funcGasCost uint64,
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.DCDTGlobalSettingsHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*dcdtBurn, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	e := &dcdtBurn{
		funcGasCost:           funcGasCost,
		marshaller:            marshaller,
		keyPrefix:             []byte(baseDCDTKeyPrefix),
		globalSettingsHandler: globalSettingsHandler,
	}

	e.baseActiveHandler.activeHandler = func() bool {
		return enableEpochsHandler.IsFlagEnabled(GlobalMintBurnFlag)
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dcdtBurn) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.DCDTBurn
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves DCDT burn function call
func (e *dcdtBurn) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkBasicDCDTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) != 2 {
		return nil, ErrInvalidArguments
	}
	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	if value.Cmp(zero) <= 0 {
		return nil, ErrNegativeValue
	}
	if !bytes.Equal(vmInput.RecipientAddr, core.DCDTSCAddress) {
		return nil, ErrAddressIsNotDCDTSystemSC
	}
	if check.IfNil(acntSnd) {
		return nil, ErrNilUserAccount
	}

	dcdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)

	if vmInput.GasProvided < e.funcGasCost {
		return nil, ErrNotEnoughGas
	}

	err = addToDCDTBalance(acntSnd, dcdtTokenKey, big.NewInt(0).Neg(value), e.marshaller, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	gasRemaining := computeGasRemaining(acntSnd, vmInput.GasProvided, e.funcGasCost)
	vmOutput := &vmcommon.VMOutput{GasRemaining: gasRemaining, ReturnCode: vmcommon.Ok}
	if vmcommon.IsSmartContractAddress(vmInput.CallerAddr) {
		addOutputTransferToVMOutput(
			1,
			vmInput.CallerAddr,
			core.BuiltInFunctionDCDTBurn,
			vmInput.Arguments,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	addDCDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionDCDTBurn), vmInput.Arguments[0], 0, value, vmInput.CallerAddr)

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dcdtBurn) IsInterfaceNil() bool {
	return e == nil
}
