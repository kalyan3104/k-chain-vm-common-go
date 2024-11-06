package datafield

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDCDTTransfer(t *testing.T) {
	t.Parallel()

	args := createMockArgumentsOperationParser()
	parser, _ := NewOperationDataFieldParser(args)

	t.Run("TransferNonHexArguments", func(t *testing.T) {
		t.Parallel()

		dataField := []byte("DCDTTransfer@1234@011")
		res := parser.Parse(dataField, sender, receiver, 3)
		require.Equal(t, &ResponseParseData{
			Operation: OperationTransfer,
		}, res)
	})

	t.Run("TransferNotEnoughArguments", func(t *testing.T) {
		t.Parallel()

		dataField := []byte("DCDTTransfer@1234")
		res := parser.Parse(dataField, sender, receiver, 3)
		require.Equal(t, &ResponseParseData{
			Operation: "DCDTTransfer",
		}, res)
	})

	t.Run("TransferEmptyArguments", func(t *testing.T) {
		t.Parallel()

		dataField := []byte("DCDTTransfer@544f4b454e@")
		res := parser.Parse(dataField, sender, receiver, 3)
		require.Equal(t, &ResponseParseData{
			Operation:  "DCDTTransfer",
			Tokens:     []string{"TOKEN"},
			DCDTValues: []string{"0"},
		}, res)
	})

	t.Run("TransferWithSCCall", func(t *testing.T) {
		t.Parallel()

		dataField := []byte("DCDTTransfer@544f4b454e@01@63616c6c4d65")
		res := parser.Parse(dataField, sender, receiverSC, 3)
		require.Equal(t, &ResponseParseData{
			Operation:  "DCDTTransfer",
			Function:   "callMe",
			DCDTValues: []string{"1"},
			Tokens:     []string{"TOKEN"},
		}, res)
	})

	t.Run("TransferNonAsciiStringToken", func(t *testing.T) {
		dataField := []byte("DCDTTransfer@055de6a779bbac0000@01")
		res := parser.Parse(dataField, sender, receiverSC, 3)
		require.Equal(t, &ResponseParseData{
			Operation: "DCDTTransfer",
		}, res)
	})
}
