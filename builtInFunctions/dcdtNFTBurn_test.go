package builtInFunctions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/DharitriOne/drt-chain-core-go/core"
	"github.com/DharitriOne/drt-chain-core-go/core/check"
	"github.com/DharitriOne/drt-chain-core-go/data/dcdt"
	vmcommon "github.com/DharitriOne/drt-chain-vm-common-go"
	"github.com/DharitriOne/drt-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDCDTNFTBurnFunc(t *testing.T) {
	t.Parallel()

	// nil marshaller
	ebf, err := NewDCDTNFTBurnFunc(10, nil, nil, nil)
	require.True(t, check.IfNil(ebf))
	require.Equal(t, ErrNilDCDTNFTStorageHandler, err)

	// nil pause handler
	ebf, err = NewDCDTNFTBurnFunc(10, createNewDCDTDataStorageHandler(), nil, nil)
	require.True(t, check.IfNil(ebf))
	require.Equal(t, ErrNilGlobalSettingsHandler, err)

	// nil roles handler
	ebf, err = NewDCDTNFTBurnFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, nil)
	require.True(t, check.IfNil(ebf))
	require.Equal(t, ErrNilRolesHandler, err)

	// should work
	ebf, err = NewDCDTNFTBurnFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{})
	require.False(t, check.IfNil(ebf))
	require.NoError(t, err)
}

func TestDCDTNFTBurn_SetNewGasConfig_NilGasCost(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	ebf, _ := NewDCDTNFTBurnFunc(defaultGasCost, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{})

	ebf.SetNewGasConfig(nil)
	require.Equal(t, defaultGasCost, ebf.funcGasCost)
}

func TestDcdtNFTBurnFunc_SetNewGasConfig_ShouldWork(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	newGasCost := uint64(37)
	ebf, _ := NewDCDTNFTBurnFunc(defaultGasCost, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{})

	ebf.SetNewGasConfig(
		&vmcommon.GasCost{
			BuiltInCost: vmcommon.BuiltInCost{
				DCDTNFTBurn: newGasCost,
			},
		},
	)

	require.Equal(t, newGasCost, ebf.funcGasCost)
}

func TestDcdtNFTBurnFunc_ProcessBuiltinFunctionErrorOnCheckDCDTNFTCreateBurnAddInput(t *testing.T) {
	t.Parallel()

	ebf, _ := NewDCDTNFTBurnFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{})

	// nil vm input
	output, err := ebf.ProcessBuiltinFunction(mock.NewAccountWrapMock([]byte("addr")), nil, nil)
	require.Nil(t, output)
	require.Equal(t, ErrNilVmInput, err)

	// vm input - value not zero
	output, err = ebf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: big.NewInt(37),
			},
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrBuiltInFunctionCalledWithValue, err)

	// vm input - invalid number of arguments
	output, err = ebf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: big.NewInt(0),
				Arguments: [][]byte{[]byte("single arg")},
			},
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrInvalidArguments, err)

	// vm input - invalid number of arguments
	output, err = ebf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue: big.NewInt(0),
				Arguments: [][]byte{[]byte("arg0")},
			},
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrInvalidArguments, err)

	// vm input - invalid receiver
	output, err = ebf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				Arguments:  [][]byte{[]byte("arg0"), []byte("arg1")},
				CallerAddr: []byte("address 1"),
			},
			RecipientAddr: []byte("address 2"),
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrInvalidRcvAddr, err)

	// nil user account
	output, err = ebf.ProcessBuiltinFunction(
		nil,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:  big.NewInt(0),
				Arguments:  [][]byte{[]byte("arg0"), []byte("arg1")},
				CallerAddr: []byte("address 1"),
			},
			RecipientAddr: []byte("address 1"),
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrNilUserAccount, err)

	// not enough gas
	output, err = ebf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 1,
			},
			RecipientAddr: []byte("address 1"),
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrNotEnoughGas, err)
}

