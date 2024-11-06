package builtInFunctions

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/kalyan3104/k-chain-core-go/core"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
	"github.com/stretchr/testify/require"
)

func TestNewEntryForNFT(t *testing.T) {
	t.Parallel()

	vmOutput := &vmcommon.VMOutput{}
	addDCDTEntryInVMOutput(vmOutput, []byte(core.BuiltInFunctionDCDTNFTCreate), []byte("my-token"), 5, big.NewInt(1), []byte("caller"), []byte("receiver"))
	require.Equal(t, &vmcommon.LogEntry{
		Identifier: []byte(core.BuiltInFunctionDCDTNFTCreate),
		Address:    []byte("caller"),
		Topics:     [][]byte{[]byte("my-token"), big.NewInt(0).SetUint64(5).Bytes(), big.NewInt(1).Bytes(), []byte("receiver")},
		Data:       nil,
	}, vmOutput.Logs[0])
}

func TestExtractTokenIdentifierAndNonceDCDTWipe(t *testing.T) {
	t.Parallel()

	hexArg := "534b4537592d37336262636404"
	args, _ := hex.DecodeString(hexArg)

	identifier, nonce := extractTokenIdentifierAndNonceDCDTWipe(args)
	require.Equal(t, uint64(4), nonce)
	require.Equal(t, []byte("SKE7Y-73bbcd"), identifier)

	hexArg = "57524557412d376662623930"
	args, _ = hex.DecodeString(hexArg)

	identifier, nonce = extractTokenIdentifierAndNonceDCDTWipe(args)
	require.Equal(t, uint64(0), nonce)
	require.Equal(t, []byte("WREWA-7fbb90"), identifier)
}
