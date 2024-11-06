package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/core/check"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
)

type dcdtFreezeWipe struct {
	baseAlwaysActiveHandler
	dcdtStorageHandler  vmcommon.DCDTNFTStorageHandler
	enableEpochsHandler vmcommon.EnableEpochsHandler
	marshaller          vmcommon.Marshalizer
	keyPrefix           []byte
	wipe                bool
	freeze              bool
}

// NewDCDTFreezeWipeFunc returns the dcdt freeze/un-freeze/wipe built-in function component
func NewDCDTFreezeWipeFunc(
	dcdtStorageHandler vmcommon.DCDTNFTStorageHandler,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
	marshaller vmcommon.Marshalizer,
	freeze bool,
	wipe bool,
) (*dcdtFreezeWipe, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(dcdtStorageHandler) {
		return nil, ErrNilDCDTNFTStorageHandler
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	e := &dcdtFreezeWipe{
		dcdtStorageHandler:  dcdtStorageHandler,
		enableEpochsHandler: enableEpochsHandler,
		marshaller:          marshaller,
		keyPrefix:           []byte(baseDCDTKeyPrefix),
		freeze:              freeze,
		wipe:                wipe,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dcdtFreezeWipe) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves DCDT transfer function call
func (e *dcdtFreezeWipe) ProcessBuiltinFunction(
	_, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != 1 {
		return nil, ErrInvalidArguments
	}
	if !bytes.Equal(vmInput.CallerAddr, core.DCDTSCAddress) {
		return nil, ErrAddressIsNotDCDTSystemSC
	}
	if check.IfNil(acntDst) {
		return nil, ErrNilUserAccount
	}

	dcdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)
	identifier, nonce := extractTokenIdentifierAndNonceDCDTWipe(vmInput.Arguments[0])

	var amount *big.Int
	var err error

	if e.wipe {
		amount, err = e.wipeIfApplicable(acntDst, dcdtTokenKey, identifier, nonce)
		if err != nil {
			return nil, err
		}

	} else {
		amount, err = e.toggleFreeze(acntDst, dcdtTokenKey)
		if err != nil {
			return nil, err
		}
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	addDCDTEntryInVMOutput(vmOutput, []byte(vmInput.Function), identifier, nonce, amount, vmInput.CallerAddr, acntDst.AddressBytes())

	return vmOutput, nil
}

func (e *dcdtFreezeWipe) wipeIfApplicable(acntDst vmcommon.UserAccountHandler, tokenKey []byte, identifier []byte, nonce uint64) (*big.Int, error) {
	tokenData, err := getDCDTDataFromKey(acntDst, tokenKey, e.marshaller)
	if err != nil {
		return nil, err
	}

	dcdtUserMetadata := DCDTUserMetadataFromBytes(tokenData.Properties)
	if !dcdtUserMetadata.Frozen {
		return nil, ErrCannotWipeAccountNotFrozen
	}

	err = acntDst.AccountDataHandler().SaveKeyValue(tokenKey, nil)
	if err != nil {
		return nil, err
	}

	err = e.removeLiquidity(identifier, nonce, tokenData.Value)
	if err != nil {
		return nil, err
	}

	wipedAmount := vmcommon.ZeroValueIfNil(tokenData.Value)
	return wipedAmount, nil
}

func (e *dcdtFreezeWipe) removeLiquidity(tokenIdentifier []byte, nonce uint64, value *big.Int) error {
	if !e.enableEpochsHandler.IsFlagEnabled(WipeSingleNFTLiquidityDecreaseFlag) {
		return nil
	}

	tokenIDKey := append(e.keyPrefix, tokenIdentifier...)
	return e.dcdtStorageHandler.AddToLiquiditySystemAcc(tokenIDKey, nonce, big.NewInt(0).Neg(value))
}

func (e *dcdtFreezeWipe) toggleFreeze(acntDst vmcommon.UserAccountHandler, tokenKey []byte) (*big.Int, error) {
	tokenData, err := getDCDTDataFromKey(acntDst, tokenKey, e.marshaller)
	if err != nil {
		return nil, err
	}

	dcdtUserMetadata := DCDTUserMetadataFromBytes(tokenData.Properties)
	dcdtUserMetadata.Frozen = e.freeze
	tokenData.Properties = dcdtUserMetadata.ToBytes()

	err = saveDCDTData(acntDst, tokenData, tokenKey, e.marshaller)
	if err != nil {
		return nil, err
	}

	frozenAmount := vmcommon.ZeroValueIfNil(tokenData.Value)
	return frozenAmount, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dcdtFreezeWipe) IsInterfaceNil() bool {
	return e == nil
}
