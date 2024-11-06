package builtInFunctions

import (
	"bytes"
	"math/big"
	"strings"
	"testing"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/core/check"
	"github.com/kalyan3104/k-chain-core-go/data/dcdt"
	"github.com/kalyan3104/k-chain-core-go/data/vm"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
	"github.com/kalyan3104/k-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewDCDTTransferFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		transferFunc, err := NewDCDTTransferFunc(10, nil, nil, nil, nil, nil)
		assert.Equal(t, ErrNilMarshalizer, err)
		assert.True(t, check.IfNil(transferFunc))
	})
	t.Run("nil global settings handler should error", func(t *testing.T) {
		t.Parallel()

		transferFunc, err := NewDCDTTransferFunc(10, &mock.MarshalizerMock{}, nil, nil, nil, nil)
		assert.Equal(t, ErrNilGlobalSettingsHandler, err)
		assert.True(t, check.IfNil(transferFunc))
	})
	t.Run("nil shard coordinator should error", func(t *testing.T) {
		t.Parallel()

		transferFunc, err := NewDCDTTransferFunc(10, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, nil, nil, nil)
		assert.Equal(t, ErrNilShardCoordinator, err)
		assert.True(t, check.IfNil(transferFunc))
	})
	t.Run("nil roles handler should error", func(t *testing.T) {
		t.Parallel()

		transferFunc, err := NewDCDTTransferFunc(10, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, nil, nil)
		assert.Equal(t, ErrNilRolesHandler, err)
		assert.True(t, check.IfNil(transferFunc))
	})
	t.Run("nil enable epochs handler should error", func(t *testing.T) {
		t.Parallel()

		transferFunc, err := NewDCDTTransferFunc(10, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, &mock.DCDTRoleHandlerStub{}, nil)
		assert.Equal(t, ErrNilEnableEpochsHandler, err)
		assert.True(t, check.IfNil(transferFunc))
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		transferFunc, err := NewDCDTTransferFunc(10, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})
		assert.Nil(t, err)
		assert.False(t, check.IfNil(transferFunc))
	})
}
func TestDCDTTransfer_ProcessBuiltInFunctionErrors(t *testing.T) {
	t.Parallel()

	shardC := &mock.ShardCoordinatorStub{}
	transferFunc, _ := NewDCDTTransferFunc(10, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, shardC, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == CheckCorrectTokenIDForTransferRoleFlag
		},
	})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})
	_, err := transferFunc.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = transferFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := []byte("value")
	input.Arguments = [][]byte{key, value}
	_, err = transferFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Nil(t, err)

	input.GasProvided = transferFunc.funcGasCost - 1
	accSnd := mock.NewUserAccount([]byte("address"))
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrNotEnoughGas)

	input.GasProvided = transferFunc.funcGasCost
	input.RecipientAddr = core.DCDTSCAddress
	shardC.ComputeIdCalled = func(address []byte) uint32 {
		return core.MetachainShardId
	}
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrInvalidRcvAddr)
}

func TestDCDTTransfer_ProcessBuiltInFunctionSingleShard(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	dcdtRoleHandler := &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.DCDTRoleTransfer, string(action))
			return nil
		},
	}
	enableEpochsHandler := &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == CheckCorrectTokenIDForTransferRoleFlag
		},
	}
	transferFunc, _ := NewDCDTTransferFunc(10, marshaller, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, dcdtRoleHandler, enableEpochsHandler)
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))
	accDst := mock.NewUserAccount([]byte("dst"))

	_, err := transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrInsufficientFunds)

	dcdtKey := append(transferFunc.keyPrefix, key...)
	dcdtToken := &dcdt.DCDigitalToken{Value: big.NewInt(100)}
	marshaledData, _ := marshaller.Marshal(dcdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(dcdtKey, marshaledData)

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)
	marshaledData, _, _ = accSnd.AccountDataHandler().RetrieveValue(dcdtKey)
	_ = marshaller.Unmarshal(dcdtToken, marshaledData)
	assert.True(t, dcdtToken.Value.Cmp(big.NewInt(90)) == 0)

	marshaledData, _, _ = accDst.AccountDataHandler().RetrieveValue(dcdtKey)
	_ = marshaller.Unmarshal(dcdtToken, marshaledData)
	assert.True(t, dcdtToken.Value.Cmp(big.NewInt(10)) == 0)
}

func TestDCDTTransfer_ProcessBuiltInFunctionSenderInShard(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	transferFunc, _ := NewDCDTTransferFunc(10, marshaller, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == CheckCorrectTokenIDForTransferRoleFlag
		},
	})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))

	dcdtKey := append(transferFunc.keyPrefix, key...)
	dcdtToken := &dcdt.DCDigitalToken{Value: big.NewInt(100)}
	marshaledData, _ := marshaller.Marshal(dcdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(dcdtKey, marshaledData)

	_, err := transferFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Nil(t, err)
	marshaledData, _, _ = accSnd.AccountDataHandler().RetrieveValue(dcdtKey)
	_ = marshaller.Unmarshal(dcdtToken, marshaledData)
	assert.True(t, dcdtToken.Value.Cmp(big.NewInt(90)) == 0)
}

