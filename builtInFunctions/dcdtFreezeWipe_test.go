package builtInFunctions

import (
	"math/big"
	"testing"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/data/dcdt"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
	"github.com/kalyan3104/k-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDCDTFreezeWipe_ProcessBuiltInFunctionErrors(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	freeze, _ := NewDCDTFreezeWipeFunc(createNewDCDTDataStorageHandler(), &mock.EnableEpochsHandlerStub{}, marshaller, true, false)
	_, err := freeze.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input = &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			GasProvided: 50,
			CallValue:   big.NewInt(1),
		},
	}
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrBuiltInFunctionCalledWithValue)

	input.CallValue = big.NewInt(0)
	key := []byte("key")
	value := []byte("value")
	input.Arguments = [][]byte{key, value}
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrInvalidArguments)

	input.Arguments = [][]byte{key}
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrAddressIsNotDCDTSystemSC)

	input.CallerAddr = core.DCDTSCAddress
	_, err = freeze.ProcessBuiltinFunction(nil, nil, input)
	assert.Equal(t, err, ErrNilUserAccount)

	input.RecipientAddr = []byte("dst")
	acnt := mock.NewUserAccount(input.RecipientAddr)
	vmOutput, err := freeze.ProcessBuiltinFunction(nil, acnt, input)
	assert.Nil(t, err)

	frozenAmount := big.NewInt(42)
	dcdtToken := &dcdt.DCDigitalToken{
		Value: frozenAmount,
	}
	dcdtKey := append(freeze.keyPrefix, key...)
	marshaledData, _, _ := acnt.AccountDataHandler().RetrieveValue(dcdtKey)
	_ = marshaller.Unmarshal(dcdtToken, marshaledData)

	dcdtUserData := DCDTUserMetadataFromBytes(dcdtToken.Properties)
	assert.True(t, dcdtUserData.Frozen)
	assert.Len(t, vmOutput.Logs, 1)
	assert.Equal(t, [][]byte{key, {}, frozenAmount.Bytes(), []byte("dst")}, vmOutput.Logs[0].Topics)
}

func TestDCDTFreezeWipe_ProcessBuiltInFunction(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	freeze, _ := NewDCDTFreezeWipeFunc(createNewDCDTDataStorageHandler(), &mock.EnableEpochsHandlerStub{}, marshaller, true, false)
	_, err := freeze.ProcessBuiltinFunction(nil, nil, nil)
	assert.Equal(t, err, ErrNilVmInput)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	key := []byte("key")

	input.Arguments = [][]byte{key}
	input.CallerAddr = core.DCDTSCAddress
	input.RecipientAddr = []byte("dst")
	dcdtKey := append(freeze.keyPrefix, key...)
	dcdtToken := &dcdt.DCDigitalToken{Value: big.NewInt(10)}
	marshaledData, _ := freeze.marshaller.Marshal(dcdtToken)
	acnt := mock.NewUserAccount(input.RecipientAddr)
	_ = acnt.AccountDataHandler().SaveKeyValue(dcdtKey, marshaledData)

	_, err = freeze.ProcessBuiltinFunction(nil, acnt, input)
	assert.Nil(t, err)

	dcdtToken = &dcdt.DCDigitalToken{}
	marshaledData, _, _ = acnt.AccountDataHandler().RetrieveValue(dcdtKey)
	_ = marshaller.Unmarshal(dcdtToken, marshaledData)

	dcdtUserData := DCDTUserMetadataFromBytes(dcdtToken.Properties)
	assert.True(t, dcdtUserData.Frozen)

	unFreeze, _ := NewDCDTFreezeWipeFunc(createNewDCDTDataStorageHandler(), &mock.EnableEpochsHandlerStub{}, marshaller, false, false)
	_, err = unFreeze.ProcessBuiltinFunction(nil, acnt, input)
	assert.Nil(t, err)

	marshaledData, _, _ = acnt.AccountDataHandler().RetrieveValue(dcdtKey)
	_ = marshaller.Unmarshal(dcdtToken, marshaledData)

	dcdtUserData = DCDTUserMetadataFromBytes(dcdtToken.Properties)
	assert.False(t, dcdtUserData.Frozen)

	// cannot wipe if account is not frozen
	wipe, _ := NewDCDTFreezeWipeFunc(createNewDCDTDataStorageHandler(), &mock.EnableEpochsHandlerStub{}, marshaller, false, true)
	_, err = wipe.ProcessBuiltinFunction(nil, acnt, input)
	assert.Equal(t, ErrCannotWipeAccountNotFrozen, err)

	marshaledData, _, _ = acnt.AccountDataHandler().RetrieveValue(dcdtKey)
	assert.NotEqual(t, 0, len(marshaledData))

	// can wipe as account is frozen
	metaData := DCDTUserMetadata{Frozen: true}
	wipedAmount := big.NewInt(42)
	dcdtToken = &dcdt.DCDigitalToken{
		Value:      wipedAmount,
		Properties: metaData.ToBytes(),
	}
	dcdtTokenBytes, _ := marshaller.Marshal(dcdtToken)
	err = acnt.AccountDataHandler().SaveKeyValue(dcdtKey, dcdtTokenBytes)
	assert.NoError(t, err)

	wipe, _ = NewDCDTFreezeWipeFunc(createNewDCDTDataStorageHandler(), &mock.EnableEpochsHandlerStub{}, marshaller, false, true)
	vmOutput, err := wipe.ProcessBuiltinFunction(nil, acnt, input)
	assert.NoError(t, err)

	marshaledData, _, _ = acnt.AccountDataHandler().RetrieveValue(dcdtKey)
	assert.Equal(t, 0, len(marshaledData))
	assert.Len(t, vmOutput.Logs, 1)
	assert.Equal(t, [][]byte{key, {}, wipedAmount.Bytes(), []byte("dst")}, vmOutput.Logs[0].Topics)
}

