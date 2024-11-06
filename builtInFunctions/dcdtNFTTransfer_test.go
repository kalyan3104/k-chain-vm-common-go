package builtInFunctions

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/DharitriOne/drt-chain-core-go/core"
	"github.com/DharitriOne/drt-chain-core-go/core/check"
	"github.com/DharitriOne/drt-chain-core-go/data/dcdt"
	vmcommon "github.com/DharitriOne/drt-chain-vm-common-go"
	"github.com/DharitriOne/drt-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var keyPrefix = []byte(baseDCDTKeyPrefix)

func createNftTransferWithStubArguments() *dcdtNFTTransfer {
	nftTransfer, _ := NewDCDTNFTTransferFunc(
		0,
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.AccountsStub{},
		&mock.ShardCoordinatorStub{},
		vmcommon.BaseOperationCost{},
		&mock.DCDTRoleHandlerStub{},
		createNewDCDTDataStorageHandler(),
		&mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == SaveToSystemAccountFlag || flag == CheckCorrectTokenIDForTransferRoleFlag
			},
		},
	)

	return nftTransfer
}

func createNFTTransferAndStorageHandler(selfShard, numShards uint32, globalSettingsHandler vmcommon.ExtendedDCDTGlobalSettingsHandler, enableEpochsHandler vmcommon.EnableEpochsHandler) (*dcdtNFTTransfer, *dcdtDataStorage) {
	marshaller := &mock.MarshalizerMock{}
	shardCoordinator := mock.NewMultiShardsCoordinatorMock(numShards)
	shardCoordinator.CurrentShard = selfShard
	shardCoordinator.ComputeIdCalled = func(address []byte) uint32 {
		lastByte := uint32(address[len(address)-1])
		return lastByte
	}
	mapAccounts := make(map[string]vmcommon.UserAccountHandler)
	accounts := &mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			_, ok := mapAccounts[string(address)]
			if !ok {
				mapAccounts[string(address)] = mock.NewUserAccount(address)
			}
			return mapAccounts[string(address)], nil
		},
		GetExistingAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			_, ok := mapAccounts[string(address)]
			if !ok {
				mapAccounts[string(address)] = mock.NewUserAccount(address)
			}
			return mapAccounts[string(address)], nil
		},
	}

	dcdtStorageHandler := createNewDCDTDataStorageHandlerWithArgs(globalSettingsHandler, accounts, enableEpochsHandler)
	nftTransfer, _ := NewDCDTNFTTransferFunc(
		1,
		marshaller,
		globalSettingsHandler,
		accounts,
		shardCoordinator,
		vmcommon.BaseOperationCost{},
		&mock.DCDTRoleHandlerStub{
			CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
				if bytes.Equal(action, []byte(core.DCDTRoleTransfer)) {
					return ErrActionNotAllowed
				}
				return nil
			},
		},
		dcdtStorageHandler,
		enableEpochsHandler,
	)

	return nftTransfer, dcdtStorageHandler
}

func createNftTransferWithMockArguments(selfShard uint32, numShards uint32, globalSettingsHandler vmcommon.ExtendedDCDTGlobalSettingsHandler) *dcdtNFTTransfer {
	nftTransfer, _ := createNFTTransferAndStorageHandler(selfShard, numShards, globalSettingsHandler, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == CheckTransferFlag || flag == CheckFrozenCollectionFlag
		},
	})
	return nftTransfer
}

