package builtInFunctions

import (
	"math/big"
	"sync"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/core/check"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
)

type dcdtNFTupdate struct {
	baseActiveHandler
	keyPrefix             []byte
	dcdtStorageHandler    vmcommon.DCDTNFTStorageHandler
	globalSettingsHandler vmcommon.DCDTGlobalSettingsHandler
	rolesHandler          vmcommon.DCDTRoleHandler
	gasConfig             vmcommon.BaseOperationCost
	funcGasCost           uint64
	mutExecution          sync.RWMutex
}

// NewDCDTNFTUpdateAttributesFunc returns the dcdt NFT update attribute built-in function component
func NewDCDTNFTUpdateAttributesFunc(
	funcGasCost uint64,
	gasConfig vmcommon.BaseOperationCost,
	dcdtStorageHandler vmcommon.DCDTNFTStorageHandler,
	globalSettingsHandler vmcommon.DCDTGlobalSettingsHandler,
	rolesHandler vmcommon.DCDTRoleHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*dcdtNFTupdate, error) {
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

	e := &dcdtNFTupdate{
		keyPrefix:             []byte(baseDCDTKeyPrefix),
		dcdtStorageHandler:    dcdtStorageHandler,
		funcGasCost:           funcGasCost,
		mutExecution:          sync.RWMutex{},
		globalSettingsHandler: globalSettingsHandler,
		gasConfig:             gasConfig,
		rolesHandler:          rolesHandler,
	}

	e.baseActiveHandler.activeHandler = func() bool {
		return enableEpochsHandler.IsFlagEnabled(DCDTNFTImprovementV1Flag)
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dcdtNFTupdate) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.DCDTNFTUpdateAttributes
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves DCDT NFT update attributes function call
// Requires 3 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - new attributes
func (e *dcdtNFTupdate) ProcessBuiltinFunction(
	acntSnd, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkDCDTNFTCreateBurnAddInput(acntSnd, vmInput, e.funcGasCost)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) != 3 {
		return nil, ErrInvalidArguments
	}

	err = e.rolesHandler.CheckAllowedToExecute(acntSnd, vmInput.Arguments[0], []byte(core.DCDTRoleNFTUpdateAttributes))
	if err != nil {
		return nil, err
	}

	gasCostForStore := uint64(len(vmInput.Arguments[2])) * e.gasConfig.StorePerByte
	if vmInput.GasProvided < e.funcGasCost+gasCostForStore {
		return nil, ErrNotEnoughGas
	}

	dcdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}
	dcdtData, err := e.dcdtStorageHandler.GetDCDTNFTTokenOnSender(acntSnd, dcdtTokenKey, nonce)
	if err != nil {
		return nil, err
	}

	dcdtData.TokenMetaData.Attributes = vmInput.Arguments[2]

	_, err = e.dcdtStorageHandler.SaveDCDTNFTToken(acntSnd.AddressBytes(), acntSnd, dcdtTokenKey, nonce, dcdtData, true, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost - gasCostForStore,
	}

	addDCDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionDCDTNFTUpdateAttributes), vmInput.Arguments[0], nonce, big.NewInt(0), vmInput.CallerAddr, vmInput.Arguments[2])

	return vmOutput, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dcdtNFTupdate) IsInterfaceNil() bool {
	return e == nil
}
