package datafield

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"unicode"

	"github.com/kalyan3104/k-chain-core-go/core"
)

const (
	dcdtIdentifierSeparator  = "-"
	dcdtRandomSequenceLength = 6
)

// TODO refactor this part to use the built-in container for the list of all the built-in functions
func getAllBuiltInFunctions() []string {
	return []string{
		core.BuiltInFunctionClaimDeveloperRewards,
		core.BuiltInFunctionChangeOwnerAddress,
		core.BuiltInFunctionSetUserName,
		core.BuiltInFunctionSaveKeyValue,
		core.BuiltInFunctionDCDTTransfer,
		core.BuiltInFunctionDCDTBurn,
		core.BuiltInFunctionDCDTFreeze,
		core.BuiltInFunctionDCDTUnFreeze,
		core.BuiltInFunctionDCDTWipe,
		core.BuiltInFunctionDCDTPause,
		core.BuiltInFunctionDCDTUnPause,
		core.BuiltInFunctionSetDCDTRole,
		core.BuiltInFunctionUnSetDCDTRole,
		core.BuiltInFunctionDCDTSetLimitedTransfer,
		core.BuiltInFunctionDCDTUnSetLimitedTransfer,
		core.BuiltInFunctionDCDTLocalMint,
		core.BuiltInFunctionDCDTLocalBurn,
		core.BuiltInFunctionDCDTNFTTransfer,
		core.BuiltInFunctionDCDTNFTCreate,
		core.BuiltInFunctionDCDTNFTAddQuantity,
		core.BuiltInFunctionDCDTNFTCreateRoleTransfer,
		core.BuiltInFunctionDCDTNFTBurn,
		core.BuiltInFunctionDCDTNFTAddURI,
		core.BuiltInFunctionDCDTNFTUpdateAttributes,
		core.BuiltInFunctionMultiDCDTNFTTransfer,
		core.BuiltInFunctionMigrateDataTrie,
		core.DCDTRoleLocalMint,
		core.DCDTRoleLocalBurn,
		core.DCDTRoleNFTCreate,
		core.DCDTRoleNFTCreateMultiShard,
		core.DCDTRoleNFTAddQuantity,
		core.DCDTRoleNFTBurn,
		core.DCDTRoleNFTAddURI,
		core.DCDTRoleNFTUpdateAttributes,
		core.DCDTRoleTransfer,
		core.BuiltInFunctionSetGuardian,
		core.BuiltInFunctionUnGuardAccount,
		core.BuiltInFunctionGuardAccount,
	}
}

func isBuiltInFunction(builtInFunctionsList []string, function string) bool {
	for _, builtInFunction := range builtInFunctionsList {
		if builtInFunction == function {
			return true
		}
	}

	return false
}

func computeTokenIdentifier(token string, nonce uint64) string {
	if token == "" || nonce == 0 {
		return ""
	}

	nonceBig := big.NewInt(0).SetUint64(nonce)
	hexEncodedNonce := hex.EncodeToString(nonceBig.Bytes())
	return fmt.Sprintf("%s-%s", token, hexEncodedNonce)
}

func extractTokenAndNonce(arg []byte) (string, uint64) {
	argsSplit := bytes.Split(arg, []byte(dcdtIdentifierSeparator))
	if len(argsSplit) < 2 {
		return string(arg), 0
	}

	if len(argsSplit[1]) <= dcdtRandomSequenceLength {
		return string(arg), 0
	}

	identifier := []byte(fmt.Sprintf("%s-%s", argsSplit[0], argsSplit[1][:dcdtRandomSequenceLength]))
	nonce := big.NewInt(0).SetBytes(argsSplit[1][dcdtRandomSequenceLength:])

	return string(identifier), nonce.Uint64()
}

func isEmptyAddr(addrLength int, address []byte) bool {
	emptyAddr := make([]byte, addrLength)

	return bytes.Equal(address, emptyAddr)
}

func isASCIIString(input string) bool {
	for i := 0; i < len(input); i++ {
		if input[i] > unicode.MaxASCII {
			return false
		}
	}

	return true
}
