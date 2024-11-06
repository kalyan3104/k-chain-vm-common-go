package builtInFunctions

import (
	"errors"
	"math/big"
	"testing"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/core/check"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
	"github.com/kalyan3104/k-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewDCDTGlobalSettingsFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil accounts should error", func(t *testing.T) {
		t.Parallel()

		globalSettingsFunc, err := NewDCDTGlobalSettingsFunc(nil, &mock.MarshalizerMock{}, true, core.BuiltInFunctionDCDTPause, trueHandler)
		assert.Equal(t, ErrNilAccountsAdapter, err)
		assert.True(t, check.IfNil(globalSettingsFunc))
	})
	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		globalSettingsFunc, err := NewDCDTGlobalSettingsFunc(&mock.AccountsStub{}, nil, true, core.BuiltInFunctionDCDTPause, trueHandler)
		assert.Equal(t, ErrNilMarshalizer, err)
		assert.True(t, check.IfNil(globalSettingsFunc))
	})
	t.Run("nil active handler should error", func(t *testing.T) {
		t.Parallel()

		globalSettingsFunc, err := NewDCDTGlobalSettingsFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, true, core.BuiltInFunctionDCDTPause, nil)
		assert.Equal(t, ErrNilActiveHandler, err)
		assert.True(t, check.IfNil(globalSettingsFunc))
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		globalSettingsFunc, err := NewDCDTGlobalSettingsFunc(&mock.AccountsStub{}, &mock.MarshalizerMock{}, true, core.BuiltInFunctionDCDTPause, falseHandler)
		assert.Nil(t, err)
		assert.False(t, check.IfNil(globalSettingsFunc))
	})
}

func TestDCDTGlobalSettingsPause_ProcessBuiltInFunction(t *testing.T) {
	t.Parallel()

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	globalSettingsFunc, _ := NewDCDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, true, core.BuiltInFunctionDCDTPause, falseHandler)
	_, err := globalSettingsFunc.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(1),
		},
	}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrBuiltInFunctionCalledWithValue)

	input.CallValue = big.NewInt(0)
	key := []byte("key")
	value := []byte("value")
	input.Arguments = [][]byte{key, value}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input.Arguments = [][]byte{key}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrAddressIsNotDCDTSystemSC)

	input.CallerAddr = core.DCDTSCAddress
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrOnlySystemAccountAccepted)

	input.RecipientAddr = vmcommon.SystemAccountAddress
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	pauseKey := []byte(baseDCDTKeyPrefix + string(key))
	assert.True(t, globalSettingsFunc.IsPaused(pauseKey))
	assert.False(t, globalSettingsFunc.IsLimitedTransfer(pauseKey))

	dcdtGlobalSettingsFalse, _ := NewDCDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, false, core.BuiltInFunctionDCDTUnPause, falseHandler)

	_, err = dcdtGlobalSettingsFalse.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	assert.False(t, globalSettingsFunc.IsPaused(pauseKey))
	assert.False(t, globalSettingsFunc.IsLimitedTransfer(pauseKey))
}

func TestDCDTGlobalSettingsPause_ProcessBuiltInFunctionGetNodeFromDbErr(t *testing.T) {
	t.Parallel()

	globalSettingsFunc, _ := NewDCDTGlobalSettingsFunc(
		&mock.AccountsStub{
			LoadAccountCalled: func(_ []byte) (vmcommon.AccountHandler, error) {
				return &mock.AccountWrapMock{
					RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
						return nil, 0, core.NewGetNodeFromDBErrWithKey([]byte("key"), errors.New("error"), "")
					},
				}, nil
			},
		},
		&mock.MarshalizerMock{},
		true,
		core.BuiltInFunctionDCDTPause,
		falseHandler,
	)

	key := []byte("key")
	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{key},
			CallerAddr:  core.DCDTSCAddress,
		},
		RecipientAddr: vmcommon.SystemAccountAddress,
	}

	output, err := globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, output)
	assert.True(t, core.IsGetNodeFromDBError(err))
}

