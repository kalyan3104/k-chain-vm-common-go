package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/core/check"
	"github.com/kalyan3104/k-chain-core-go/data/dcdt"
	"github.com/kalyan3104/k-chain-core-go/data/vm"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
)

const baseDCDTKeyPrefix = core.ProtectedKeyPrefix + core.DCDTKeyIdentifier

var oneValue = big.NewInt(1)
var zeroByteArray = []byte{0}

type dcdtNFTTransfer struct {
	baseAlwaysActiveHandler
	keyPrefix             []byte
	marshaller            vmcommon.Marshalizer
	globalSettingsHandler vmcommon.ExtendedDCDTGlobalSettingsHandler
	payableHandler        vmcommon.PayableChecker
	funcGasCost           uint64
	accounts              vmcommon.AccountsAdapter
	shardCoordinator      vmcommon.Coordinator
	gasConfig             vmcommon.BaseOperationCost
	mutExecution          sync.RWMutex
	rolesHandler          vmcommon.DCDTRoleHandler
	dcdtStorageHandler    vmcommon.DCDTNFTStorageHandler
	enableEpochsHandler   vmcommon.EnableEpochsHandler
}

// NewDCDTNFTTransferFunc returns the dcdt NFT transfer built-in function component
func NewDCDTNFTTransferFunc(
	funcGasCost uint64,
	marshaller vmcommon.Marshalizer,
	globalSettingsHandler vmcommon.ExtendedDCDTGlobalSettingsHandler,
	accounts vmcommon.AccountsAdapter,
	shardCoordinator vmcommon.Coordinator,
	gasConfig vmcommon.BaseOperationCost,
	rolesHandler vmcommon.DCDTRoleHandler,
	dcdtStorageHandler vmcommon.DCDTNFTStorageHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*dcdtNFTTransfer, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(globalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
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
	if check.IfNil(dcdtStorageHandler) {
		return nil, ErrNilDCDTNFTStorageHandler
	}

	e := &dcdtNFTTransfer{
		keyPrefix:             []byte(baseDCDTKeyPrefix),
		marshaller:            marshaller,
		globalSettingsHandler: globalSettingsHandler,
		funcGasCost:           funcGasCost,
		accounts:              accounts,
		shardCoordinator:      shardCoordinator,
		gasConfig:             gasConfig,
		mutExecution:          sync.RWMutex{},
		payableHandler:        &disabledPayableHandler{},
		rolesHandler:          rolesHandler,
		enableEpochsHandler:   enableEpochsHandler,
		dcdtStorageHandler:    dcdtStorageHandler,
	}

	return e, nil
}

// SetPayableChecker will set the payableCheck handler to the function
func (e *dcdtNFTTransfer) SetPayableChecker(payableHandler vmcommon.PayableChecker) error {
	if check.IfNil(payableHandler) {
		return ErrNilPayableHandler
	}

	e.payableHandler = payableHandler
	return nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dcdtNFTTransfer) SetNewGasConfig(gasCost *vmcommon.GasCost) {
	if gasCost == nil {
		return
	}

	e.mutExecution.Lock()
	e.funcGasCost = gasCost.BuiltInCost.DCDTNFTTransfer
	e.gasConfig = gasCost.BaseOperationCost
	e.mutExecution.Unlock()
}

// ProcessBuiltinFunction resolves DCDT NFT transfer roles function call
// Requires 4 arguments:
// arg0 - token identifier
// arg1 - nonce
// arg2 - quantity to transfer
// arg3 - destination address
// if cross-shard, the rest of arguments will be filled inside the SCR
func (e *dcdtNFTTransfer) ProcessBuiltinFunction(
	acntSnd, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	e.mutExecution.RLock()
	defer e.mutExecution.RUnlock()

	err := checkBasicDCDTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if len(vmInput.Arguments) < 4 {
		return nil, ErrInvalidArguments
	}

	if bytes.Equal(vmInput.CallerAddr, vmInput.RecipientAddr) {
		return e.processNFTTransferOnSenderShard(acntSnd, vmInput)
	}

	// in cross shard NFT transfer the sender account must be nil
	if !check.IfNil(acntSnd) {
		return nil, ErrInvalidRcvAddr
	}
	if check.IfNil(acntDst) {
		return nil, ErrInvalidRcvAddr
	}

	tickerID := vmInput.Arguments[0]
	dcdtTokenKey := append(e.keyPrefix, tickerID...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	value := big.NewInt(0).SetBytes(vmInput.Arguments[2])

	dcdtTransferData := &dcdt.DCDigitalToken{}
	if !bytes.Equal(vmInput.Arguments[3], zeroByteArray) {
		marshaledNFTTransfer := vmInput.Arguments[3]
		err = e.marshaller.Unmarshal(dcdtTransferData, marshaledNFTTransfer)
		if err != nil {
			return nil, err
		}
	} else {
		dcdtTransferData.Value = big.NewInt(0).Set(value)
		dcdtTransferData.Type = uint32(core.NonFungible)
	}

	err = e.payableHandler.CheckPayable(vmInput, vmInput.RecipientAddr, core.MinLenArgumentsDCDTNFTTransfer)
	if err != nil {
		return nil, err
	}
	err = e.addNFTToDestination(vmInput.CallerAddr, vmInput.RecipientAddr, acntDst, dcdtTransferData, dcdtTokenKey, nonce, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	// no need to consume gas on destination - sender already paid for it
	vmOutput := &vmcommon.VMOutput{GasRemaining: vmInput.GasProvided}
	if len(vmInput.Arguments) > core.MinLenArgumentsDCDTNFTTransfer && vmcommon.IsSmartContractAddress(vmInput.RecipientAddr) {
		var callArgs [][]byte
		if len(vmInput.Arguments) > core.MinLenArgumentsDCDTNFTTransfer+1 {
			callArgs = vmInput.Arguments[core.MinLenArgumentsDCDTNFTTransfer+1:]
		}

		addOutputTransferToVMOutput(
			1,
			vmInput.CallerAddr,
			string(vmInput.Arguments[core.MinLenArgumentsDCDTNFTTransfer]),
			callArgs,
			vmInput.RecipientAddr,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	addDCDTEntryForTransferInVMOutput(
		vmInput, vmOutput,
		[]byte(core.BuiltInFunctionDCDTNFTTransfer),
		acntDst.AddressBytes(),
		[]*TopicTokenData{{
			vmInput.Arguments[0],
			nonce,
			value,
		}},
	)

	return vmOutput, nil
}

func (e *dcdtNFTTransfer) processNFTTransferOnSenderShard(
	acntSnd vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	dstAddress := vmInput.Arguments[3]
	if len(dstAddress) != len(vmInput.CallerAddr) {
		return nil, fmt.Errorf("%w, not a valid destination address", ErrInvalidArguments)
	}
	if bytes.Equal(dstAddress, vmInput.CallerAddr) {
		return nil, fmt.Errorf("%w, can not transfer to self", ErrInvalidArguments)
	}
	isTransferToMeta := e.shardCoordinator.ComputeId(dstAddress) == core.MetachainShardId
	if isTransferToMeta {
		return nil, ErrInvalidRcvAddr
	}
	if vmInput.GasProvided < e.funcGasCost {
		return nil, ErrNotEnoughGas
	}

	tickerID := vmInput.Arguments[0]
	dcdtTokenKey := append(e.keyPrefix, tickerID...)
	nonce := big.NewInt(0).SetBytes(vmInput.Arguments[1]).Uint64()
	dcdtData, err := e.dcdtStorageHandler.GetDCDTNFTTokenOnSender(acntSnd, dcdtTokenKey, nonce)
	if err != nil {
		return nil, err
	}
	if nonce == 0 {
		return nil, ErrNFTDoesNotHaveMetadata
	}

	if len(vmInput.Arguments[2]) > core.MaxLenForDCDTIssueMint && e.enableEpochsHandler.IsFlagEnabled(ConsistentTokensValuesLengthCheckFlag) {
		return nil, fmt.Errorf("%w: max length for a transfer value is %d", ErrInvalidArguments, core.MaxLenForDCDTIssueMint)
	}
	quantityToTransfer := big.NewInt(0).SetBytes(vmInput.Arguments[2])
	if dcdtData.Value.Cmp(quantityToTransfer) < 0 {
		return nil, ErrInvalidNFTQuantity
	}

	isCheckTransferFlagEnabled := e.enableEpochsHandler.IsFlagEnabled(CheckTransferFlag)
	if isCheckTransferFlagEnabled && quantityToTransfer.Cmp(zero) <= 0 {
		return nil, ErrInvalidNFTQuantity
	}
	dcdtData.Value.Sub(dcdtData.Value, quantityToTransfer)

	_, err = e.dcdtStorageHandler.SaveDCDTNFTToken(acntSnd.AddressBytes(), acntSnd, dcdtTokenKey, nonce, dcdtData, false, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	dcdtData.Value.Set(quantityToTransfer)

	var userAccount vmcommon.UserAccountHandler
	if e.shardCoordinator.SelfId() == e.shardCoordinator.ComputeId(dstAddress) {
		accountHandler, errLoad := e.accounts.LoadAccount(dstAddress)
		if errLoad != nil {
			return nil, errLoad
		}

		var ok bool
		userAccount, ok = accountHandler.(vmcommon.UserAccountHandler)
		if !ok {
			return nil, ErrWrongTypeAssertion
		}

		err = e.payableHandler.CheckPayable(vmInput, dstAddress, core.MinLenArgumentsDCDTNFTTransfer)
		if err != nil {
			return nil, err
		}
		err = e.addNFTToDestination(vmInput.CallerAddr, dstAddress, userAccount, dcdtData, dcdtTokenKey, nonce, vmInput.ReturnCallAfterError)
		if err != nil {
			return nil, err
		}

		err = e.accounts.SaveAccount(userAccount)
		if err != nil {
			return nil, err
		}
	} else {
		err = e.dcdtStorageHandler.AddToLiquiditySystemAcc(dcdtTokenKey, nonce, big.NewInt(0).Neg(quantityToTransfer))
		if err != nil {
			return nil, err
		}
	}

	tokenID := dcdtTokenKey
	if e.enableEpochsHandler.IsFlagEnabled(CheckCorrectTokenIDForTransferRoleFlag) {
		tokenID = tickerID
	}

	err = checkIfTransferCanHappenWithLimitedTransfer(tokenID, dcdtTokenKey, acntSnd.AddressBytes(), dstAddress, e.globalSettingsHandler, e.rolesHandler, acntSnd, userAccount, vmInput.ReturnCallAfterError)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: vmInput.GasProvided - e.funcGasCost,
	}
	err = e.createNFTOutputTransfers(vmInput, vmOutput, dcdtData, dstAddress, tickerID, nonce)
	if err != nil {
		return nil, err
	}

	addDCDTEntryForTransferInVMOutput(
		vmInput, vmOutput,
		[]byte(core.BuiltInFunctionDCDTNFTTransfer),
		dstAddress,
		[]*TopicTokenData{{
			vmInput.Arguments[0],
			nonce,
			quantityToTransfer,
		}},
	)

	return vmOutput, nil
}

func (e *dcdtNFTTransfer) createNFTOutputTransfers(
	vmInput *vmcommon.ContractCallInput,
	vmOutput *vmcommon.VMOutput,
	dcdtTransferData *dcdt.DCDigitalToken,
	dstAddress []byte,
	tickerID []byte,
	nonce uint64,
) error {
	nftTransferCallArgs := make([][]byte, 0)
	nftTransferCallArgs = append(nftTransferCallArgs, vmInput.Arguments[:3]...)

	wasAlreadySent, err := e.dcdtStorageHandler.WasAlreadySentToDestinationShardAndUpdateState(tickerID, nonce, dstAddress)
	if err != nil {
		return err
	}

	if !wasAlreadySent || dcdtTransferData.Value.Cmp(oneValue) == 0 {
		marshaledNFTTransfer, err := e.marshaller.Marshal(dcdtTransferData)
		if err != nil {
			return err
		}

		gasForTransfer := uint64(len(marshaledNFTTransfer)) * e.gasConfig.DataCopyPerByte
		if gasForTransfer > vmOutput.GasRemaining {
			return ErrNotEnoughGas
		}
		vmOutput.GasRemaining -= gasForTransfer
		nftTransferCallArgs = append(nftTransferCallArgs, marshaledNFTTransfer)
	} else {
		nftTransferCallArgs = append(nftTransferCallArgs, zeroByteArray)
	}

	if len(vmInput.Arguments) > core.MinLenArgumentsDCDTNFTTransfer {
		nftTransferCallArgs = append(nftTransferCallArgs, vmInput.Arguments[4:]...)
	}

	isSCCallAfter := e.payableHandler.DetermineIsSCCallAfter(vmInput, dstAddress, core.MinLenArgumentsDCDTNFTTransfer)

	if e.shardCoordinator.SelfId() != e.shardCoordinator.ComputeId(dstAddress) {
		gasToTransfer := uint64(0)
		if isSCCallAfter {
			gasToTransfer = vmOutput.GasRemaining
			vmOutput.GasRemaining = 0
		}
		addNFTTransferToVMOutput(
			1,
			vmInput.CallerAddr,
			dstAddress,
			core.BuiltInFunctionDCDTNFTTransfer,
			nftTransferCallArgs,
			vmInput.GasLocked,
			gasToTransfer,
			vmInput.CallType,
			vmOutput,
		)

		return nil
	}

	if isSCCallAfter {
		var callArgs [][]byte
		if len(vmInput.Arguments) > core.MinLenArgumentsDCDTNFTTransfer+1 {
			callArgs = vmInput.Arguments[core.MinLenArgumentsDCDTNFTTransfer+1:]
		}

		addOutputTransferToVMOutput(
			1,
			vmInput.CallerAddr,
			string(vmInput.Arguments[core.MinLenArgumentsDCDTNFTTransfer]),
			callArgs,
			dstAddress,
			vmInput.GasLocked,
			vmInput.CallType,
			vmOutput)
	}

	return nil
}

func (e *dcdtNFTTransfer) addNFTToDestination(
	sndAddress []byte,
	dstAddress []byte,
	userAccount vmcommon.UserAccountHandler,
	dcdtDataToTransfer *dcdt.DCDigitalToken,
	dcdtTokenKey []byte,
	nonce uint64,
	isReturnWithError bool,
) error {
	currentDCDTData, _, err := e.dcdtStorageHandler.GetDCDTNFTTokenOnDestination(userAccount, dcdtTokenKey, nonce)
	if err != nil && !errors.Is(err, ErrNFTTokenDoesNotExist) {
		return err
	}
	err = checkFrozeAndPause(dstAddress, dcdtTokenKey, currentDCDTData, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	transferValue := big.NewInt(0).Set(dcdtDataToTransfer.Value)
	dcdtDataToTransfer.Value.Add(dcdtDataToTransfer.Value, currentDCDTData.Value)
	_, err = e.dcdtStorageHandler.SaveDCDTNFTToken(sndAddress, userAccount, dcdtTokenKey, nonce, dcdtDataToTransfer, false, isReturnWithError)
	if err != nil {
		return err
	}

	isSameShard := e.shardCoordinator.SameShard(sndAddress, dstAddress)
	if !isSameShard {
		err = e.dcdtStorageHandler.AddToLiquiditySystemAcc(dcdtTokenKey, nonce, transferValue)
		if err != nil {
			return err
		}
	}

	return nil
}

func addNFTTransferToVMOutput(
	index uint32,
	senderAddress []byte,
	recipient []byte,
	funcToCall string,
	arguments [][]byte,
	gasLocked uint64,
	gasLimit uint64,
	callType vm.CallType,
	vmOutput *vmcommon.VMOutput,
) {
	nftTransferTxData := funcToCall
	for _, arg := range arguments {
		nftTransferTxData += "@" + hex.EncodeToString(arg)
	}
	outTransfer := vmcommon.OutputTransfer{
		Index:         index,
		Value:         big.NewInt(0),
		GasLimit:      gasLimit,
		GasLocked:     gasLocked,
		Data:          []byte(nftTransferTxData),
		CallType:      callType,
		SenderAddress: senderAddress,
	}
	vmOutput.OutputAccounts = make(map[string]*vmcommon.OutputAccount)
	vmOutput.OutputAccounts[string(recipient)] = &vmcommon.OutputAccount{
		Address:         recipient,
		OutputTransfers: []vmcommon.OutputTransfer{outTransfer},
	}
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dcdtNFTTransfer) IsInterfaceNil() bool {
	return e == nil
}
