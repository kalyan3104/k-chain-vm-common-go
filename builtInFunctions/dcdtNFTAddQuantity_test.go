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

func TestNewDCDTNFTAddQuantityFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		eqf, err := NewDCDTNFTAddQuantityFunc(10, nil, nil, nil, nil)
		require.True(t, check.IfNil(eqf))
		require.Equal(t, ErrNilDCDTNFTStorageHandler, err)
	})
	t.Run("nil global settings handler should error", func(t *testing.T) {
		t.Parallel()

		eqf, err := NewDCDTNFTAddQuantityFunc(10, createNewDCDTDataStorageHandler(), nil, nil, nil)
		require.True(t, check.IfNil(eqf))
		require.Equal(t, ErrNilGlobalSettingsHandler, err)
	})
	t.Run("nil roles handler should error", func(t *testing.T) {
		t.Parallel()

		eqf, err := NewDCDTNFTAddQuantityFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, nil, nil)
		require.True(t, check.IfNil(eqf))
		require.Equal(t, ErrNilRolesHandler, err)
	})
	t.Run("nil enable epochs handler should error", func(t *testing.T) {
		t.Parallel()

		eqf, err := NewDCDTNFTAddQuantityFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, nil)
		require.True(t, check.IfNil(eqf))
		require.Equal(t, ErrNilEnableEpochsHandler, err)
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		eqf, err := NewDCDTNFTAddQuantityFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
		require.False(t, check.IfNil(eqf))
		require.NoError(t, err)
	})
}

func TestDcdtNFTAddQuantity_SetNewGasConfig_NilGasCost(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	eqf, _ := NewDCDTNFTAddQuantityFunc(defaultGasCost, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ValueLengthCheckFlag
		},
	})

	eqf.SetNewGasConfig(nil)
	require.Equal(t, defaultGasCost, eqf.funcGasCost)
}

func TestDcdtNFTAddQuantity_SetNewGasConfig_ShouldWork(t *testing.T) {
	t.Parallel()

	defaultGasCost := uint64(10)
	newGasCost := uint64(37)
	eqf, _ := NewDCDTNFTAddQuantityFunc(defaultGasCost, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ValueLengthCheckFlag
		},
	})

	eqf.SetNewGasConfig(
		&vmcommon.GasCost{
			BuiltInCost: vmcommon.BuiltInCost{
				DCDTNFTAddQuantity: newGasCost,
			},
		},
	)

	require.Equal(t, newGasCost, eqf.funcGasCost)
}

func TestDcdtNFTAddQuantity_ProcessBuiltinFunctionErrorOnCheckDCDTNFTCreateBurnAddInput(t *testing.T) {
	t.Parallel()

	eqf, _ := NewDCDTNFTAddQuantityFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ValueLengthCheckFlag
		},
	})

	// nil vm input
	output, err := eqf.ProcessBuiltinFunction(mock.NewAccountWrapMock([]byte("addr")), nil, nil)
	require.Nil(t, output)
	require.Equal(t, ErrNilVmInput, err)

	// vm input - value not zero
	output, err = eqf.ProcessBuiltinFunction(
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
	output, err = eqf.ProcessBuiltinFunction(
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
	output, err = eqf.ProcessBuiltinFunction(
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
	output, err = eqf.ProcessBuiltinFunction(
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
	output, err = eqf.ProcessBuiltinFunction(
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
	output, err = eqf.ProcessBuiltinFunction(
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

func TestDcdtNFTAddQuantity_ProcessBuiltinFunctionInvalidNumberOfArguments(t *testing.T) {
	t.Parallel()

	eqf, _ := NewDCDTNFTAddQuantityFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ValueLengthCheckFlag
		},
	})
	output, err := eqf.ProcessBuiltinFunction(
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

func TestDcdtNFTAddQuantity_ProcessBuiltinFunctionCheckAllowedToExecuteError(t *testing.T) {
	t.Parallel()

	localErr := errors.New("err")
	rolesHandler := &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(_ vmcommon.UserAccountHandler, _ []byte, _ []byte) error {
			return localErr
		},
	}
	eqf, _ := NewDCDTNFTAddQuantityFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, rolesHandler, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ValueLengthCheckFlag
		},
	})
	output, err := eqf.ProcessBuiltinFunction(
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

func TestDcdtNFTAddQuantity_ProcessBuiltinFunctionNewSenderShouldErr(t *testing.T) {
	t.Parallel()

	eqf, _ := NewDCDTNFTAddQuantityFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ValueLengthCheckFlag
		},
	})
	output, err := eqf.ProcessBuiltinFunction(
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

func TestDcdtNFTAddQuantity_ProcessBuiltinFunctionMetaDataMissing(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	eqf, _ := NewDCDTNFTAddQuantityFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ValueLengthCheckFlag
		},
	})

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dcdtData := &dcdt.DCDigitalToken{}
	dcdtDataBytes, _ := marshaller.Marshal(dcdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.ProtectedKeyPrefix+core.DCDTKeyIdentifier+"arg0"), dcdtDataBytes)
	output, err := eqf.ProcessBuiltinFunction(
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

func TestDcdtNFTAddQuantity_ProcessBuiltinFunctionShouldErrOnSaveBecauseTokenIsPaused(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{
		IsPausedCalled: func(_ []byte) bool {
			return true
		},
	}
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{}
	eqf, _ := NewDCDTNFTAddQuantityFunc(10, createNewDCDTDataStorageHandlerWithArgs(globalSettingsHandler, &mock.AccountsStub{}, enableEpochsHandler), globalSettingsHandler, &mock.DCDTRoleHandlerStub{}, enableEpochsHandler)

	userAcc := mock.NewAccountWrapMock([]byte("addr"))
	dcdtData := &dcdt.DCDigitalToken{
		TokenMetaData: &dcdt.MetaData{
			Name: []byte("test"),
		},
		Value: big.NewInt(10),
	}
	dcdtDataBytes, _ := marshaller.Marshal(dcdtData)
	_ = userAcc.AccountDataHandler().SaveKeyValue([]byte(core.ProtectedKeyPrefix+core.DCDTKeyIdentifier+"arg0"+"arg1"), dcdtDataBytes)

	output, err := eqf.ProcessBuiltinFunction(
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

func TestDcdtNFTAddQuantity_ProcessBuiltinFunctionShouldWork(t *testing.T) {
	t.Parallel()

	tokenIdentifier := "testTkn"
	key := baseDCDTKeyPrefix + tokenIdentifier

	nonce := big.NewInt(33)
	initialValue := big.NewInt(5)
	valueToAdd := big.NewInt(37)
	expectedValue := big.NewInt(0).Add(initialValue, valueToAdd)

	marshaller := &mock.MarshalizerMock{}
	dcdtRoleHandler := &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.DCDTRoleNFTAddQuantity, string(action))
			return nil
		},
	}
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ValueLengthCheckFlag
		},
	}
	eqf, _ := NewDCDTNFTAddQuantityFunc(10, createNewDCDTDataStorageHandler(), &mock.GlobalSettingsHandlerStub{}, dcdtRoleHandler, enableEpochsHandler)

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

	output, err := eqf.ProcessBuiltinFunction(
		userAcc,
		nil,
		&vmcommon.ContractCallInput{
			VMInput: vmcommon.VMInput{
				CallValue:   big.NewInt(0),
				Arguments:   [][]byte{[]byte(tokenIdentifier), nonce.Bytes(), valueToAdd.Bytes()},
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
	require.Equal(t, expectedValue.Bytes(), finalTokenData.Value.Bytes())
}
