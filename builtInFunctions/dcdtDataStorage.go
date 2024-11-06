package builtInFunctions

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/core/check"
	"github.com/kalyan3104/k-chain-core-go/data"
	"github.com/kalyan3104/k-chain-core-go/data/dcdt"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
	"github.com/kalyan3104/k-chain-vm-common-go/parsers"
)

const existsOnShard = byte(1)

type queryOptions struct {
	isCustomSystemAccountSet bool
	customSystemAccount      vmcommon.UserAccountHandler
}

func defaultQueryOptions() queryOptions {
	return queryOptions{}
}

type dcdtDataStorage struct {
	accounts              vmcommon.AccountsAdapter
	globalSettingsHandler vmcommon.DCDTGlobalSettingsHandler
	marshaller            vmcommon.Marshalizer
	keyPrefix             []byte
	shardCoordinator      vmcommon.Coordinator
	txDataParser          vmcommon.CallArgsParser
	enableEpochsHandler   vmcommon.EnableEpochsHandler
}

// ArgsNewDCDTDataStorage defines the argument list for new dcdt data storage handler
type ArgsNewDCDTDataStorage struct {
	Accounts              vmcommon.AccountsAdapter
	GlobalSettingsHandler vmcommon.DCDTGlobalSettingsHandler
	Marshalizer           vmcommon.Marshalizer
	EnableEpochsHandler   vmcommon.EnableEpochsHandler
	ShardCoordinator      vmcommon.Coordinator
}

// NewDCDTDataStorage creates a new dcdt data storage handler
func NewDCDTDataStorage(args ArgsNewDCDTDataStorage) (*dcdtDataStorage, error) {
	if check.IfNil(args.Accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(args.GlobalSettingsHandler) {
		return nil, ErrNilGlobalSettingsHandler
	}
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.EnableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, ErrNilShardCoordinator
	}

	e := &dcdtDataStorage{
		accounts:              args.Accounts,
		globalSettingsHandler: args.GlobalSettingsHandler,
		marshaller:            args.Marshalizer,
		keyPrefix:             []byte(baseDCDTKeyPrefix),
		shardCoordinator:      args.ShardCoordinator,
		txDataParser:          parsers.NewCallArgsParser(),
		enableEpochsHandler:   args.EnableEpochsHandler,
	}

	return e, nil
}

// GetDCDTNFTTokenOnSender gets the nft token on sender account
func (e *dcdtDataStorage) GetDCDTNFTTokenOnSender(
	accnt vmcommon.UserAccountHandler,
	dcdtTokenKey []byte,
	nonce uint64,
) (*dcdt.DCDigitalToken, error) {
	dcdtData, isNew, err := e.GetDCDTNFTTokenOnDestination(accnt, dcdtTokenKey, nonce)
	if err != nil {
		return nil, err
	}
	if isNew {
		return nil, ErrNewNFTDataOnSenderAddress
	}

	return dcdtData, nil
}

// GetDCDTNFTTokenOnDestination gets the nft token on destination account
func (e *dcdtDataStorage) GetDCDTNFTTokenOnDestination(
	accnt vmcommon.UserAccountHandler,
	dcdtTokenKey []byte,
	nonce uint64,
) (*dcdt.DCDigitalToken, bool, error) {
	return e.getDCDTNFTTokenOnDestinationWithAccountsAdapterOptions(accnt, dcdtTokenKey, nonce, defaultQueryOptions())
}

// GetDCDTNFTTokenOnDestinationWithCustomSystemAccount gets the nft token on destination account by using a custom system account
func (e *dcdtDataStorage) GetDCDTNFTTokenOnDestinationWithCustomSystemAccount(
	accnt vmcommon.UserAccountHandler,
	dcdtTokenKey []byte,
	nonce uint64,
	customSystemAccount vmcommon.UserAccountHandler,
) (*dcdt.DCDigitalToken, bool, error) {
	if check.IfNil(customSystemAccount) {
		return nil, false, ErrNilUserAccount
	}

	queryOpts := queryOptions{
		isCustomSystemAccountSet: true,
		customSystemAccount:      customSystemAccount,
	}

	return e.getDCDTNFTTokenOnDestinationWithAccountsAdapterOptions(accnt, dcdtTokenKey, nonce, queryOpts)
}

