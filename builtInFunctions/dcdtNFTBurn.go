package builtInFunctions

import (
	"math/big"
	"sync"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/core/check"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
)

type dcdtNFTBurn struct {
	baseAlwaysActiveHandler
	keyPrefix             []byte
	dcdtStorageHandler    vmcommon.DCDTNFTStorageHandler
	globalSettingsHandler vmcommon.ExtendedDCDTGlobalSettingsHandler
	rolesHandler          vmcommon.DCDTRoleHandler
	funcGasCost           uint64
	mutExecution          sync.RWMutex
}

// NewDCDTNFTBurnFunc returns the dcdt NFT burn built-in function component
func NewDCDTNFTBurnFunc(
	funcGasCost uint64,
	dcdtStorageHandler vmcommon.DCDTNFTStorageHandler,
	globalSettingsHandler vmcommon.ExtendedDCDTGlobalSettingsHandler,
	rolesHandler vmcommon.DCDTRoleHandler,
) (*dcdtNFTBurn, error) {
	if check.IfNil(dcdtStorageHandler) {
		return nil, ErrNilDCDTNFTStorageHandler
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}

	e := &dcdtNFTBurn{
		keyPrefix:             []byte(baseDCDTKeyPrefix),
		dcdtStorageHandler:    dcdtStorageHandler,
		globalSettingsHandler: globalSettingsHandler,
		rolesHandler:          rolesHandler,
		funcGasCost:           funcGasCost,
		mutExecution:          sync.RWMutex{},
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dcdtNFTBurn) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.DCDTNFTBurn
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves DCDT NFT burn function call
// Requires 3 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - quantity to burn
func (e *dcdtNFTBurn) ProcessBuiltinFunction(
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

	dcdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	err = e.isAllowedToBurn(acntSnd, vmInput.Arguments[0])
	if err != nil {
		return nil, err
	}

	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	dcdtData, err := e.dcdtStorageHandler.GetDCDTNFTTokenOnSender(acntSnd, dcdtTokenKey, nonce)
	if err != nil {
		return nil, err
	}
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}

	quantityToBurn := big.NewInt(0).SetBytes(vmInput.Arguments[2])
	if dcdtData.Value.Cmp(quantityToBurn) < 0 {
		return nil, ErrInvalidNFTQuantity
	}

	dcdtData.Value.Sub(dcdtData.Value, quantityToBurn)

	_, err = e.dcdtStorageHandler.SaveDCDTNFTToken(acntSnd.AddressBytes(), acntSnd, dcdtTokenKey, nonce, dcdtData, false, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	err = e.dcdtStorageHandler.AddToLiquiditySystemAcc(dcdtTokenKey, nonce, big.NewInt(0).Neg(quantityToBurn))
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost,
	}

	addDCDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionDCDTNFTBurn), vmInput.Arguments[0], nonce, quantityToBurn, vmInput.CallerAddr)

	return vmOutput, nil
}

func (e *dcdtNFTBurn) isAllowedToBurn(acntSnd vmcommon.UserAccountHandler, tokenID []byte) error {
	dcdtTokenKey := append(e.keyPrefix, tokenID...)
	isBurnForAll := e.globalSettingsHandler.IsBurnForAll(dcdtTokenKey)
	if isBurnForAll {
		return nil
	}

	return e.rolesHandler.CheckAllowedToExecute(acntSnd, tokenID, []byte(core.DCDTRoleNFTBurn))
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dcdtNFTBurn) IsInterfaceNil() bool {
	return e == nil
}