func TestDCDTGlobalSettingsLimitedTransfer_ProcessBuiltInFunction(t *testing.T) {
	t.Parallel()

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	globalSettingsFunc, _ := NewDCDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, true, core.BuiltInFunctionDCDTSetLimitedTransfer, trueHandler)
	_, err := globalSettingsFunc.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(1),
		},
	}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrBuiltInFunctionCalledWithValue)

	input.CallValue = big.NewInt(0)
	key := []byte("key")
	value := []byte("value")
	input.Arguments = [][]byte{key, value}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input.Arguments = [][]byte{key}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrAddressIsNotDCDTSystemSC)

	input.CallerAddr = core.DCDTSCAddress
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrOnlySystemAccountAccepted)

	input.RecipientAddr = vmcommon.SystemAccountAddress
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	tokenID := []byte(baseDCDTKeyPrefix + string(key))
	assert.False(t, globalSettingsFunc.IsPaused(tokenID))
	assert.True(t, globalSettingsFunc.IsLimitedTransfer(tokenID))

	pauseFunc, _ := NewDCDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, true, core.BuiltInFunctionDCDTPause, falseHandler)

	_, err = pauseFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)
	assert.True(t, globalSettingsFunc.IsPaused(tokenID))
	assert.True(t, globalSettingsFunc.IsLimitedTransfer(tokenID))

	dcdtGlobalSettingsFalse, _ := NewDCDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, false, core.BuiltInFunctionDCDTUnSetLimitedTransfer, trueHandler)

	_, err = dcdtGlobalSettingsFalse.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	assert.False(t, globalSettingsFunc.IsLimitedTransfer(tokenID))
}

func TestDCDTGlobalSettingsBurnForAll_ProcessBuiltInFunction(t *testing.T) {
	t.Parallel()

	acnt := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	globalSettingsFunc, _ := NewDCDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, true, vmcommon.BuiltInFunctionDCDTSetBurnRoleForAll, falseHandler)
	_, err := globalSettingsFunc.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(1),
		},
	}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrBuiltInFunctionCalledWithValue)

	input.CallValue = big.NewInt(0)
	key := []byte("key")
	value := []byte("value")
	input.Arguments = [][]byte{key, value}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input.Arguments = [][]byte{key}
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrAddressIsNotDCDTSystemSC)

	input.CallerAddr = core.DCDTSCAddress
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrOnlySystemAccountAccepted)

	input.RecipientAddr = vmcommon.SystemAccountAddress
	_, err = globalSettingsFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	tokenID := []byte(baseDCDTKeyPrefix + string(key))
	assert.False(t, globalSettingsFunc.IsPaused(tokenID))
	assert.False(t, globalSettingsFunc.IsLimitedTransfer(tokenID))
	assert.True(t, globalSettingsFunc.IsBurnForAll(tokenID))

	pauseFunc, _ := NewDCDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, true, core.BuiltInFunctionDCDTPause, falseHandler)

	_, err = pauseFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)
	assert.True(t, globalSettingsFunc.IsPaused(tokenID))
	assert.False(t, globalSettingsFunc.IsLimitedTransfer(tokenID))
	assert.True(t, globalSettingsFunc.IsBurnForAll(tokenID))

	dcdtGlobalSettingsFalse, _ := NewDCDTGlobalSettingsFunc(&mock.AccountsStub{
		LoadAccountCalled: func(address []byte) (vmcommon.AccountHandler, error) {
			return acnt, nil
		},
	}, &mock.MarshalizerMock{}, false, vmcommon.BuiltInFunctionDCDTUnSetBurnRoleForAll, falseHandler)

	_, err = dcdtGlobalSettingsFalse.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	assert.False(t, globalSettingsFunc.IsLimitedTransfer(tokenID))
}
