package builtInFunctions

import (
	"bytes"
	"errors"
	"math/big"
	"testing"

	"github.com/DharitriOne/drt-chain-core-go/core"
	"github.com/DharitriOne/drt-chain-core-go/core/check"
	"github.com/DharitriOne/drt-chain-core-go/data/dcdt"
	"github.com/DharitriOne/drt-chain-core-go/data/vm"
	vmcommon "github.com/DharitriOne/drt-chain-vm-common-go"
	"github.com/DharitriOne/drt-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createNftCreateWithStubArguments() *dcdtNFTCreate {
	nftCreate, _ := NewDCDTNFTCreateFunc(
		1,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.DCDTRoleHandlerStub{},
		createNewDCDTDataStorageHandler(),
		&mock.AccountsStub{},
		&mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == ValueLengthCheckFlag
			},
		},
	)

	return nftCreate
}

func TestNewDCDTNFTCreateFunc_NilArgumentsShouldErr(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		nftCreate, err := NewDCDTNFTCreateFunc(
			0,
			vmcommon.BaseOperationCost{},
			nil,
			&mock.GlobalSettingsHandlerStub{},
			&mock.DCDTRoleHandlerStub{},
			createNewDCDTDataStorageHandler(),
			&mock.AccountsStub{},
			&mock.EnableEpochsHandlerStub{},
		)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilMarshalizer, err)
	})
	t.Run("nil global settings handler should error", func(t *testing.T) {
		t.Parallel()

		nftCreate, err := NewDCDTNFTCreateFunc(
			0,
			vmcommon.BaseOperationCost{},
			&mock.MarshalizerMock{},
			nil,
			&mock.DCDTRoleHandlerStub{},
			createNewDCDTDataStorageHandler(),
			&mock.AccountsStub{},
			&mock.EnableEpochsHandlerStub{},
		)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilGlobalSettingsHandler, err)
	})
	t.Run("nil roles handler should error", func(t *testing.T) {
		t.Parallel()

		nftCreate, err := NewDCDTNFTCreateFunc(
			0,
			vmcommon.BaseOperationCost{},
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			nil,
			createNewDCDTDataStorageHandler(),
			&mock.AccountsStub{},
			&mock.EnableEpochsHandlerStub{},
		)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilRolesHandler, err)
	})
	t.Run("nil dcdt storage handler should error", func(t *testing.T) {
		t.Parallel()

		nftCreate, err := NewDCDTNFTCreateFunc(
			0,
			vmcommon.BaseOperationCost{},
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.DCDTRoleHandlerStub{},
			nil,
			&mock.AccountsStub{},
			&mock.EnableEpochsHandlerStub{},
		)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilDCDTNFTStorageHandler, err)
	})
	t.Run("nil enable epochs handler should error", func(t *testing.T) {
		t.Parallel()

		nftCreate, err := NewDCDTNFTCreateFunc(
			0,
			vmcommon.BaseOperationCost{},
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.DCDTRoleHandlerStub{},
			createNewDCDTDataStorageHandler(),
			&mock.AccountsStub{},
			nil,
		)
		assert.True(t, check.IfNil(nftCreate))
		assert.Equal(t, ErrNilEnableEpochsHandler, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		nftCreate, err := NewDCDTNFTCreateFunc(
			0,
			vmcommon.BaseOperationCost{},
			&mock.MarshalizerMock{},
			&mock.GlobalSettingsHandlerStub{},
			&mock.DCDTRoleHandlerStub{},
			createNewDCDTDataStorageHandler(),
			&mock.AccountsStub{},
			&mock.EnableEpochsHandlerStub{},
		)
		assert.Nil(t, err)
		assert.False(t, check.IfNil(nftCreate))
	})
}

func TestNewDCDTNFTCreateFunc(t *testing.T) {
	t.Parallel()

	nftCreate, err := NewDCDTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.DCDTRoleHandlerStub{},
		createNewDCDTDataStorageHandler(),
		&mock.AccountsStub{},
		&mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == ValueLengthCheckFlag
			},
		},
	)
	assert.False(t, check.IfNil(nftCreate))
	assert.Nil(t, err)
}

func TestDcdtNFTCreate_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	nftCreate := createNftCreateWithStubArguments()
	nftCreate.SetNewGasConfig(nil)
	assert.Equal(t, uint64(1), nftCreate.funcGasCost)
	assert.Equal(t, vmcommon.BaseOperationCost{}, nftCreate.gasConfig)

	gasCost := createMockGasCost()
	nftCreate.SetNewGasConfig(&gasCost)
	assert.Equal(t, gasCost.BuiltInCost.DCDTNFTCreate, nftCreate.funcGasCost)
	assert.Equal(t, gasCost.BaseOperationCost, nftCreate.gasConfig)
}