func createMockGasCost() vmcommon.GasCost {
	return vmcommon.GasCost{
		BaseOperationCost: vmcommon.BaseOperationCost{
			StorePerByte:      10,
			ReleasePerByte:    20,
			DataCopyPerByte:   30,
			PersistPerByte:    40,
			CompilePerByte:    50,
			AoTPreparePerByte: 60,
		},
		BuiltInCost: vmcommon.BuiltInCost{
			ChangeOwnerAddress:       70,
			ClaimDeveloperRewards:    80,
			SaveUserName:             90,
			SaveKeyValue:             100,
			DCDTTransfer:             110,
			DCDTBurn:                 120,
			DCDTLocalMint:            130,
			DCDTLocalBurn:            140,
			DCDTNFTCreate:            150,
			DCDTNFTAddQuantity:       160,
			DCDTNFTBurn:              170,
			DCDTNFTTransfer:          180,
			DCDTNFTChangeCreateOwner: 190,
			DCDTNFTUpdateAttributes:  200,
			DCDTNFTAddURI:            210,
			DCDTNFTMultiTransfer:     220,
		},
	}
}

func createDCDTNFTToken(
	tokenName []byte,
	nftType core.DCDTType,
	nonce uint64,
	value *big.Int,
	marshaller vmcommon.Marshalizer,
	account vmcommon.UserAccountHandler,
) {
	tokenId := append(keyPrefix, tokenName...)
	dcdtNFTTokenKey := computeDCDTNFTTokenKey(tokenId, nonce)
	dcdtData := &dcdt.DCDigitalToken{
		Type:  uint32(nftType),
		Value: value,
	}

	if nonce > 0 {
		dcdtData.TokenMetaData = &dcdt.MetaData{
			URIs:  [][]byte{[]byte("uri")},
			Nonce: nonce,
			Hash:  []byte("NFT hash"),
		}
	}

	buff, _ := marshaller.Marshal(dcdtData)

	_ = account.AccountDataHandler().SaveKeyValue(dcdtNFTTokenKey, buff)
}

func testNFTTokenShouldExist(
	tb testing.TB,
	marshaller vmcommon.Marshalizer,
	account vmcommon.AccountHandler,
	tokenName []byte,
	nonce uint64,
	expectedValue *big.Int,
) {
	tokenId := append(keyPrefix, tokenName...)
	dcdtNFTTokenKey := computeDCDTNFTTokenKey(tokenId, nonce)
	dcdtData := &dcdt.DCDigitalToken{Value: big.NewInt(0), Type: uint32(core.Fungible)}
	marshaledData, _, _ := account.(vmcommon.UserAccountHandler).AccountDataHandler().RetrieveValue(dcdtNFTTokenKey)
	_ = marshaller.Unmarshal(dcdtData, marshaledData)
	assert.Equal(tb, expectedValue, dcdtData.Value)
}

func TestNewDCDTNFTTransferFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		nftTransfer, err := NewDCDTNFTTransferFunc(
			0,
			nil,
			&mock.GlobalSettingsHandlerStub{},
			&mock.AccountsStub{},
			&mock.ShardCoordinatorStub{},
			vmcommon.BaseOperationCost{},
			&mock.DCDTRoleHandlerStub{},
			createNewDCDTDataStorageHandler(),
			&mock.EnableEpochsHandlerStub{},
		)
		assert.True(t, check.IfNil(nftTransfer))
		assert.Equal(t, ErrNilMarshalizer, err)
	})
	t.Run("nil global settings handler should error", func(t *testing.T) {
		t.Parallel()

		nftTransfer, err := NewDCDTNFTTransferFunc(
			0,
			&mock.MarshalizerMock{},
			nil,
			&mock.AccountsStub{},
			&mock.ShardCoordinatorStub{},
			vmcommon.BaseOperationCost{},
			&mock.DCDTRoleHandlerStub{},
			createNewDCDTDataStorageHandler(),
			&mock.EnableEpochsHandlerStub{},
		)
		assert.True(t, check.IfNil(nftTransfer))
		assert.Equal(t, ErrNilGlobalSettingsHandler, err)
	})
	t.Run("nil accounts adapter should error", func(t *testing.T) {
		t.Parallel()

		nftTransfer, err := NewDCDTNFTTransferFunc(
			0,
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			nil,
			&mock.ShardCoordinatorStub{},
			vmcommon.BaseOperationCost{},
			&mock.DCDTRoleHandlerStub{},
			createNewDCDTDataStorageHandler(),
			&mock.EnableEpochsHandlerStub{},
		)
		assert.True(t, check.IfNil(nftTransfer))
		assert.Equal(t, ErrNilAccountsAdapter, err)
	})
	t.Run("nil shard coordinator should error", func(t *testing.T) {
		t.Parallel()

		nftTransfer, err := NewDCDTNFTTransferFunc(
			0,
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.AccountsStub{},
			nil,
			vmcommon.BaseOperationCost{},
			&mock.DCDTRoleHandlerStub{},
			createNewDCDTDataStorageHandler(),
			&mock.EnableEpochsHandlerStub{},
		)
		assert.True(t, check.IfNil(nftTransfer))
		assert.Equal(t, ErrNilShardCoordinator, err)
	})
	t.Run("nil roles handler should error", func(t *testing.T) {
		t.Parallel()

		nftTransfer, err := NewDCDTNFTTransferFunc(
			0,
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.AccountsStub{},
			&mock.ShardCoordinatorStub{},
			vmcommon.BaseOperationCost{},
			nil,
			createNewDCDTDataStorageHandler(),
			&mock.EnableEpochsHandlerStub{},
		)
		assert.True(t, check.IfNil(nftTransfer))
		assert.Equal(t, ErrNilRolesHandler, err)
	})
	t.Run("nil dcdt storage handler should error", func(t *testing.T) {
		t.Parallel()

		nftTransfer, err := NewDCDTNFTTransferFunc(
			0,
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.AccountsStub{},
			&mock.ShardCoordinatorStub{},
			vmcommon.BaseOperationCost{},
			&mock.DCDTRoleHandlerStub{},
			nil,
			&mock.EnableEpochsHandlerStub{},
		)
		assert.True(t, check.IfNil(nftTransfer))
		assert.Equal(t, ErrNilDCDTNFTStorageHandler, err)
	})
	t.Run("nil enable epochs handler should error", func(t *testing.T) {
		t.Parallel()

		nftTransfer, err := NewDCDTNFTTransferFunc(
			0,
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.AccountsStub{},
			&mock.ShardCoordinatorStub{},
			vmcommon.BaseOperationCost{},
			&mock.DCDTRoleHandlerStub{},
			createNewDCDTDataStorageHandler(),
			nil,
		)
		assert.True(t, check.IfNil(nftTransfer))
		assert.Equal(t, ErrNilEnableEpochsHandler, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		nftTransfer, err := NewDCDTNFTTransferFunc(
			0,
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.AccountsStub{},
			&mock.ShardCoordinatorStub{},
			vmcommon.BaseOperationCost{},
			&mock.DCDTRoleHandlerStub{},
			createNewDCDTDataStorageHandler(),
			&mock.EnableEpochsHandlerStub{},
		)
		assert.False(t, check.IfNil(nftTransfer))
		assert.Nil(t, err)
	})
}

func TestDcdtNFTTransfer_SetPayable(t *testing.T) {
	t.Parallel()

	nftTransfer := createNftTransferWithStubArguments()
	err := nftTransfer.SetPayableChecker(nil)
	assert.Equal(t, ErrNilPayableHandler, err)

	handler := &mock.PayableHandlerStub{}
	err = nftTransfer.SetPayableChecker(handler)
	assert.Nil(t, err)
	assert.True(t, handler == nftTransfer.payableHandler) // pointer testing
}

func TestDcdtNFTTransfer_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	nftTransfer := createNftTransferWithStubArguments()
	nftTransfer.SetNewGasConfig(nil)
	assert.Equal(t, uint64(0), nftTransfer.funcGasCost)
	assert.Equal(t, vmcommon.BaseOperationCost{}, nftTransfer.gasConfig)

	gasCost := createMockGasCost()
	nftTransfer.SetNewGasConfig(&gasCost)
	assert.Equal(t, gasCost.BuiltInCost.DCDTNFTTransfer, nftTransfer.funcGasCost)
	assert.Equal(t, gasCost.BaseOperationCost, nftTransfer.gasConfig)
}