func TestDCDTTransfer_ProcessBuiltInFunctionDestInShard(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	transferFunc, _ := NewDCDTTransferFunc(10, marshaller, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == CheckCorrectTokenIDForTransferRoleFlag
		},
	})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accDst := mock.NewUserAccount([]byte("dst"))

	vmOutput, err := transferFunc.ProcessBuiltinFunction(nil, accDst, input)
	assert.Nil(t, err)
	dcdtKey := append(transferFunc.keyPrefix, key...)
	dcdtToken := &dcdt.DCDigitalToken{}
	marshaledData, _, _ := accDst.AccountDataHandler().RetrieveValue(dcdtKey)
	_ = marshaller.Unmarshal(dcdtToken, marshaledData)
	assert.True(t, dcdtToken.Value.Cmp(big.NewInt(10)) == 0)
	assert.Equal(t, uint64(0), vmOutput.GasRemaining)
}

func TestDCDTTransfer_ProcessBuiltInFunctionTooLongValue(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	transferFunc, _ := NewDCDTTransferFunc(10, marshaller, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == CheckCorrectTokenIDForTransferRoleFlag
		},
	})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	bigValueStr := "1" + strings.Repeat("0", 1000)
	bigValue, _ := big.NewInt(0).SetString(bigValueStr, 10)
	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("tkn"), bigValue.Bytes()},
		},
	}
	accDst := mock.NewUserAccount([]byte("dst"))

	// before the activation of the flag, large values should not return error
	vmOutput, err := transferFunc.ProcessBuiltinFunction(nil, accDst, input)
	assert.Nil(t, err)
	assert.NotEmpty(t, vmOutput)

	// after the activation, it should return an error
	transferFunc.enableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ConsistentTokensValuesLengthCheckFlag
		},
	}
	vmOutput, err = transferFunc.ProcessBuiltinFunction(nil, accDst, input)
	assert.Equal(t, "invalid arguments to process built-in function: max length for dcdt transfer value is 100", err.Error())
	assert.Empty(t, vmOutput)
}

func TestDCDTTransfer_SndDstFrozen(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	accountStub := &mock.AccountsStub{}
	dcdtGlobalSettingsFunc, _ := NewDCDTGlobalSettingsFunc(accountStub, marshaller, true, core.BuiltInFunctionDCDTPause, trueHandler)
	transferFunc, _ := NewDCDTTransferFunc(10, marshaller, dcdtGlobalSettingsFunc, &mock.ShardCoordinatorStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == CheckCorrectTokenIDForTransferRoleFlag
		},
	})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))
	accDst := mock.NewUserAccount([]byte("dst"))

	dcdtFrozen := DCDTUserMetadata{Frozen: true}
	dcdtNotFrozen := DCDTUserMetadata{Frozen: false}

	dcdtKey := append(transferFunc.keyPrefix, key...)
	dcdtToken := &dcdt.DCDigitalToken{Value: big.NewInt(100), Properties: dcdtFrozen.ToBytes()}
	marshaledData, _ := marshaller.Marshal(dcdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(dcdtKey, marshaledData)

	_, err := transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrDCDTIsFrozenForAccount)

	dcdtToken = &dcdt.DCDigitalToken{Value: big.NewInt(100), Properties: dcdtNotFrozen.ToBytes()}
	marshaledData, _ = marshaller.Marshal(dcdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(dcdtKey, marshaledData)

	dcdtToken = &dcdt.DCDigitalToken{Value: big.NewInt(100), Properties: dcdtFrozen.ToBytes()}
	marshaledData, _ = marshaller.Marshal(dcdtToken)
	_ = accDst.AccountDataHandler().SaveKeyValue(dcdtKey, marshaledData)

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrDCDTIsFrozenForAccount)

	marshaledData, _, _ = accDst.AccountDataHandler().RetrieveValue(dcdtKey)
	_ = marshaller.Unmarshal(dcdtToken, marshaledData)
	assert.True(t, dcdtToken.Value.Cmp(big.NewInt(100)) == 0)

	input.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)

	dcdtToken = &dcdt.DCDigitalToken{Value: big.NewInt(100), Properties: dcdtNotFrozen.ToBytes()}
	marshaledData, _ = marshaller.Marshal(dcdtToken)
	_ = accDst.AccountDataHandler().SaveKeyValue(dcdtKey, marshaledData)

	systemAccount := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	dcdtGlobal := DCDTGlobalMetadata{Paused: true}
	pauseKey := []byte(baseDCDTKeyPrefix + string(key))
	_ = systemAccount.AccountDataHandler().SaveKeyValue(pauseKey, dcdtGlobal.ToBytes())

	accountStub.LoadAccountCalled = func(address []byte) (vmcommon.AccountHandler, error) {
		if bytes.Equal(address, vmcommon.SystemAccountAddress) {
			return systemAccount, nil
		}
		return accDst, nil
	}

	input.ReturnCallAfterError = false
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrDCDTTokenIsPaused)

	input.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)
}

