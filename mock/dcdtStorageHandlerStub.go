package mock

import (
	"math/big"

	"github.com/DharitriOne/drt-chain-core-go/data"
	"github.com/DharitriOne/drt-chain-core-go/data/dcdt"
	vmcommon "github.com/DharitriOne/drt-chain-vm-common-go"
)

// DCDTNFTStorageHandlerStub -
type DCDTNFTStorageHandlerStub struct {
	SaveDCDTNFTTokenCalled                                    func(senderAddress []byte, acnt vmcommon.UserAccountHandler, dcdtTokenKey []byte, nonce uint64, dcdtData *dcdt.DCDigitalToken, mustUpdateAllFields bool, isReturnWithError bool) ([]byte, error)
	GetDCDTNFTTokenOnSenderCalled                             func(acnt vmcommon.UserAccountHandler, dcdtTokenKey []byte, nonce uint64) (*dcdt.DCDigitalToken, error)
	GetDCDTNFTTokenOnDestinationCalled                        func(acnt vmcommon.UserAccountHandler, dcdtTokenKey []byte, nonce uint64) (*dcdt.DCDigitalToken, bool, error)
	GetDCDTNFTTokenOnDestinationWithCustomSystemAccountCalled func(accnt vmcommon.UserAccountHandler, dcdtTokenKey []byte, nonce uint64, systemAccount vmcommon.UserAccountHandler) (*dcdt.DCDigitalToken, bool, error)
	WasAlreadySentToDestinationShardAndUpdateStateCalled      func(tickerID []byte, nonce uint64, dstAddress []byte) (bool, error)
	SaveNFTMetaDataToSystemAccountCalled                      func(tx data.TransactionHandler) error
	AddToLiquiditySystemAccCalled                             func(dcdtTokenKey []byte, nonce uint64, transferValue *big.Int) error
}

// SaveDCDTNFTToken -
func (stub *DCDTNFTStorageHandlerStub) SaveDCDTNFTToken(senderAddress []byte, acnt vmcommon.UserAccountHandler, dcdtTokenKey []byte, nonce uint64, dcdtData *dcdt.DCDigitalToken, mustUpdateAllFields bool, isReturnWithError bool) ([]byte, error) {
	if stub.SaveDCDTNFTTokenCalled != nil {
		return stub.SaveDCDTNFTTokenCalled(senderAddress, acnt, dcdtTokenKey, nonce, dcdtData, mustUpdateAllFields, isReturnWithError)
	}
	return nil, nil
}

// GetDCDTNFTTokenOnSender -
func (stub *DCDTNFTStorageHandlerStub) GetDCDTNFTTokenOnSender(acnt vmcommon.UserAccountHandler, dcdtTokenKey []byte, nonce uint64) (*dcdt.DCDigitalToken, error) {
	if stub.GetDCDTNFTTokenOnSenderCalled != nil {
		return stub.GetDCDTNFTTokenOnSenderCalled(acnt, dcdtTokenKey, nonce)
	}
	return nil, nil
}

// GetDCDTNFTTokenOnDestination -
func (stub *DCDTNFTStorageHandlerStub) GetDCDTNFTTokenOnDestination(acnt vmcommon.UserAccountHandler, dcdtTokenKey []byte, nonce uint64) (*dcdt.DCDigitalToken, bool, error) {
	if stub.GetDCDTNFTTokenOnDestinationCalled != nil {
		return stub.GetDCDTNFTTokenOnDestinationCalled(acnt, dcdtTokenKey, nonce)
	}
	return nil, false, nil
}

// GetDCDTNFTTokenOnDestinationWithCustomSystemAccount -
func (stub *DCDTNFTStorageHandlerStub) GetDCDTNFTTokenOnDestinationWithCustomSystemAccount(accnt vmcommon.UserAccountHandler, dcdtTokenKey []byte, nonce uint64, systemAccount vmcommon.UserAccountHandler) (*dcdt.DCDigitalToken, bool, error) {
	if stub.GetDCDTNFTTokenOnDestinationWithCustomSystemAccountCalled != nil {
		return stub.GetDCDTNFTTokenOnDestinationWithCustomSystemAccountCalled(accnt, dcdtTokenKey, nonce, systemAccount)
	}
	return nil, false, nil
}

// WasAlreadySentToDestinationShardAndUpdateState -
func (stub *DCDTNFTStorageHandlerStub) WasAlreadySentToDestinationShardAndUpdateState(tickerID []byte, nonce uint64, dstAddress []byte) (bool, error) {
	if stub.WasAlreadySentToDestinationShardAndUpdateStateCalled != nil {
		return stub.WasAlreadySentToDestinationShardAndUpdateStateCalled(tickerID, nonce, dstAddress)
	}
	return false, nil
}

// SaveNFTMetaDataToSystemAccount -
func (stub *DCDTNFTStorageHandlerStub) SaveNFTMetaDataToSystemAccount(tx data.TransactionHandler) error {
	if stub.SaveNFTMetaDataToSystemAccountCalled != nil {
		return stub.SaveNFTMetaDataToSystemAccountCalled(tx)
	}
	return nil
}

// AddToLiquiditySystemAcc -
func (stub *DCDTNFTStorageHandlerStub) AddToLiquiditySystemAcc(dcdtTokenKey []byte, nonce uint64, transferValue *big.Int) error {
	if stub.AddToLiquiditySystemAccCalled != nil {
		return stub.AddToLiquiditySystemAccCalled(dcdtTokenKey, nonce, transferValue)
	}
	return nil
}

// IsInterfaceNil -
func (stub *DCDTNFTStorageHandlerStub) IsInterfaceNil() bool {
	return stub == nil
}