func TestDcdtNFTTransfer_ProcessBuiltinFunctionInvalidArgumentsShouldErr(t *testing.T) {
	t.Parallel()

	nftTransfer := createNftTransferWithStubArguments()
	vmOutput, err := nftTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, nil)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNilVmInput, err)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("arg1"), []byte("arg2")},
		},
	}
	vmOutput, err = nftTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidArguments, err)

	nftTransfer.shardCoordinator = &mock.ShardCoordinatorStub{ComputeIdCalled: func(address []byte) uint32 {
		return core.MetachainShardId
	}}

	tokenName := []byte("token")
	senderAddress := bytes.Repeat([]byte{2}, 32)
	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, big.NewInt(1).Bytes(), big.NewInt(1).Bytes(), core.DCDTSCAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}
	vmOutput, err = nftTransfer.ProcessBuiltinFunction(&mock.UserAccountStub{}, &mock.UserAccountStub{}, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidRcvAddr, err)
}

func TestDcdtNFTTransfer_SenderDoesNotHaveNFT(t *testing.T) {
	t.Parallel()

	nftTransfer := createNftTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})
	_ = nftTransfer.SetPayableChecker(
		&mock.PayableHandlerStub{
			IsPayableCalled: func(address []byte) (bool, error) {
				return true, nil
			},
		})
	senderAddress := bytes.Repeat([]byte{2}, 32)
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := nftTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err := nftTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(0)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}

	_, err = nftTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	require.Equal(t, err, ErrNewNFTDataOnSenderAddress)
}

func TestDcdtNFTTransfer_ProcessWithZeroValue(t *testing.T) {
	t.Parallel()

	nftTransfer := createNftTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32)
	destinationAddress := bytes.Repeat([]byte{1}, 32)

	sender, err := nftTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err := nftTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCDTNFTToken(tokenName, core.NonFungible, tokenNonce, initialTokens, nftTransfer.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = nftTransfer.accounts.SaveAccount(sender)
	_ = nftTransfer.accounts.SaveAccount(destination)
	_, _ = nftTransfer.accounts.Commit()

	// reload accounts
	sender, err = nftTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err = nftTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(0).Bytes()
	scCallArgs := [][]byte{[]byte(scCallFunctionAsHex), []byte(scCallArg)}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	_, err = nftTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	require.Equal(t, err, ErrInvalidNFTQuantity)
}

func TestDcdtNFTTransfer_TransferValueLengthChecks(t *testing.T) {
	t.Parallel()

	nftTransfer := createNftTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32)
	destinationAddress := bytes.Repeat([]byte{1}, 32)

	sender, err := nftTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err := nftTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCDTNFTToken(tokenName, core.NonFungible, tokenNonce, initialTokens, nftTransfer.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = nftTransfer.accounts.SaveAccount(sender)
	_ = nftTransfer.accounts.SaveAccount(destination)
	_, _ = nftTransfer.accounts.Commit()

	// reload accounts
	sender, err = nftTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err = nftTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantity, _ := big.NewInt(0).SetString("1"+strings.Repeat("0", 250), 10)
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, nonceBytes, quantity.Bytes(), destinationAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}

	// before flag activation
	_, err = nftTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	require.Equal(t, err, ErrInvalidNFTQuantity)

	// after flag activation
	nftTransfer.enableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ConsistentTokensValuesLengthCheckFlag
		},
	}
	_, err = nftTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	require.Contains(t, err.Error(), "max length for a transfer value is")
}

