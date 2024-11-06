package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"

	"github.com/DharitriOne/drt-chain-core-go/core"
	"github.com/DharitriOne/drt-chain-core-go/core/check"
	"github.com/DharitriOne/drt-chain-core-go/data/dcdt"
	"github.com/DharitriOne/drt-chain-core-go/data/vm"
	vmcommon "github.com/DharitriOne/drt-chain-vm-common-go"
)

var zero = big.NewInt(0)

type dcdtTransfer struct {
	baseAlwaysActiveHandler
	funcGasCost           uint64
	marshaller            vmcommon.Marshalizer
	keyPrefix             []byte
	globalSettingsHandler vmcommon.ExtendedDCDTGlobalSettingsHandler
	payableHandler        vmcommon.PayableChecker
	shardCoordinator      vmcommon.Coordinator
	mutExecution          sync.RWMutex

	rolesHandler        vmcommon.DCDTRoleHandler
	enableEpochsHandler vmcommon.EnableEpochsHandler
}

// NewDCDTTransferFunc returns the dcdt transfer built-in function component
func NewDCDTTransferFunc(
	funcGasCost uint64,
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ExtendedDCDTGlobalSettingsHandler,
	shardCoordinator vmcommon.Coordinator,
	rolesHandler vmcommon.DCDTRoleHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*dcdtTransfer, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(shardCoordinator) {
		return nil, ErrNilShardCoordinator
	}
	if check.IfNil(rolesHandler) {
		return nil, ErrNilRolesHandler
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	e := &dcdtTransfer{
		funcGasCost:           funcGasCost,
		marshaller:            marshaller,
		keyPrefix:             []byte(baseDCDTKeyPrefix),
		globalSettingsHandler: globalSettingsHandler,
		payableHandler:        &disabledPayableHandler{},
		shardCoordinator:      shardCoordinator,
		rolesHandler:          rolesHandler,
		enableEpochsHandler:   enableEpochsHandler,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dcdtTransfer) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.DCDTTransfer
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves DCDT transfer function calls
func (e *dcdtTransfer) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkBasicDCDTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	isTransferToMeta := e.shardCoordinator.ComputeId(vmInput.RecipientAddr) == core.MetachainShardId
	if isTransferToMeta {
		return nil, ErrInvalidRcvAddr
	}

	if e.enableEpochsHandler.IsFlagEnabled(ConsistentTokensValuesLengthCheckFlag) {
		if len(vmInput.Arguments[1]) > core.MaxLenForDCDTIssueMint {
			return nil, fmt.Errorf("%w: max length for dcdt transfer value is %d", ErrInvalidArguments, core.MaxLenForDCDTIssueMint)
		}
	}
	value := big.NewInt(0).SetBytes(vmInput.Arguments[1])
	if value.Cmp(zero) <= 0 {
		return nil, ErrNegativeValue
	}

	gasRemaining := computeGasRemaining(acntSnd, vmInput.GasProvided, e.funcGasCost)
	dcdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	tokenID := vmInput.Arguments[0]

	keyToCheck := dcdtTokenKey
	if e.enableEpochsHandler.IsFlagEnabled(CheckCorrectTokenIDForTransferRoleFlag) {
		keyToCheck = tokenID
	}

	err = checkIfTransferCanHappenWithLimitedTransfer(keyToCheck, dcdtTokenKey, vmInput.CallerAddr, vmInput.RecipientAddr, e.globalSettingsHandler, e.rolesHandler, acntSnd, acntDst, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	if !check.IfNil(acntSnd) {
		// gas is paid only by sender
		if vmInput.GasProvided < e.funcGasCost {
			return nil, ErrNotEnoughGas
		}

		err = addToDCDTBalance(acntSnd, dcdtTokenKey, big.NewInt(0).Neg(value), e.marshaller, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
		if err != nil {
			return nil, err
		}
	}

	isSCCallAfter := e.payableHandler.DetermineIsSCCallAfter(vmInput, vmInput.RecipientAddr, core.MinLenArgumentsDCDTTransfer)
	vmOutput := &vmcommon.VMOutput{GasRemaining: gasRemaining, ReturnCode: vmcommon.Ok}
	if !check.IfNil(acntDst) {
		err = e.payableHandler.CheckPayable(vmInput, vmInput.RecipientAddr, core.MinLenArgumentsDCDTTransfer)
		if err != nil {
			return nil, err
		}

		err = addToDCDTBalance(acntDst, dcdtTokenKey, value, e.marshaller, e.globalSettingsHandler, vmInput.ReturnCallAfterError)
		if err != nil {
			return nil, err
		}

		if isSCCallAfter {
			vmOutput.GasRemaining, _ = vmcommon.SafeSubUint64(vmInput.GasProvided, e.funcGasCost)
			var callArgs [][]byte
			if len(vmInput.Arguments) > core.MinLenArgumentsDCDTTransfer+1 {
				callArgs = vmInput.Arguments[core.MinLenArgumentsDCDTTransfer+1:]
			}

			addOutputTransferToVMOutput(
				1,
				vmInput.CallerAddr,
				string(vmInput.Arguments[core.MinLenArgumentsDCDTTransfer]),
				callArgs,
				vmInput.RecipientAddr,
				vmInput.GasLocked,
				vmInput.CallType,
				vmOutput)

			addDCDTEntryForTransferInVMOutput(
				vmInput, vmOutput,
				[]byte(core.BuiltInFunctionDCDTTransfer),
				acntDst.AddressBytes(),
				[]*TopicTokenData{{
					tokenID,
					0,
					value,
				}},
			)
			return vmOutput, nil
		}

		if vmInput.CallType == vm.AsynchronousCallBack && check.IfNil(acntSnd) {
			// gas was already consumed on sender shard
			vmOutput.GasRemaining = vmInput.GasProvided
		}

		addDCDTEntryForTransferInVMOutput(
			vmInput, vmOutput,
			[]byte(core.BuiltInFunctionDCDTTransfer),
			acntDst.AddressBytes(),
			[]*TopicTokenData{{
				tokenID,
				0,
				value,
			}})
		return vmOutput, nil
	}

	// cross-shard DCDT transfer call through a smart contract
	if vmcommon.IsSmartContractAddress(vmInput.CallerAddr) {
		addOutputTransferToVMOutput(
			1,
			vmInput.CallerAddr,
			core.BuiltInFunctionDCDTTransfer,
			vmInput.Arguments,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	addDCDTEntryForTransferInVMOutput(
		vmInput, vmOutput,
		[]byte(core.BuiltInFunctionDCDTTransfer),
		vmInput.RecipientAddr,
		[]*TopicTokenData{{
			tokenID,
			0,
			value,
		}})
	return vmOutput, nil
}

func addOutputTransferToVMOutput(
	index uint32,
	senderAddress []byte,
	function string,
	arguments [][]byte,
	recipient []byte,
	gasLocked uint64,
	callType vm.CallType,
	vmOutput *vmcommon.VMOutput,
) {
	dcdtTransferTxData := function
	for _, arg := range arguments {
		dcdtTransferTxData += "@" + hex.EncodeToString(arg)
	}
	outTransfer := vmcommon.OutputTransfer{
		Index:         index,
		Value:         big.NewInt(0),
		GasLimit:      vmOutput.GasRemaining,
		GasLocked:     gasLocked,
		Data:          []byte(dcdtTransferTxData),
		CallType:      callType,
		SenderAddress: senderAddress,
	}
	vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
	vmOutput.OutputAccounts[string(recipient)] = &vmcommon.OutputAccount{
		Address:         recipient,
		OutputTransfers: []vmcommon.OutputTransfer{outTransfer},
	}
	vmOutput.GasRemaining = 0
}

func addToDCDTBalance(
	userAcnt vmcommon.UserAccountHandler,
	key []byte,
	value *big.Int,
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.DCDTGlobalSettingsHandler,
	isReturnWithError bool,
) error {
	dcdtData, err := getDCDTDataFromKey(userAcnt, key, marshaller)
	if err != nil {
		return err
	}

	if dcdtData.Type != uint32(core.Fungible) {
		return ErrOnlyFungibleTokensHaveBalanceTransfer
	}

	err = checkFrozeAndPause(userAcnt.AddressBytes(), key, dcdtData, globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	dcdtData.Value.Add(dcdtData.Value, value)
	if dcdtData.Value.Cmp(zero) < 0 {
		return ErrInsufficientFunds
	}

	err = saveDCDTData(userAcnt, dcdtData, key, marshaller)
	if err != nil {
		return err
	}

	return nil
}

func checkFrozeAndPause(
	senderAddr []byte,
	key []byte,
	dcdtData *dcdt.DCDigitalToken,
	globalSettingsHandler vmcommon.DCDTGlobalSettingsHandler,
	isReturnWithError bool,
) error {
	if isReturnWithError {
		return nil
	}
	if bytes.Equal(senderAddr, core.DCDTSCAddress) {
		return nil
	}

	dcdtUserMetaData := DCDTUserMetadataFromBytes(dcdtData.Properties)
	if dcdtUserMetaData.Frozen {
		return ErrDCDTIsFrozenForAccount
	}

	if globalSettingsHandler.IsPaused(key) {
		return ErrDCDTTokenIsPaused
	}

	return nil
}

func arePropertiesEmpty(properties []byte) bool {
	for _, property := range properties {
		if property != 0 {
			return false
		}
	}
	return true
}

func saveDCDTData(
	userAcnt vmcommon.UserAccountHandler,
	dcdtData *dcdt.DCDigitalToken,
	key []byte,
	marshaller vmcommon.Marshalizer,
) error {
	isValueZero := dcdtData.Value.Cmp(zero) == 0
	if isValueZero && arePropertiesEmpty(dcdtData.Properties) {
		return userAcnt.AccountDataHandler().SaveKeyValue(key, nil)
	}

	marshaledData, err := marshaller.Marshal(dcdtData)
	if err != nil {
		return err
	}

	return userAcnt.AccountDataHandler().SaveKeyValue(key, marshaledData)
}

func getDCDTDataFromKey(
	userAcnt vmcommon.UserAccountHandler,
	key []byte,
	marshaller vmcommon.Marshalizer,
) (*dcdt.DCDigitalToken, error) {
	dcdtData := &dcdt.DCDigitalToken{Value: big.NewInt(0), Type: uint32(core.Fungible)}
	marshaledData, _, err := userAcnt.AccountDataHandler().RetrieveValue(key)
	if core.IsGetNodeFromDBError(err) {
		return nil, err
	}
	if err != nil || len(marshaledData) == 0 {
		return dcdtData, nil
	}

	err = marshaller.Unmarshal(dcdtData, marshaledData)
	if err != nil {
		return nil, err
	}

	return dcdtData, nil
}

// will return nil if transfer is not limited
// if we are at sender shard, the sender or the destination must have the transfer role
// we cannot transfer a limited dcdt to destination shard, as there we do not know if that token was transferred or not
// by an account with transfer account
func checkIfTransferCanHappenWithLimitedTransfer(
	tokenID []byte, dcdtTokenKey []byte,
	senderAddress, destinationAddress []byte,
	globalSettingsHandler vmcommon.ExtendedDCDTGlobalSettingsHandler,
	roleHandler vmcommon.DCDTRoleHandler,
	acntSnd, acntDst vmcommon.UserAccountHandler,
	isReturnWithError bool,
) error {
	if isReturnWithError {
		return nil
	}
	if check.IfNil(acntSnd) {
		return nil
	}
	if !globalSettingsHandler.IsLimitedTransfer(dcdtTokenKey) {
		return nil
	}

	if globalSettingsHandler.IsSenderOrDestinationWithTransferRole(senderAddress, destinationAddress, tokenID) {
		return nil
	}

	errSender := roleHandler.CheckAllowedToExecute(acntSnd, tokenID, []byte(core.DCDTRoleTransfer))
	if errSender == nil {
		return nil
	}

	errDestination := roleHandler.CheckAllowedToExecute(acntDst, tokenID, []byte(core.DCDTRoleTransfer))
	return errDestination
}

// SetPayableChecker will set the payableCheck handler to the function
func (e *dcdtTransfer) SetPayableChecker(payableHandler vmcommon.PayableChecker) error {
	if check.IfNil(payableHandler) {
		return ErrNilPayableHandler
	}

	e.payableHandler = payableHandler
	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dcdtTransfer) IsInterfaceNil() bool {
	return e == nil
}