func TestDcdtNFTBurnFunc_ProcessBuiltinFunctionInvalidNumberOfArguments(t *testing.T) {
	t.Parallel()

	ebf, _ := NewDCDTNFTBurnFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{})
	output, err := ebf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)
	require.Nil(t, output)
	require.Equal(t, ErrInvalidArguments, err)
}

func TestDcdtNFTBurnFunc_ProcessBuiltinFunctionCheckAllowedToExecuteError(t *testing.T) {
	t.Parallel()

	localErr := errors.New("err")
	rolesHandler := &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(_ vmcommon.UserAccountHandler, _ []byte, _ []byte) error {
			return localErr
		},
	}
	ebf, _ := NewDCDTNFTBurnFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, rolesHandler)
	output, err := ebf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1"), []byte("arg2")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Equal(t, localErr, err)
}

func TestDcdtNFTBurnFunc_ProcessBuiltinFunctionNewSenderShouldErr(t *testing.T) {
	t.Parallel()

	ebf, _ := NewDCDTNFTBurnFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{})
	output, err := ebf.ProcessBuiltinFunction(
		mock.NewAccountWrapMock([]byte("addr")),
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1"), []byte("arg2")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Error(t, err)
	require.Equal(t, ErrNewNFTDataOnSenderAddress, err)
}

func TestDcdtNFTBurnFunc_ProcessBuiltinFunctionMetaDataMissing(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	ebf, _ := NewDCDTNFTBurnFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dcdtData := &dcdt.DCDigitalToken{}
	dcdtDataBytes, _ := marshaller.Marshal(dcdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.ProtectedKeyPrefix+core.DCDTKeyIdentifier+"arg0"), dcdtDataBytes)
	output, err := ebf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), {0}, []byte("arg2")},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Equal(t, ErrNFTDoesNotHaveMetadata, err)
}

func TestDcdtNFTBurnFunc_ProcessBuiltinFunctionInvalidBurnQuantity(t *testing.T) {
	t.Parallel()

	initialQuantity := big.NewInt(55)
	quantityToBurn := big.NewInt(75)

	marshaller := &mock.MarshalizerMock{}

	ebf, _ := NewDCDTNFTBurnFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dcdtData := &dcdt.DCDigitalToken{
		TokenMetaData: &dcdt.MetaData{
			Name: []byte("test"),
		},
		Value: initialQuantity,
	}
	dcdtDataBytes, _ := marshaller.Marshal(dcdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.ProtectedKeyPrefix+core.DCDTKeyIdentifier+"arg0"+"arg1"), dcdtDataBytes)
	output, err := ebf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1"), quantityToBurn.Bytes()},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Equal(t, ErrInvalidNFTQuantity, err)
}

func TestDcdtNFTBurnFunc_ProcessBuiltinFunctionShouldErrOnSaveBecauseTokenIsPaused(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{
		IsPausedCalled: func(_ []byte) bool {
			return true
		},
	}

	ebf, _ := NewDCDTNFTBurnFunc(10, createNewDCDTDataStorageHandlerWithArgs(globalSettingsHandler, &mock.AccountsStub{}, &mock.EnableEpochsHandlerStub{}), globalSettingsHandler, &mock.DCDTRoleHandlerStub{})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dcdtData := &dcdt.DCDigitalToken{
		TokenMetaData: &dcdt.MetaData{
			Name: []byte("test"),
		},
		Value: big.NewInt(10),
	}
	dcdtDataBytes, _ := marshaller.Marshal(dcdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.ProtectedKeyPrefix+core.DCDTKeyIdentifier+"arg0"+"arg1"), dcdtDataBytes)
	output, err := ebf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte("arg0"), []byte("arg1"), big.NewInt(5).Bytes()},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.Nil(t, output)
	require.Equal(t, ErrDCDTTokenIsPaused, err)
}