func TestDcdtNFTTransfer_ProcessBuiltinFunctionOnSameShardWithScCall(t *testing.T) {
	t.Parallel()

	nftTransfer := createNftTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})

	payableChecker, _ := NewPayableCheckFunc(
		&mock.PayableHandlerStub{
			IsPayableCalled: func(address []byte) (bool, error) {
				return true, nil
			},
		}, &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == FixAsyncCallbackCheckFlag || flag == CheckFunctionArgumentFlag
			},
		})

	_ = nftTransfer.SetPayableChecker(payableChecker)
	senderAddress := bytes.Repeat([]byte{2}, 32)
	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1
	sender, err := nftTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err := nftTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCDTNFTToken(tokenName, core.NonFungible, tokenNonce, initialTokens, nftTransfer.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = nftTransfer.accounts.SaveAccount(sender)
	_ = nftTransfer.accounts.SaveAccount(destination)
	_, _ = nftTransfer.accounts.Commit()

	// reload accounts
	sender, err = nftTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err = nftTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	scCallArgs := [][]byte{[]byte(scCallFunctionAsHex), []byte(scCallArg)}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	vmOutput, err := nftTransfer.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = nftTransfer.accounts.SaveAccount(sender)
	_, _ = nftTransfer.accounts.Commit()

	// reload accounts
	sender, err = nftTransfer.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)
	destination, err = nftTransfer.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, nftTransfer.marshaller, sender, tokenName, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred
	testNFTTokenShouldExist(t, nftTransfer.marshaller, destination, tokenName, tokenNonce, big.NewInt(1))
	funcName, args := extractScResultsFromVmOutput(t, vmOutput)
	assert.Equal(t, scCallFunctionAsHex, funcName)
	require.Equal(t, 1, len(args))
	require.Equal(t, []byte(scCallArg), args[0])
}

func TestDcdtNFTTransfer_ProcessBuiltinFunctionOnCrossShardsDestinationDoesNotHoldingNFTWithSCCall(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	nftTransferSenderShard := createNftTransferWithMockArguments(1, 2, &mock.GlobalSettingsHandlerStub{})
	_ = nftTransferSenderShard.SetPayableChecker(payableHandler)

	nftTransferDestinationShard := createNftTransferWithMockArguments(0, 2, &mock.GlobalSettingsHandlerStub{})
	_ = nftTransferDestinationShard.SetPayableChecker(payableHandler)

	senderAddress := bytes.Repeat([]byte{1}, 32)
	destinationAddress := bytes.Repeat([]byte{0}, 32)
	destinationAddress[25] = 1
	sender, err := nftTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCDTNFTToken(tokenName, core.NonFungible, tokenNonce, initialTokens, nftTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = nftTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = nftTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = nftTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	scCallArgs := [][]byte{[]byte(scCallFunctionAsHex), []byte(scCallArg)}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	vmOutput, err := nftTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = nftTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = nftTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = nftTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, nftTransferSenderShard.marshaller, sender, tokenName, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred

	_, args := extractScResultsFromVmOutput(t, vmOutput)

	destination, err := nftTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	vmOutput, err = nftTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	_ = nftTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = nftTransferDestinationShard.accounts.Commit()

	destination, err = nftTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, nftTransferDestinationShard.marshaller, destination, tokenName, tokenNonce, big.NewInt(1))
	funcName, args := extractScResultsFromVmOutput(t, vmOutput)
	assert.Equal(t, scCallFunctionAsHex, funcName)
	require.Equal(t, 1, len(args))
	require.Equal(t, []byte(scCallArg), args[0])
}