func TestDcdtNFTCreate_ProcessBuiltinFunctionInvalidArguments(t *testing.T) {
	t.Parallel()

	nftCreate := createNftCreateWithStubArguments()
	sender := mock.NewAccountWrapMock([]byte("address"))
	vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, nil)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNilVmInput, err)

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: []byte("caller"),
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), []byte("arg2")},
		},
		RecipientAddr: []byte("recipient"),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrInvalidRcvAddr, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), []byte("arg2")},
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNilUserAccount, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), []byte("arg2")},
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, ErrNotEnoughGas, err)

	vmInput = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr:  sender.AddressBytes(),
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), []byte("arg2")},
			GasProvided: 1,
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err = nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.True(t, errors.Is(err, ErrInvalidArguments))
}

func TestDcdtNFTCreate_ProcessBuiltinFunctionNotAllowedToExecute(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("expected error")
	dcdtDataStorage := createNewDCDTDataStorageHandler()
	nftCreate, _ := NewDCDTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.DCDTRoleHandlerStub{
			CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
				return expectedErr
			},
		},
		dcdtDataStorage,
		dcdtDataStorage.accounts,
		&mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == ValueLengthCheckFlag
			},
		},
	)
	sender := mock.NewAccountWrapMock([]byte("address"))
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments:  make([][]byte, 7),
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, vmOutput)
	assert.Equal(t, expectedErr, err)
}

func TestDcdtNFTCreate_ProcessBuiltinFunctionShouldWork(t *testing.T) {
	t.Parallel()

	dcdtDataStorage := createNewDCDTDataStorageHandler()
	firstCheck := true
	dcdtRoleHandler := &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			if firstCheck {
				assert.Equal(t, core.DCDTRoleNFTCreate, string(action))
				firstCheck = false
			} else {
				assert.Equal(t, core.DCDTRoleNFTAddQuantity, string(action))
			}
			return nil
		},
	}
	nftCreate, _ := NewDCDTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		dcdtRoleHandler,
		dcdtDataStorage,
		dcdtDataStorage.accounts,
		&mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == ValueLengthCheckFlag
			},
		},
	)
	address := bytes.Repeat([]byte{1}, 32)
	sender := mock.NewUserAccount(address)
	//add some data in the trie, otherwise the creation will fail (it won't happen in real case usage as the create NFT
	//will be called after the creation permission was set in the account's data)
	_ = sender.AccountDataHandler().SaveKeyValue([]byte("key"), []byte("value"))

	token := "token"
	quantity := big.NewInt(2)
	name := "name"
	royalties := 100 //1%
	hash := []byte("12345678901234567890123456789012")
	attributes := []byte("attributes")
	uris := [][]byte{[]byte("uri1"), []byte("uri2")}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: sender.AddressBytes(),
			CallValue:  big.NewInt(0),
			Arguments: [][]byte{
				[]byte(token),
				quantity.Bytes(),
				[]byte(name),
				big.NewInt(int64(royalties)).Bytes(),
				hash,
				attributes,
				uris[0],
				uris[1],
			},
		},
		RecipientAddr: sender.AddressBytes(),
	}
	vmOutput, err := nftCreate.ProcessBuiltinFunction(sender, nil, vmInput)
	assert.Nil(t, err)
	require.NotNil(t, vmOutput)

	createdDcdt, latestNonce := readNFTData(t, sender, nftCreate.marshaller, []byte(token), 1, address)
	assert.Equal(t, uint64(1), latestNonce)
	expectedDcdt := &dcdt.DCDigitalToken{
		Type:  uint32(core.NonFungible),
		Value: quantity,
	}
	assert.Equal(t, expectedDcdt, createdDcdt)

	tokenMetaData := &dcdt.MetaData{
		Nonce:      1,
		Name:       []byte(name),
		Creator:    address,
		Royalties:  uint32(royalties),
		Hash:       hash,
		URIs:       uris,
		Attributes: attributes,
	}

	tokenKey := []byte(baseDCDTKeyPrefix + token)
	tokenKey = append(tokenKey, big.NewInt(1).Bytes()...)

	dcdtData, _, _ := dcdtDataStorage.getDCDTDigitalTokenDataFromSystemAccount(tokenKey, defaultQueryOptions())
	assert.Equal(t, tokenMetaData, dcdtData.TokenMetaData)
	assert.Equal(t, dcdtData.Value, quantity)

	dcdtDataBytes := vmOutput.Logs[0].Topics[3]
	var dcdtDataFromLog dcdt.DCDigitalToken
	_ = nftCreate.marshaller.Unmarshal(&dcdtDataFromLog, dcdtDataBytes)
	require.Equal(t, dcdtData.TokenMetaData, dcdtDataFromLog.TokenMetaData)
}