func (e *dcdtDataStorage) getDCDTNFTTokenOnDestinationWithAccountsAdapterOptions(
	accnt vmcommon.UserAccountHandler,
	dcdtTokenKey []byte,
	nonce uint64,
	options queryOptions,
) (*dcdt.DCDigitalToken, bool, error) {
	dcdtNFTTokenKey := computeDCDTNFTTokenKey(dcdtTokenKey, nonce)
	dcdtData := &dcdt.DCDigitalToken{
		Value: big.NewInt(0),
		Type:  uint32(core.Fungible),
	}
	marshaledData, _, err := accnt.AccountDataHandler().RetrieveValue(dcdtNFTTokenKey)
	if core.IsGetNodeFromDBError(err) {
		return nil, false, err
	}
	if err != nil || len(marshaledData) == 0 {
		return dcdtData, true, nil
	}

	err = e.marshaller.Unmarshal(dcdtData, marshaledData)
	if err != nil {
		return nil, false, err
	}

	if !e.enableEpochsHandler.IsFlagEnabled(SaveToSystemAccountFlag) || nonce == 0 {
		return dcdtData, false, nil
	}

	dcdtMetaData, err := e.getDCDTMetaDataFromSystemAccount(dcdtNFTTokenKey, options)
	if err != nil {
		return nil, false, err
	}
	if dcdtMetaData != nil {
		dcdtData.TokenMetaData = dcdtMetaData
	}

	return dcdtData, false, nil
}

func (e *dcdtDataStorage) getDCDTDigitalTokenDataFromSystemAccount(
	tokenKey []byte,
	options queryOptions,
) (*dcdt.DCDigitalToken, vmcommon.UserAccountHandler, error) {
	systemAcc, err := e.getSystemAccount(options)
	if err != nil {
		return nil, nil, err
	}

	marshaledData, _, err := systemAcc.AccountDataHandler().RetrieveValue(tokenKey)
	if core.IsGetNodeFromDBError(err) {
		return nil, systemAcc, err
	}
	if err != nil || len(marshaledData) == 0 {
		return nil, systemAcc, nil
	}

	dcdtData := &dcdt.DCDigitalToken{}
	err = e.marshaller.Unmarshal(dcdtData, marshaledData)
	if err != nil {
		return nil, nil, err
	}

	return dcdtData, systemAcc, nil
}

func (e *dcdtDataStorage) getDCDTMetaDataFromSystemAccount(
	tokenKey []byte,
	options queryOptions,
) (*dcdt.MetaData, error) {
	dcdtData, _, err := e.getDCDTDigitalTokenDataFromSystemAccount(tokenKey, options)
	if err != nil {
		return nil, err
	}
	if dcdtData == nil {
		return nil, nil
	}

	return dcdtData.TokenMetaData, nil
}

// CheckCollectionIsFrozenForAccount returns
func (e *dcdtDataStorage) checkCollectionIsFrozenForAccount(
	accnt vmcommon.UserAccountHandler,
	dcdtTokenKey []byte,
	nonce uint64,
	isReturnWithError bool,
) error {
	if !e.enableEpochsHandler.IsFlagEnabled(CheckFrozenCollectionFlag) {
		return nil
	}
	if nonce == 0 || isReturnWithError {
		return nil
	}

	dcdtData := &dcdt.DCDigitalToken{
		Value: big.NewInt(0),
		Type:  uint32(core.Fungible),
	}
	marshaledData, _, err := accnt.AccountDataHandler().RetrieveValue(dcdtTokenKey)
	if core.IsGetNodeFromDBError(err) {
		return err
	}
	if err != nil || len(marshaledData) == 0 {
		return nil
	}

	err = e.marshaller.Unmarshal(dcdtData, marshaledData)
	if err != nil {
		return err
	}

	dcdtUserMetaData := DCDTUserMetadataFromBytes(dcdtData.Properties)
	if dcdtUserMetaData.Frozen {
		return ErrDCDTIsFrozenForAccount
	}

	return nil
}