func TestDcdtNFTTransfer_ProcessBuiltinFunctionOnCrossShardsDestinationHoldsNFT(t *testing.T) {
	t.Parallel()

	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	nftTransferSenderShard := createNftTransferWithMockArguments(0, 2, &mock.GlobalSettingsHandlerStub{})
	_ = nftTransferSenderShard.SetPayableChecker(payableHandler)

	nftTransferDestinationShard := createNftTransferWithMockArguments(1, 2, &mock.GlobalSettingsHandlerStub{})
	_ = nftTransferDestinationShard.SetPayableChecker(payableHandler)

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := nftTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCDTNFTToken(tokenName, core.NonFungible, tokenNonce, initialTokens, nftTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = nftTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = nftTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = nftTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}

	vmOutput, err := nftTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = nftTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = nftTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = nftTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	testNFTTokenShouldExist(t, nftTransferSenderShard.marshaller, sender, tokenName, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred

	_, args := extractScResultsFromVmOutput(t, vmOutput)

	destinationNumTokens := big.NewInt(1000)
	destination, err := nftTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)
	createDCDTNFTToken(tokenName, core.NonFungible, tokenNonce, destinationNumTokens, nftTransferDestinationShard.marshaller, destination.(vmcommon.UserAccountHandler))
	_ = nftTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = nftTransferDestinationShard.accounts.Commit()

	destination, err = nftTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			CallerAddr: senderAddress,
			Arguments:  args,
		},
		RecipientAddr: destinationAddress,
	}

	vmOutput, err = nftTransferDestinationShard.ProcessBuiltinFunction(nil, destination.(vmcommon.UserAccountHandler), vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)
	_ = nftTransferDestinationShard.accounts.SaveAccount(destination)
	_, _ = nftTransferDestinationShard.accounts.Commit()

	destination, err = nftTransferDestinationShard.accounts.LoadAccount(destinationAddress)
	require.Nil(t, err)

	expected := big.NewInt(0).Add(destinationNumTokens, big.NewInt(1))
	testNFTTokenShouldExist(t, nftTransferDestinationShard.marshaller, destination, tokenName, tokenNonce, expected)
}

func TestDCDTNFTTransfer_SndDstFrozen(t *testing.T) {
	t.Parallel()

	globalSettings := &mock.GlobalSettingsHandlerStub{}
	transferFunc := createNftTransferWithMockArguments(0, 1, globalSettings)
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	destinationAddress[31] = 0
	sender, err := transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCDTNFTToken(tokenName, core.NonFungible, tokenNonce, initialTokens, transferFunc.marshaller, sender.(vmcommon.UserAccountHandler))
	dcdtFrozen := DCDTUserMetadata{Frozen: true}

	_ = transferFunc.accounts.SaveAccount(sender)
	_, _ = transferFunc.accounts.Commit()
	// reload sender account
	sender, err = transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}

	destination, _ := transferFunc.accounts.LoadAccount(destinationAddress)
	tokenId := append(keyPrefix, tokenName...)
	dcdtKey := computeDCDTNFTTokenKey(tokenId, tokenNonce)
	dcdtToken := &dcdt.DCDigitalToken{Value: big.NewInt(0), Properties: dcdtFrozen.ToBytes()}
	marshaledData, _ := transferFunc.marshaller.Marshal(dcdtToken)
	_ = destination.(vmcommon.UserAccountHandler).AccountDataHandler().SaveKeyValue(dcdtKey, marshaledData)
	_ = transferFunc.accounts.SaveAccount(destination)
	_, _ = transferFunc.accounts.Commit()

	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	assert.Equal(t, ErrDCDTIsFrozenForAccount, err)

	vmInput.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	assert.Nil(t, err)
}

func TestDCDTNFTTransfer_WithLimitedTransfer(t *testing.T) {
	t.Parallel()

	globalSettings := &mock.GlobalSettingsHandlerStub{}
	transferFunc := createNftTransferWithMockArguments(0, 1, globalSettings)
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	destinationAddress[31] = 0
	sender, err := transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCDTNFTToken(tokenName, core.NonFungible, tokenNonce, initialTokens, transferFunc.marshaller, sender.(vmcommon.UserAccountHandler))

	_ = transferFunc.accounts.SaveAccount(sender)
	_, _ = transferFunc.accounts.Commit()
	// reload sender account
	sender, err = transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}

	destination, _ := transferFunc.accounts.LoadAccount(destinationAddress)
	globalSettings.IsLimiterTransferCalled = func(token []byte) bool {
		return true
	}
	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	assert.Equal(t, ErrActionNotAllowed, err)

	vmInput.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	assert.Nil(t, err)
}