func TestDcdtNFTCreate_ProcessBuiltinFunctionWithExecByCaller(t *testing.T) {
	t.Parallel()

	accounts := createAccountsAdapterWithMap()
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ValueLengthCheckFlag || flag == SaveToSystemAccountFlag || flag == CheckFrozenCollectionFlag
		},
	}
	dcdtDataStorage := createNewDCDTDataStorageHandlerWithArgs(&mock.GlobalSettingsHandlerStub{}, accounts, enableEpochsHandler)
	nftCreate, _ := NewDCDTNFTCreateFunc(
		0,
		vmcommon.BaseOperationCost{},
		&mock.MarshalizerMock{},
		&mock.GlobalSettingsHandlerStub{},
		&mock.DCDTRoleHandlerStub{},
		dcdtDataStorage,
		dcdtDataStorage.accounts,
		enableEpochsHandler,
	)
	address := bytes.Repeat([]byte{1}, 32)
	userAddress := bytes.Repeat([]byte{2}, 32)
	token := "token"
	quantity := big.NewInt(2)
	name := "name"
	royalties := 100 //1%
	hash := []byte("12345678901234567890123456789012")
	attributes := []byte("attributes")
	uris := [][]byte{[]byte("uri1"), []byte("uri2")}
	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallerAddr: userAddress,
			CallValue:  big.NewInt(0),
			Arguments: [][]byte{
				[]byte(token),
				quantity.Bytes(),
				[]byte(name),
				big.NewInt(int64(royalties)).Bytes(),
				hash,
				attributes,
				uris[0],
				uris[1],
				address,
			},
			CallType: vm.ExecOnDestByCaller,
		},
		RecipientAddr: userAddress,
	}
	vmOutput, err := nftCreate.ProcessBuiltinFunction(nil, nil, vmInput)
	assert.Nil(t, err)
	require.NotNil(t, vmOutput)

	roleAcc, _ := nftCreate.getAccount(address)

	createdDcdt, latestNonce := readNFTData(t, roleAcc, nftCreate.marshaller, []byte(token), 1, address)
	assert.Equal(t, uint64(1), latestNonce)
	expectedDcdt := &dcdt.DCDigitalToken{
		Type:  uint32(core.NonFungible),
		Value: quantity,
	}
	assert.Equal(t, expectedDcdt, createdDcdt)

	tokenMetaData := &dcdt.MetaData{
		Nonce:      1,
		Name:       []byte(name),
		Creator:    userAddress,
		Royalties:  uint32(royalties),
		Hash:       hash,
		URIs:       uris,
		Attributes: attributes,
	}

	tokenKey := []byte(baseDCDTKeyPrefix + token)
	tokenKey = append(tokenKey, big.NewInt(1).Bytes()...)

	metaData, _ := dcdtDataStorage.getDCDTMetaDataFromSystemAccount(tokenKey, defaultQueryOptions())
	assert.Equal(t, tokenMetaData, metaData)
}

func readNFTData(t *testing.T, account vmcommon.UserAccountHandler, marshaller vmcommon.Marshalizer, tokenID []byte, nonce uint64, _ []byte) (*dcdt.DCDigitalToken, uint64) {
	nonceKey := getNonceKey(tokenID)
	latestNonceBytes, _, err := account.AccountDataHandler().RetrieveValue(nonceKey)
	require.Nil(t, err)
	latestNonce := big.NewInt(0).SetBytes(latestNonceBytes).Uint64()

	createdTokenID := []byte(baseDCDTKeyPrefix)
	createdTokenID = append(createdTokenID, tokenID...)
	tokenKey := computeDCDTNFTTokenKey(createdTokenID, nonce)
	data, _, err := account.AccountDataHandler().RetrieveValue(tokenKey)
	require.Nil(t, err)

	dcdtData := &dcdt.DCDigitalToken{}
	err = marshaller.Unmarshal(dcdtData, data)
	require.Nil(t, err)

	return dcdtData, latestNonce
}