func (e *dcdtDataStorage) checkFrozenPauseProperties(
	acnt vmcommon.UserAccountHandler,
	dcdtTokenKey []byte,
	nonce uint64,
	dcdtData *dcdt.DCDigitalToken,
	isReturnWithError bool,
) error {
	err := checkFrozeAndPause(acnt.AddressBytes(), dcdtTokenKey, dcdtData, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	dcdtNFTTokenKey := computeDCDTNFTTokenKey(dcdtTokenKey, nonce)
	err = checkFrozeAndPause(acnt.AddressBytes(), dcdtNFTTokenKey, dcdtData, e.globalSettingsHandler, isReturnWithError)
	if err != nil {
		return err
	}

	err = e.checkCollectionIsFrozenForAccount(acnt, dcdtTokenKey, nonce, isReturnWithError)
	if err != nil {
		return err
	}

	return nil
}

// AddToLiquiditySystemAcc will increase/decrease the liquidity for DCDT Tokens on the metadata
func (e *dcdtDataStorage) AddToLiquiditySystemAcc(
	dcdtTokenKey []byte,
	nonce uint64,
	transferValue *big.Int,
) error {
	isSaveToSystemAccountFlagEnabled := e.enableEpochsHandler.IsFlagEnabled(SaveToSystemAccountFlag)
	isSendAlwaysFlagEnabled := e.enableEpochsHandler.IsFlagEnabled(SendAlwaysFlag)
	if !isSaveToSystemAccountFlagEnabled || !isSendAlwaysFlagEnabled || nonce == 0 {
		return nil
	}

	dcdtNFTTokenKey := computeDCDTNFTTokenKey(dcdtTokenKey, nonce)
	dcdtData, systemAcc, err := e.getDCDTDigitalTokenDataFromSystemAccount(dcdtNFTTokenKey, defaultQueryOptions())
	if err != nil {
		return err
	}

	if dcdtData == nil {
		return ErrNilDCDTData
	}

	// old style metaData - nothing to do
	if len(dcdtData.Reserved) == 0 {
		return nil
	}

	if e.enableEpochsHandler.IsFlagEnabled(FixOldTokenLiquidityFlag) {
		// old tokens which were transferred intra shard before the activation of this flag
		if dcdtData.Value.Cmp(zero) == 0 && transferValue.Cmp(zero) < 0 {
			dcdtData.Reserved = nil
			return e.marshalAndSaveData(systemAcc, dcdtData, dcdtNFTTokenKey)
		}
	}

	dcdtData.Value.Add(dcdtData.Value, transferValue)
	if dcdtData.Value.Cmp(zero) < 0 {
		return ErrInvalidLiquidityForDCDT
	}

	if dcdtData.Value.Cmp(zero) == 0 {
		err = systemAcc.AccountDataHandler().SaveKeyValue(dcdtNFTTokenKey, nil)
		if err != nil {
			return err
		}

		return e.accounts.SaveAccount(systemAcc)
	}

	err = e.marshalAndSaveData(systemAcc, dcdtData, dcdtNFTTokenKey)
	if err != nil {
		return err
	}

	return nil
}

// SaveDCDTNFTToken saves the nft token to the account and system account
func (e *dcdtDataStorage) SaveDCDTNFTToken(
	senderAddress []byte,
	acnt vmcommon.UserAccountHandler,
	dcdtTokenKey []byte,
	nonce uint64,
	dcdtData *dcdt.DCDigitalToken,
	mustUpdateAllFields bool,
	isReturnWithError bool,
) ([]byte, error) {
	err := e.checkFrozenPauseProperties(acnt, dcdtTokenKey, nonce, dcdtData, isReturnWithError)
	if err != nil {
		return nil, err
	}

	dcdtNFTTokenKey := computeDCDTNFTTokenKey(dcdtTokenKey, nonce)
	senderShardID := e.shardCoordinator.ComputeId(senderAddress)
	if e.enableEpochsHandler.IsFlagEnabled(SaveToSystemAccountFlag) {
		err = e.saveDCDTMetaDataToSystemAccount(acnt, senderShardID, dcdtNFTTokenKey, nonce, dcdtData, mustUpdateAllFields)
		if err != nil {
			return nil, err
		}
	}

	if dcdtData.Value.Cmp(zero) <= 0 {
		return nil, acnt.AccountDataHandler().SaveKeyValue(dcdtNFTTokenKey, nil)
	}

	if !e.enableEpochsHandler.IsFlagEnabled(SaveToSystemAccountFlag) {
		marshaledData, errMarshal := e.marshaller.Marshal(dcdtData)
		if errMarshal != nil {
			return nil, errMarshal
		}

		return marshaledData, acnt.AccountDataHandler().SaveKeyValue(dcdtNFTTokenKey, marshaledData)
	}

	dcdtDataOnAccount := &dcdt.DCDigitalToken{
		Type:       dcdtData.Type,
		Value:      dcdtData.Value,
		Properties: dcdtData.Properties,
	}
	marshaledData, err := e.marshaller.Marshal(dcdtDataOnAccount)
	if err != nil {
		return nil, err
	}

	return marshaledData, acnt.AccountDataHandler().SaveKeyValue(dcdtNFTTokenKey, marshaledData)
}

func (e *dcdtDataStorage) saveDCDTMetaDataToSystemAccount(
	userAcc vmcommon.UserAccountHandler,
	senderShardID uint32,
	dcdtNFTTokenKey []byte,
	nonce uint64,
	dcdtData *dcdt.DCDigitalToken,
	mustUpdateAllFields bool,
) error {
	if nonce == 0 {
		return nil
	}
	if dcdtData.TokenMetaData == nil {
		return nil
	}

	systemAcc, err := e.getSystemAccount(defaultQueryOptions())
	if err != nil {
		return err
	}

	currentSaveData, _, err := systemAcc.AccountDataHandler().RetrieveValue(dcdtNFTTokenKey)
	if core.IsGetNodeFromDBError(err) {
		return err
	}
	err = e.saveMetadataIfRequired(dcdtNFTTokenKey, systemAcc, currentSaveData, dcdtData)
	if err != nil {
		return err
	}

	if !mustUpdateAllFields && len(currentSaveData) > 0 {
		return nil
	}

	dcdtDataOnSystemAcc := &dcdt.DCDigitalToken{
		Type:          dcdtData.Type,
		Value:         big.NewInt(0),
		TokenMetaData: dcdtData.TokenMetaData,
		Properties:    make([]byte, e.shardCoordinator.NumberOfShards()),
	}
	isSendAlwaysFlagEnabled := e.enableEpochsHandler.IsFlagEnabled(SendAlwaysFlag)
	if len(currentSaveData) == 0 && isSendAlwaysFlagEnabled {
		dcdtDataOnSystemAcc.Properties = nil
		dcdtDataOnSystemAcc.Reserved = []byte{1}

		err = e.setReservedToNilForOldToken(dcdtDataOnSystemAcc, userAcc, dcdtNFTTokenKey)
		if err != nil {
			return err
		}
	}

	if !isSendAlwaysFlagEnabled {
		selfID := e.shardCoordinator.SelfId()
		if selfID != core.MetachainShardId {
			dcdtDataOnSystemAcc.Properties[selfID] = existsOnShard
		}
		if senderShardID != core.MetachainShardId {
			dcdtDataOnSystemAcc.Properties[senderShardID] = existsOnShard
		}
	}

	return e.marshalAndSaveData(systemAcc, dcdtDataOnSystemAcc, dcdtNFTTokenKey)
}

func (e *dcdtDataStorage) saveMetadataIfRequired(
	dcdtNFTTokenKey []byte,
	systemAcc vmcommon.UserAccountHandler,
	currentSaveData []byte,
	dcdtData *dcdt.DCDigitalToken,
) error {
	if !e.enableEpochsHandler.IsFlagEnabled(AlwaysSaveTokenMetaDataFlag) {
		return nil
	}
	if !e.enableEpochsHandler.IsFlagEnabled(SendAlwaysFlag) {
		// do not re-write the metadata if it is not sent, as it will cause data loss
		return nil
	}
	if len(currentSaveData) == 0 {
		// optimization: do not try to write here the token metadata, it will be written automatically by the next step
		return nil
	}

	dcdtDataOnSystemAcc := &dcdt.DCDigitalToken{}
	err := e.marshaller.Unmarshal(dcdtDataOnSystemAcc, currentSaveData)
	if err != nil {
		return err
	}
	if len(dcdtDataOnSystemAcc.Reserved) > 0 {
		return nil
	}

	dcdtDataOnSystemAcc.TokenMetaData = dcdtData.TokenMetaData
	return e.marshalAndSaveData(systemAcc, dcdtDataOnSystemAcc, dcdtNFTTokenKey)
}

func (e *dcdtDataStorage) setReservedToNilForOldToken(
	dcdtDataOnSystemAcc *dcdt.DCDigitalToken,
	userAcc vmcommon.UserAccountHandler,
	dcdtNFTTokenKey []byte,
) error {
	if !e.enableEpochsHandler.IsFlagEnabled(FixOldTokenLiquidityFlag) {
		return nil
	}

	if check.IfNil(userAcc) {
		return ErrNilUserAccount
	}
	dataOnUserAcc, _, errNotCritical := userAcc.AccountDataHandler().RetrieveValue(dcdtNFTTokenKey)
	if core.IsGetNodeFromDBError(errNotCritical) {
		return errNotCritical
	}
	shouldIgnoreToken := errNotCritical != nil || len(dataOnUserAcc) == 0
	if shouldIgnoreToken {
		return nil
	}

	dcdtDataOnUserAcc := &dcdt.DCDigitalToken{}
	err := e.marshaller.Unmarshal(dcdtDataOnUserAcc, dataOnUserAcc)
	if err != nil {
		return err
	}

	// tokens which were last moved before flagOptimizeNFTStore keep the dcdt metaData on the user account
	// these are not compatible with the new liquidity model,so we set the reserved field to nil
	if dcdtDataOnUserAcc.TokenMetaData != nil {
		dcdtDataOnSystemAcc.Reserved = nil
	}

	return nil
}

func (e *dcdtDataStorage) marshalAndSaveData(
	systemAcc vmcommon.UserAccountHandler,
	dcdtData *dcdt.DCDigitalToken,
	dcdtNFTTokenKey []byte,
) error {
	marshaledData, err := e.marshaller.Marshal(dcdtData)
	if err != nil {
		return err
	}

	err = systemAcc.AccountDataHandler().SaveKeyValue(dcdtNFTTokenKey, marshaledData)
	if err != nil {
		return err
	}

	return e.accounts.SaveAccount(systemAcc)
}

func (e *dcdtDataStorage) getSystemAccount(options queryOptions) (vmcommon.UserAccountHandler, error) {
	if options.isCustomSystemAccountSet && !check.IfNil(options.customSystemAccount) {
		return options.customSystemAccount, nil
	}

	return e.loadSystemAccount()
}

func (e *dcdtDataStorage) loadSystemAccount() (vmcommon.UserAccountHandler, error) {
	systemSCAccount, err := e.accounts.LoadAccount(vmcommon.SystemAccountAddress)
	if err != nil {
		return nil, err
	}

	userAcc, ok := systemSCAccount.(vmcommon.UserAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return userAcc, nil
}

//TODO: merge properties in case of shard merge

// WasAlreadySentToDestinationShardAndUpdateState checks whether NFT metadata was sent to destination shard or not
// and saves the destination shard as sent
func (e *dcdtDataStorage) WasAlreadySentToDestinationShardAndUpdateState(
	tickerID []byte,
	nonce uint64,
	dstAddress []byte,
) (bool, error) {
	if !e.enableEpochsHandler.IsFlagEnabled(SaveToSystemAccountFlag) {
		return false, nil
	}

	if nonce == 0 {
		return true, nil
	}
	dstShardID := e.shardCoordinator.ComputeId(dstAddress)
	if dstShardID == e.shardCoordinator.SelfId() {
		return true, nil
	}

	if e.enableEpochsHandler.IsFlagEnabled(SendAlwaysFlag) {
		return false, nil
	}

	if dstShardID == core.MetachainShardId {
		return true, nil
	}
	dcdtTokenKey := append(e.keyPrefix, tickerID...)
	dcdtNFTTokenKey := computeDCDTNFTTokenKey(dcdtTokenKey, nonce)

	dcdtData, systemAcc, err := e.getDCDTDigitalTokenDataFromSystemAccount(dcdtNFTTokenKey, defaultQueryOptions())
	if err != nil {
		return false, err
	}
	if dcdtData == nil {
		return false, nil
	}

	if uint32(len(dcdtData.Properties)) < e.shardCoordinator.NumberOfShards() {
		newSlice := make([]byte, e.shardCoordinator.NumberOfShards())
		copy(newSlice, dcdtData.Properties)
		dcdtData.Properties = newSlice
	}

	if dcdtData.Properties[dstShardID] > 0 {
		return true, nil
	}

	dcdtData.Properties[dstShardID] = existsOnShard
	return false, e.marshalAndSaveData(systemAcc, dcdtData, dcdtNFTTokenKey)
}

// SaveNFTMetaDataToSystemAccount this saves the NFT metadata to the system account even if there was an error in processing
func (e *dcdtDataStorage) SaveNFTMetaDataToSystemAccount(
	tx data.TransactionHandler,
) error {
	if !e.enableEpochsHandler.IsFlagEnabled(SaveToSystemAccountFlag) {
		return nil
	}
	if e.enableEpochsHandler.IsFlagEnabled(SendAlwaysFlag) {
		return nil
	}
	if check.IfNil(tx) {
		return ErrNilTransactionHandler
	}

	sndShardID := e.shardCoordinator.ComputeId(tx.GetSndAddr())
	dstShardID := e.shardCoordinator.ComputeId(tx.GetRcvAddr())
	isCrossShardTxAtDest := sndShardID != dstShardID && e.shardCoordinator.SelfId() == dstShardID
	if !isCrossShardTxAtDest {
		return nil
	}

	function, arguments, err := e.txDataParser.ParseData(string(tx.GetData()))
	if err != nil {
		return nil
	}
	if len(arguments) < 4 {
		return nil
	}

	switch function {
	case core.BuiltInFunctionDCDTNFTTransfer:
		return e.addMetaDataToSystemAccountFromNFTTransfer(sndShardID, arguments)
	case core.BuiltInFunctionMultiDCDTNFTTransfer:
		return e.addMetaDataToSystemAccountFromMultiTransfer(sndShardID, arguments)
	default:
		return nil
	}
}

func (e *dcdtDataStorage) addMetaDataToSystemAccountFromNFTTransfer(
	sndShardID uint32,
	arguments [][]byte,
) error {
	if !bytes.Equal(arguments[3], zeroByteArray) {
		dcdtTransferData := &dcdt.DCDigitalToken{}
		err := e.marshaller.Unmarshal(dcdtTransferData, arguments[3])
		if err != nil {
			return err
		}
		dcdtTokenKey := append(e.keyPrefix, arguments[0]...)
		nonce := big.NewInt(0).SetBytes(arguments[1]).Uint64()
		dcdtNFTTokenKey := computeDCDTNFTTokenKey(dcdtTokenKey, nonce)

		return e.saveDCDTMetaDataToSystemAccount(nil, sndShardID, dcdtNFTTokenKey, nonce, dcdtTransferData, true)
	}
	return nil
}

func (e *dcdtDataStorage) addMetaDataToSystemAccountFromMultiTransfer(
	sndShardID uint32,
	arguments [][]byte,
) error {
	numOfTransfers := big.NewInt(0).SetBytes(arguments[0]).Uint64()
	if numOfTransfers == 0 {
		return fmt.Errorf("%w, 0 tokens to transfer", ErrInvalidArguments)
	}
	minNumOfArguments := numOfTransfers*argumentsPerTransfer + 1
	if uint64(len(arguments)) < minNumOfArguments {
		return fmt.Errorf("%w, invalid number of arguments", ErrInvalidArguments)
	}

	startIndex := uint64(1)
	for i := uint64(0); i < numOfTransfers; i++ {
		tokenStartIndex := startIndex + i*argumentsPerTransfer
		tokenID := arguments[tokenStartIndex]
		nonce := big.NewInt(0).SetBytes(arguments[tokenStartIndex+1]).Uint64()

		if nonce > 0 && len(arguments[tokenStartIndex+2]) > vmcommon.MaxLengthForValueToOptTransfer {
			dcdtTransferData := &dcdt.DCDigitalToken{}
			marshaledNFTTransfer := arguments[tokenStartIndex+2]
			err := e.marshaller.Unmarshal(dcdtTransferData, marshaledNFTTransfer)
			if err != nil {
				return fmt.Errorf("%w for token %s", err, string(tokenID))
			}

			dcdtTokenKey := append(e.keyPrefix, tokenID...)
			dcdtNFTTokenKey := computeDCDTNFTTokenKey(dcdtTokenKey, nonce)
			err = e.saveDCDTMetaDataToSystemAccount(nil, sndShardID, dcdtNFTTokenKey, nonce, dcdtTransferData, true)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dcdtDataStorage) IsInterfaceNil() bool {
	return e == nil
}