func TestDCDTNFTTransfer_NotEnoughGas(t *testing.T) {
	t.Parallel()

	transferFunc := createNftTransferWithMockArguments(0, 1, &mock.GlobalSettingsHandlerStub{})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	sender, err := transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCDTNFTToken(tokenName, core.NonFungible, tokenNonce, initialTokens, transferFunc.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = transferFunc.accounts.SaveAccount(sender)
	_, _ = transferFunc.accounts.Commit()
	// reload sender account
	sender, err = transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 0,
		},
		RecipientAddr: senderAddress,
	}

	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), sender.(vmcommon.UserAccountHandler), vmInput)
	assert.Equal(t, err, ErrNotEnoughGas)
}

func extractScResultsFromVmOutput(t testing.TB, vmOutput *vmcommon.VMOutput) (string, [][]byte) {
	require.NotNil(t, vmOutput)
	require.Equal(t, 1, len(vmOutput.OutputAccounts))
	var outputAccount *vmcommon.OutputAccount
	for _, account := range vmOutput.OutputAccounts {
		outputAccount = account
		break
	}
	require.NotNil(t, outputAccount)
	if outputAccount == nil {
		// suppress next warnings, goland does not know about require.NotNil
		return "", nil
	}
	require.Equal(t, 1, len(outputAccount.OutputTransfers))
	outputTransfer := outputAccount.OutputTransfers[0]
	split := strings.Split(string(outputTransfer.Data), "@")

	args := make([][]byte, len(split)-1)
	var err error
	for i, splitArg := range split[1:] {
		args[i], err = hex.DecodeString(splitArg)
		require.Nil(t, err)
	}

	return split[0], args
}

func TestDCDTNFTTransfer_SndDstFreezeCollection(t *testing.T) {
	t.Parallel()

	globalSettings := &mock.GlobalSettingsHandlerStub{}
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == CheckTransferFlag || flag == CheckFrozenCollectionFlag
		},
	}
	transferFunc, _ := createNFTTransferAndStorageHandler(0, 1, globalSettings, enableEpochsHandler)

	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	senderAddress := bytes.Repeat([]byte{2}, 32) // sender is in the same shard
	destinationAddress := bytes.Repeat([]byte{1}, 32)
	destinationAddress[31] = 0
	sender, err := transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCDTNFTToken(tokenName, core.NonFungible, tokenNonce, initialTokens, transferFunc.marshaller, sender.(vmcommon.UserAccountHandler))
	dcdtFrozen := DCDTUserMetadata{Frozen: true}

	_ = transferFunc.accounts.SaveAccount(sender)
	_, _ = transferFunc.accounts.Commit()
	// reload sender account
	sender, err = transferFunc.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}

	destination, _ := transferFunc.accounts.LoadAccount(destinationAddress)
	tokenId := append(keyPrefix, tokenName...)
	dcdtToken := &dcdt.DCDigitalToken{Value: big.NewInt(0), Properties: dcdtFrozen.ToBytes()}
	marshaledData, _ := transferFunc.marshaller.Marshal(dcdtToken)
	_ = destination.(vmcommon.UserAccountHandler).AccountDataHandler().SaveKeyValue(tokenId, marshaledData)
	_ = transferFunc.accounts.SaveAccount(destination)
	_, _ = transferFunc.accounts.Commit()

	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	assert.Equal(t, ErrDCDTIsFrozenForAccount, err)

	vmInput.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), destination.(vmcommon.UserAccountHandler), vmInput)
	assert.Nil(t, err)
}

