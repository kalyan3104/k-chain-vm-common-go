package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/DharitriOne/drt-chain-core-go/core"
	"github.com/DharitriOne/drt-chain-core-go/core/check"
	"github.com/DharitriOne/drt-chain-core-go/data/dcdt"
	vmcommon "github.com/DharitriOne/drt-chain-vm-common-go"
	"github.com/DharitriOne/drt-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewDCDTBurnFunc(t *testing.T) {
	t.Parallel()

	t.Run("nil marshaller should error", func(t *testing.T) {
		t.Parallel()

		burnFunc, err := NewDCDTBurnFunc(10, nil, &mock.GlobalSettingsHandlerStub{}, &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == GlobalMintBurnFlag
			},
		})
		assert.Equal(t, ErrNilMarshalizer, err)
		assert.True(t, check.IfNil(burnFunc))
	})
	t.Run("nil enable epochs handler should error", func(t *testing.T) {
		t.Parallel()

		burnFunc, err := NewDCDTBurnFunc(10, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, nil)
		assert.Equal(t, ErrNilEnableEpochsHandler, err)
		assert.True(t, check.IfNil(burnFunc))
	})
	t.Run("should work", func(t *testing.T) {
		t.Parallel()

		burnFunc, err := NewDCDTBurnFunc(10, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.EnableEpochsHandlerStub{
			IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
				return flag == GlobalMintBurnFlag
			},
		})
		assert.Nil(t, err)
		assert.False(t, check.IfNil(burnFunc))
	})
}

func TestDCDTBurn_ProcessBuiltInFunctionErrors(t *testing.T) {
	t.Parallel()

	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{}
	burnFunc, _ := NewDCDTBurnFunc(10, &mock.MarshalizerMock{}, globalSettingsHandler, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == GlobalMintBurnFlag
		},
	})
	_, err := burnFunc.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = burnFunc.ProcessBuiltinFunction(nil, nil, input)
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
	_, err = burnFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrAddressIsNotDCDTSystemSC)

	input.RecipientAddr = core.DCDTSCAddress
	input.GasProvided = burnFunc.funcGasCost - 1
	accSnd := mock.NewUserAccount([]byte("dst"))
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrNotEnoughGas)

	_, err = burnFunc.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrNilUserAccount)

	globalSettingsHandler.IsPausedCalled = func(token []byte) bool {
		return true
	}
	input.GasProvided = burnFunc.funcGasCost
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrDCDTTokenIsPaused)
}

func TestDCDTBurn_ProcessBuiltInFunctionSenderBurns(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	globalSettingsHandler := &mock.GlobalSettingsHandlerStub{}
	burnFunc, _ := NewDCDTBurnFunc(10, marshaller, globalSettingsHandler, &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == GlobalMintBurnFlag
		},
	})

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(0),
		},
		RecipientAddr: core.DCDTSCAddress,
	}
	key := []byte("key")
	value := big.NewInt(10).Bytes()
	input.Arguments = [][]byte{key, value}
	accSnd := mock.NewUserAccount([]byte("snd"))

	dcdtFrozen := DCDTUserMetadata{Frozen: true}
	dcdtNotFrozen := DCDTUserMetadata{Frozen: false}

	dcdtKey := append(burnFunc.keyPrefix, key...)
	dcdtToken := &dcdt.DCDigitalToken{Value: big.NewInt(100), Properties: dcdtFrozen.ToBytes()}
	marshaledData, _ := marshaller.Marshal(dcdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(dcdtKey, marshaledData)

	_, err := burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrDCDTIsFrozenForAccount)

	globalSettingsHandler.IsPausedCalled = func(token []byte) bool {
		return true
	}
	dcdtToken = &dcdt.DCDigitalToken{Value: big.NewInt(100), Properties: dcdtNotFrozen.ToBytes()}
	marshaledData, _ = marshaller.Marshal(dcdtToken)
	_ = accSnd.AccountDataHandler().SaveKeyValue(dcdtKey, marshaledData)

	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrDCDTTokenIsPaused)

	globalSettingsHandler.IsPausedCalled = func(token []byte) bool {
		return false
	}
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Nil(t, err)

	marshaledData, _, _ = accSnd.AccountDataHandler().RetrieveValue(dcdtKey)
	_ = marshaller.Unmarshal(dcdtToken, marshaledData)
	assert.True(t, dcdtToken.Value.Cmp(big.NewInt(90)) == 0)

	value = big.NewInt(100).Bytes()
	input.Arguments = [][]byte{key, value}
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Equal(t, err, ErrInsufficientFunds)

	value = big.NewInt(90).Bytes()
	input.Arguments = [][]byte{key, value}
	_, err = burnFunc.ProcessBuiltinFunction(accSnd, nil, input)
	assert.Nil(t, err)

	marshaledData, _, _ = accSnd.AccountDataHandler().RetrieveValue(dcdtKey)
	assert.Equal(t, len(marshaledData), 0)
}