func TestDcdtFreezeWipe_WipeShouldDecreaseLiquidityIfFlagIsEnabled(t *testing.T) {
	t.Parallel()

	balance := big.NewInt(37)
	addToLiquiditySystemAccCalled := false
	dcdtStorage := &mock.DCDTNFTStorageHandlerStub{
		AddToLiquiditySystemAccCalled: func(_ []byte, _ uint64, transferValue *big.Int) error {
			require.Equal(t, big.NewInt(0).Neg(balance), transferValue)
			addToLiquiditySystemAccCalled = true
			return nil
		},
	}

	marshaller := &mock.MarshalizerMock{}
	wipe, _ := NewDCDTFreezeWipeFunc(dcdtStorage, &mock.EnableEpochsHandlerStub{}, marshaller, false, true)

	acnt := mock.NewUserAccount([]byte("dst"))
	metaData := DCDTUserMetadata{Frozen: true}
	dcdtToken := &dcdt.DCDigitalToken{
		Value:      balance,
		Properties: metaData.ToBytes(),
	}
	dcdtTokenBytes, _ := marshaller.Marshal(dcdtToken)

	nonce := uint64(37)
	key := append([]byte("MYSFT-0a0a0a"), big.NewInt(int64(nonce)).Bytes()...)
	dcdtKey := append(wipe.keyPrefix, key...)

	err := acnt.AccountDataHandler().SaveKeyValue(dcdtKey, dcdtTokenBytes)
	assert.NoError(t, err)

	input := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
		},
	}
	input.Arguments = [][]byte{key}
	input.CallerAddr = core.DCDTSCAddress
	input.RecipientAddr = []byte("dst")

	acntCopy := acnt.Clone()
	_, err = wipe.ProcessBuiltinFunction(nil, acntCopy, input)
	assert.NoError(t, err)

	marshaledData, _, _ := acntCopy.AccountDataHandler().RetrieveValue(dcdtKey)
	assert.Equal(t, 0, len(marshaledData))
	assert.False(t, addToLiquiditySystemAccCalled)

	wipe.enableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == WipeSingleNFTLiquidityDecreaseFlag
		},
	}

	_, err = wipe.ProcessBuiltinFunction(nil, acnt, input)
	assert.NoError(t, err)

	marshaledData, _, _ = acnt.AccountDataHandler().RetrieveValue(dcdtKey)
	assert.Equal(t, 0, len(marshaledData))
	assert.True(t, addToLiquiditySystemAccCalled)
}