func TestDCDTTransfer_SndDstWithLimitedTransfer(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	accountStub := &mock.AccountsStub{}
	rolesHandler := &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			if bytes.Equal(action, []byte(core.DCDTRoleTransfer)) {
				return ErrActionNotAllowed
			}
			return nil
		},
	}
	dcdtGlobalSettingsFunc, _ := NewDCDTGlobalSettingsFunc(accountStub, marshaller, true, core.BuiltInFunctionDCDTSetLimitedTransfer, trueHandler)
	transferFunc, _ := NewDCDTTransferFunc(10, marshaller, dcdtGlobalSettingsFunc, &mock.ShardCoordinatorStub{}, rolesHandler, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == CheckCorrectTokenIDForTransferRoleFlag
		},
	})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))
	accDst := mock.NewUserAccount([]byte("dst"))

	dcdtKey := append(transferFunc.keyPrefix, key...)
	dcdtToken := &dcdt.DCDigitalToken{Value: big.NewInt(100)}
	marshaledData, _ := marshaller.Marshal(dcdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(dcdtKey, marshaledData)

	dcdtToken = &dcdt.DCDigitalToken{Value: big.NewInt(100)}
	marshaledData, _ = marshaller.Marshal(dcdtToken)
	_ = accDst.AccountDataHandler().SaveKeyValue(dcdtKey, marshaledData)

	systemAccount := mock.NewUserAccount(vmcommon.SystemAccountAddress)
	dcdtGlobal := DCDTGlobalMetadata{LimitedTransfer: true}
	pauseKey := []byte(baseDCDTKeyPrefix + string(key))
	_ = systemAccount.AccountDataHandler().SaveKeyValue(pauseKey, dcdtGlobal.ToBytes())

	accountStub.LoadAccountCalled = func(address []byte) (vmcommon.AccountHandler, error) {
		if bytes.Equal(address, vmcommon.SystemAccountAddress) {
			return systemAccount, nil
		}
		return accDst, nil
	}

	_, err := transferFunc.ProcessBuiltinFunction(nil, accDst, input)
	assert.Nil(t, err)

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Equal(t, err, ErrActionNotAllowed)

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrActionNotAllowed)

	input.ReturnCallAfterError = true
	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)

	input.ReturnCallAfterError = false
	rolesHandler.CheckAllowedToExecuteCalled = func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
		if bytes.Equal(account.AddressBytes(), accSnd.Address) && bytes.Equal(tokenID, key) {
			return nil
		}
		return ErrActionNotAllowed
	}

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)

	rolesHandler.CheckAllowedToExecuteCalled = func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
		if bytes.Equal(account.AddressBytes(), accDst.Address) && bytes.Equal(tokenID, key) {
			return nil
		}
		return ErrActionNotAllowed
	}

	_, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)
}

func TestDCDTTransfer_ProcessBuiltInFunctionOnAsyncCallBack(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	transferFunc, _ := NewDCDTTransferFunc(10, marshaller, &mock.GlobalSettingsHandlerStub{}, &mock.ShardCoordinatorStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == CheckCorrectTokenIDForTransferRoleFlag
		},
	})
	_ = transferFunc.SetPayableChecker(&mock.PayableHandlerStub{})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
			CallType:    vm.AsynchronousCallBack,
		},
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))
	accDst := mock.NewUserAccount(core.DCDTSCAddress)

	dcdtKey := append(transferFunc.keyPrefix, key...)
	dcdtToken := &dcdt.DCDigitalToken{Value: big.NewInt(100)}
	marshaledData, _ := marshaller.Marshal(dcdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(dcdtKey, marshaledData)

	vmOutput, err := transferFunc.ProcessBuiltinFunction(nil, accDst, input)
	assert.Nil(t, err)

	marshaledData, _, _ = accDst.AccountDataHandler().RetrieveValue(dcdtKey)
	_ = marshaller.Unmarshal(dcdtToken, marshaledData)
	assert.True(t, dcdtToken.Value.Cmp(big.NewInt(10)) == 0)

	assert.Equal(t, vmOutput.GasRemaining, input.GasProvided)

	vmOutput, err = transferFunc.ProcessBuiltinFunction(accSnd, accDst, input)
	assert.Nil(t, err)
	vmOutput.GasRemaining = input.GasProvided - transferFunc.funcGasCost

	marshaledData, _, _ = accSnd.AccountDataHandler().RetrieveValue(dcdtKey)
	_ = marshaller.Unmarshal(dcdtToken, marshaledData)
	assert.True(t, dcdtToken.Value.Cmp(big.NewInt(90)) == 0)
}
