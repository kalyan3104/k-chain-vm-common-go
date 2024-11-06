package builtInFunctions

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/core/check"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
)

const maxLenForAddNFTQuantity = 32

type dcdtNFTAddQuantity struct {
	baseAlwaysActiveHandler
	keyPrefix             []byte
	globalSettingsHandler vmcommon.DCDTGlobalSettingsHandler
	rolesHandler          vmcommon.DCDTRoleHandler
	dcdtStorageHandler    vmcommon.DCDTNFTStorageHandler
	enableEpochsHandler   vmcommon.EnableEpochsHandler
	funcGasCost           uint64
	mutExecution          sync.RWMutex
}

// NewDCDTNFTAddQuantityFunc returns the dcdt NFT add quantity built-in function component
func NewDCDTNFTAddQuantityFunc(
	funcGasCost uint64,
	dcdtStorageHandler vmcommon.DCDTNFTStorageHandler,
	globalSettingsHandler vmcommon.DCDTGlobalSettingsHandler,
	rolesHandler vmcommon.DCDTRoleHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*dcdtNFTAddQuantity, error) {
	if check.IfNil(dcdtStorageHandler) {
		return nil, ErrNilDCDTNFTStorageHandler
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

	e := &dcdtNFTAddQuantity{
		keyPrefix:             []byte(baseDCDTKeyPrefix),
		globalSettingsHandler: globalSettingsHandler,
		rolesHandler:          rolesHandler,
		funcGasCost:           funcGasCost,
		mutExecution:          sync.RWMutex{},
		dcdtStorageHandler:    dcdtStorageHandler,
		enableEpochsHandler:   enableEpochsHandler,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dcdtNFTAddQuantity) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.DCDTNFTAddQuantity
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves DCDT NFT add quantity function call
// Requires 3 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - quantity to add
func (e *dcdtNFTAddQuantity) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkDCDTNFTCreateBurnAddInput(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) < 3 {
		return nil, ErrInvalidArguments
	}

	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[0], []byte(core.DCDTRoleNFTAddQuantity))
	if err != nil {
		return nil, err
	}

	dcdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	dcdtData, err := e.dcdtStorageHandler.GetDCDTNFTTokenOnSender(acntSnd, dcdtTokenKey, nonce)
	if err != nil {
		return nil, err
	}
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}

	isValueLengthCheckFlagEnabled := e.enableEpochsHandler.IsFlagEnabled(ValueLengthCheckFlag)
	if isValueLengthCheckFlagEnabled && len(vmInput.Arguments[2]) > maxLenForAddNFTQuantity {
		return nil, fmt.Errorf("%w max length for add nft quantity is %d", ErrInvalidArguments, maxLenForAddNFTQuantity)
	}

	value := big.NewInt(0).SetBytes(vmInput.Arguments[2])
	dcdtData.Value.Add(dcdtData.Value, value)

	_, err = e.dcdtStorageHandler.SaveDCDTNFTToken(acntSnd.AddressBytes(), acntSnd, dcdtTokenKey, nonce, dcdtData, false, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}
	err = e.dcdtStorageHandler.AddToLiquiditySystemAcc(dcdtTokenKey, nonce, value)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost,
	}

	addDCDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionDCDTNFTAddQuantity), vmInput.Arguments[0], nonce, value, vmInput.CallerAddr)

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dcdtNFTAddQuantity) IsInterfaceNil() bool {
	return e == nil
}