func TestDcdtNFTBurnFunc_ProcessBuiltinFunctionShouldWork(t *testing.T) {
	t.Parallel()

	tokenIdentifier := "testTkn"
	key := baseDCDTKeyPrefix + tokenIdentifier

	nonce := big.NewInt(33)
	initialQuantity := big.NewInt(100)
	quantityToBurn := big.NewInt(37)
	expectedQuantity := big.NewInt(0).Sub(initialQuantity, quantityToBurn)

	marshaller := &mock.MarshalizerMock{}
	dcdtRoleHandler := &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.DCDTRoleNFTBurn, string(action))
			return nil
		},
	}
	storageHandler := createNewDCDTDataStorageHandler()
	ebf, _ := NewDCDTNFTBurnFunc(10, storageHandler, &mock.GlobalSettingsHandlerStub{}, dcdtRoleHandler)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dcdtData := &dcdt.DCDigitalToken{
		TokenMetaData: &dcdt.MetaData{
			Name: []byte("test"),
		},
		Value: initialQuantity,
	}
	dcdtDataBytes, _ := marshaller.Marshal(dcdtData)
	nftTokenKey := append([]byte(key), nonce.Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(nftTokenKey, dcdtDataBytes)

	_ = storageHandler.saveDCDTMetaDataToSystemAccount(userAcc, 0, nftTokenKey, nonce.Uint64(), dcdtData, true)
	_ = storageHandler.AddToLiquiditySystemAcc([]byte(key), nonce.Uint64(), initialQuantity)
	output, err := ebf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte(tokenIdentifier), nonce.Bytes(), quantityToBurn.Bytes()},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.NotNil(t, output)
	require.NoError(t, err)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)

	res, _, err := userAcc.AccountDataHandler().RetrieveValue(nftTokenKey)
	require.NoError(t, err)
	require.NotNil(t, res)

	finalTokenData := dcdt.DCDigitalToken{}
	_ = marshaller.Unmarshal(&finalTokenData, res)
	require.Equal(t, expectedQuantity.Bytes(), finalTokenData.Value.Bytes())
}

func TestDcdtNFTBurnFunc_ProcessBuiltinFunctionWithGlobalBurn(t *testing.T) {
	t.Parallel()

	tokenIdentifier := "testTkn"
	key := baseDCDTKeyPrefix + tokenIdentifier

	nonce := big.NewInt(33)
	initialQuantity := big.NewInt(100)
	quantityToBurn := big.NewInt(37)
	expectedQuantity := big.NewInt(0).Sub(initialQuantity, quantityToBurn)

	marshaller := &mock.MarshalizerMock{}
	storageHandler := createNewDCDTDataStorageHandler()
	ebf, _ := NewDCDTNFTBurnFunc(10, storageHandler, &mock.GlobalSettingsHandlerStub{
		IsBurnForAllCalled: func(token []byte) bool {
			return true
		},
	}, &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return errors.New("no burn allowed")
		},
	})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dcdtData := &dcdt.DCDigitalToken{
		TokenMetaData: &dcdt.MetaData{
			Name: []byte("test"),
		},
		Value: initialQuantity,
	}
	dcdtDataBytes, _ := marshaller.Marshal(dcdtData)
	tokenKey := append([]byte(key), nonce.Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, dcdtDataBytes)
	_ = storageHandler.saveDCDTMetaDataToSystemAccount(userAcc, 0, tokenKey, nonce.Uint64(), dcdtData, true)
	_ = storageHandler.AddToLiquiditySystemAcc([]byte(key), nonce.Uint64(), initialQuantity)

	output, err := ebf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte(tokenIdentifier), nonce.Bytes(), quantityToBurn.Bytes()},
				CallerAddr:  []byte("address 1"),
				GasProvided: 12,
			},
			RecipientAddr: []byte("address 1"),
		},
	)

	require.NotNil(t, output)
	require.NoError(t, err)
	require.Equal(t, vmcommon.Ok, output.ReturnCode)

	res, _, err := userAcc.AccountDataHandler().RetrieveValue(tokenKey)
	require.NoError(t, err)
	require.NotNil(t, res)

	finalTokenData := dcdt.DCDigitalToken{}
	_ = marshaller.Unmarshal(&finalTokenData, res)
	require.Equal(t, expectedQuantity.Bytes(), finalTokenData.Value.Bytes())
}