func TestDcdtNFTTransfer_ProcessBuiltinFunctionCrossShardsFixOldLiquidityIssue(t *testing.T) {
	t.Parallel()

	vmInput, sender, nftTransferSenderShard, dcdtDataStorageHandler, tokenName, tokenNonce := createSetupToSendNFTCrossShard(t)

	dcdtDataStorageHandler.enableEpochsHandler.(*mock.EnableEpochsHandlerStub).IsFlagEnabledCalled = func(flag core.EnableEpochFlag) bool {
		return flag == FixOldTokenLiquidityFlag
	}
	vmOutput, err := nftTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Nil(t, err)
	require.Equal(t, vmcommon.Ok, vmOutput.ReturnCode)

	_ = nftTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = nftTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = nftTransferSenderShard.accounts.LoadAccount(sender.AddressBytes())
	require.Nil(t, err)

	testNFTTokenShouldExist(t, nftTransferSenderShard.marshaller, sender, tokenName, tokenNonce, big.NewInt(2)) // 3 initial - 1 transferred
}

func TestDcdtNFTTransfer_ProcessBuiltinFunctionCrossShardsFixOldLiquidityIssueWithoutActivation(t *testing.T) {
	t.Parallel()

	vmInput, sender, nftTransferSenderShard, _, _, _ := createSetupToSendNFTCrossShard(t)
	_, err := nftTransferSenderShard.ProcessBuiltinFunction(sender.(vmcommon.UserAccountHandler), nil, vmInput)
	require.Equal(t, err, ErrInvalidLiquidityForDCDT)
}

func createSetupToSendNFTCrossShard(t *testing.T) (*vmcommon.ContractCallInput, vmcommon.AccountHandler, *dcdtNFTTransfer, *dcdtDataStorage, []byte, uint64) {
	payableHandler := &mock.PayableHandlerStub{
		IsPayableCalled: func(address []byte) (bool, error) {
			return true, nil
		},
	}

	var enableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == SendAlwaysFlag || flag == SaveToSystemAccountFlag || flag == CheckFrozenCollectionFlag
		},
	}
	nftTransferSenderShard, dcdtDataStorageHandler := createNFTTransferAndStorageHandler(1, 2, &mock.GlobalSettingsHandlerStub{}, enableEpochsHandler)
	_ = nftTransferSenderShard.SetPayableChecker(payableHandler)

	senderAddress := bytes.Repeat([]byte{1}, 32)
	destinationAddress := bytes.Repeat([]byte{2}, 32)
	sender, err := nftTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	tokenName := []byte("token")
	tokenNonce := uint64(1)

	initialTokens := big.NewInt(3)
	createDCDTNFTToken(tokenName, core.NonFungible, tokenNonce, initialTokens, nftTransferSenderShard.marshaller, sender.(vmcommon.UserAccountHandler))
	_ = nftTransferSenderShard.accounts.SaveAccount(sender)
	_, _ = nftTransferSenderShard.accounts.Commit()

	// reload sender account
	sender, err = nftTransferSenderShard.accounts.LoadAccount(senderAddress)
	require.Nil(t, err)

	nonceBytes := big.NewInt(int64(tokenNonce)).Bytes()
	quantityBytes := big.NewInt(1).Bytes()
	scCallFunctionAsHex := hex.EncodeToString([]byte("functionToCall"))
	scCallArg := hex.EncodeToString([]byte("arg"))
	scCallArgs := [][]byte{[]byte(scCallFunctionAsHex), []byte(scCallArg)}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			CallerAddr:  senderAddress,
			Arguments:   [][]byte{tokenName, nonceBytes, quantityBytes, destinationAddress},
			GasProvided: 1,
		},
		RecipientAddr: senderAddress,
	}
	vmInput.Arguments = append(vmInput.Arguments, scCallArgs...)

	return vmInput, sender, nftTransferSenderShard, dcdtDataStorageHandler, tokenName, tokenNonce
}
