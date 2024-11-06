package builtInFunctions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/core/check"
	"github.com/kalyan3104/k-chain-core-go/data/dcdt"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
	"github.com/kalyan3104/k-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDCDTNFTAddUriFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		e, err := NewDCDTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, nil, nil, nil, nil)
		require.True(t, check.IfNil(e))
		require.Equal(t, ErrNilDCDTNFTStorageHandler, err)
	})
	t.Run("nil global settings handler should error", func(t *testing.T) {
		t.Parallel()

		e, err := NewDCDTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewDCDTDataStorageHandler(), nil, nil, nil)
		require.True(t, check.IfNil(e))
		require.Equal(t, ErrNilGlobalSettingsHandler, err)
	})
	t.Run("nil roles handler should error", func(t *testing.T) {
		t.Parallel()

		e, err := NewDCDTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, nil, nil)
		require.True(t, check.IfNil(e))
		require.Equal(t, ErrNilRolesHandler, err)
	})
	t.Run("nil enable epochs handler should error", func(t *testing.T) {
		t.Parallel()

		e, err := NewDCDTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, nil)
		require.True(t, check.IfNil(e))
		require.Equal(t, ErrNilEnableEpochsHandler, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		e, err := NewDCDTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
		require.False(t, check.IfNil(e))
		require.NoError(t, err)
		require.False(t, e.IsActive())
	})
}

func TestDCDTNFTAddUri_SetNewGasConfig_NilGasCost(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	e, _ := NewDCDTNFTAddUriFunc(defaultGasCost, vmcommon.BaseOperationCost{}, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == DCDTNFTImprovementV1Flag
		},
	})

	e.SetNewGasConfig(nil)
	require.Equal(t, defaultGasCost, e.funcGasCost)
}

func TestDCDTNFTAddUri_SetNewGasConfig_ShouldWork(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	newGasCost := uint64(37)
	e, _ := NewDCDTNFTAddUriFunc(defaultGasCost, vmcommon.BaseOperationCost{}, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == DCDTNFTImprovementV1Flag
		},
	})

	e.SetNewGasConfig(
		&vmcommon.GasCost{
			BuiltInCost: vmcommon.BuiltInCost{
				DCDTNFTAddURI: newGasCost,
			},
		},
	)

	require.Equal(t, newGasCost, e.funcGasCost)
}

func TestDCDTNFTAddUri_ProcessBuiltinFunctionErrorOnCheckInput(t *testing.T) {
	t.Parallel()

	e, _ := NewDCDTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == DCDTNFTImprovementV1Flag
		},
	})

	// nil vm input
	output, err := e.ProcessBuiltinFunction(mock.NewAccountWrapMock([]byte("addr")), nil, nil)
	require.Nil(t, output)
	require.Equal(t, ErrNilVmInput, err)

	// vm input - value not zero
	output, err = e.ProcessBuiltinFunction(
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
	output, err = e.ProcessBuiltinFunction(
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
	output, err = e.ProcessBuiltinFunction(
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
	output, err = e.ProcessBuiltinFunction(
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
	output, err = e.ProcessBuiltinFunction(
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
	output, err = e.ProcessBuiltinFunction(
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

func TestDCDTNFTAddUri_ProcessBuiltinFunctionInvalidNumberOfArguments(t *testing.T) {
	t.Parallel()

	e, _ := NewDCDTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == DCDTNFTImprovementV1Flag
		},
	})
	output, err := e.ProcessBuiltinFunction(
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

func TestDCDTNFTAddUri_ProcessBuiltinFunctionCheckAllowedToExecuteError(t *testing.T) {
	t.Parallel()

	localErr := errors.New("err")
	rolesHandler := &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(_ vmcommon.UserAccountHandler, _ []byte, _ []byte) error {
			return localErr
		},
	}
	e, _ := NewDCDTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, rolesHandler, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == DCDTNFTImprovementV1Flag
		},
	})
	output, err := e.ProcessBuiltinFunction(
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

func TestDCDTNFTAddUri_ProcessBuiltinFunctionNewSenderShouldErr(t *testing.T) {
	t.Parallel()

	e, _ := NewDCDTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == DCDTNFTImprovementV1Flag
		},
	})
	output, err := e.ProcessBuiltinFunction(
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

func TestDCDTNFTAddUri_ProcessBuiltinFunctionMetaDataMissing(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	e, _ := NewDCDTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == DCDTNFTImprovementV1Flag
		},
	})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dcdtData := &dcdt.DCDigitalToken{}
	dcdtDataBytes, _ := marshaller.Marshal(dcdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.ProtectedKeyPrefix+core.DCDTKeyIdentifier+"arg0"+"arg1"), dcdtDataBytes)
	output, err := e.ProcessBuiltinFunction(
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

func TestDCDTNFTAddUri_ProcessBuiltinFunctionShouldErrOnSaveBecauseTokenIsPaused(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{
		IsPausedCalled: func(_ []byte) bool {
			return true
		},
	}
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == DCDTNFTImprovementV1Flag
		},
	}
	e, _ := NewDCDTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, createNewDCDTDataStorageHandlerWithArgs(globalSettingsHandler, &mock.AccountsStub{}, enableEpochsHandler), globalSettingsHandler, &mock.DCDTRoleHandlerStub{}, enableEpochsHandler)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dcdtData := &dcdt.DCDigitalToken{
		TokenMetaData: &dcdt.MetaData{
			Name: []byte("test"),
		},
		Value: big.NewInt(10),
	}
	dcdtDataBytes, _ := marshaller.Marshal(dcdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.ProtectedKeyPrefix+core.DCDTKeyIdentifier+"arg0"+"arg1"), dcdtDataBytes)

	output, err := e.ProcessBuiltinFunction(
		userAcc,
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
	require.Equal(t, ErrDCDTTokenIsPaused, err)
}

func TestDCDTNFTAddUri_ProcessBuiltinFunctionShouldWork(t *testing.T) {
	t.Parallel()

	tokenIdentifier := "testTkn"
	key := baseDCDTKeyPrefix + tokenIdentifier

	nonce := big.NewInt(33)
	initialValue := big.NewInt(5)
	URIToAdd := []byte("NewURI")

	dcdtDataStorage := createNewDCDTDataStorageHandler()
	marshaller := &mock.MarshalizerMock{}
	dcdtRoleHandler := &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.DCDTRoleNFTAddURI, string(action))
			return nil
		},
	}
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == DCDTNFTImprovementV1Flag
		},
	}
	e, _ := NewDCDTNFTAddUriFunc(10, vmcommon.BaseOperationCost{}, dcdtDataStorage, &mock.GlobalSettingsHandlerStub{}, dcdtRoleHandler, enableEpochsHandler)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dcdtData := &dcdt.DCDigitalToken{
		TokenMetaData: &dcdt.MetaData{
			Name: []byte("test"),
		},
		Value: initialValue,
	}
	dcdtDataBytes, _ := marshaller.Marshal(dcdtData)
	tokenKey := append([]byte(key), nonce.Bytes()...)
	_ = userAcc.AccountDataHandler().SaveKeyValue(tokenKey, dcdtDataBytes)

	output, err := e.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte(tokenIdentifier), nonce.Bytes(), URIToAdd},
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

	metaData, _ := dcdtDataStorage.getDCDTMetaDataFromSystemAccount(tokenKey, defaultQueryOptions())
	require.Equal(t, metaData.URIs[0], URIToAdd)
}
