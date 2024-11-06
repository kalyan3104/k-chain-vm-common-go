package parsers

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/data/dcdt"
	"github.com/kalyan3104/k-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
)

var sndAddr = bytes.Repeat([]byte{1}, 32)
var dstAddr = bytes.Repeat([]byte{1}, 32)

func TestNewDCDTTransferParser(t *testing.T) {
	t.Parallel()

	dcdtParser, err := NewDCDTTransferParser(nil)
	assert.Nil(t, dcdtParser)
	assert.Equal(t, err, ErrNilMarshalizer)

	dcdtParser, err = NewDCDTTransferParser(&mock.MarshalizerMock{})
	assert.Nil(t, err)
	assert.False(t, dcdtParser.IsInterfaceNil())
}

func TestDcdtTransferParser_ParseDCDTTransfersWrongFunction(t *testing.T) {
	t.Parallel()

	dcdtParser, _ := NewDCDTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := dcdtParser.ParseDCDTTransfers(nil, nil, "some", nil)
	assert.Equal(t, err, ErrNotDCDTTransferInput)
	assert.Nil(t, parsedData)
}

func TestDcdtTransferParser_ParseSingleDCDTFunction(t *testing.T) {
	t.Parallel()

	dcdtParser, _ := NewDCDTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := dcdtParser.ParseDCDTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionDCDTTransfer,
		[][]byte{[]byte("one")},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = dcdtParser.ParseDCDTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionDCDTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.DCDTTransfers[0].DCDTValue.Uint64(), big.NewInt(10).Uint64())

	parsedData, err = dcdtParser.ParseDCDTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionDCDTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), []byte("function"), []byte("arg")},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")
}

func TestDcdtTransferParser_ParseSingleNFTTransfer(t *testing.T) {
	t.Parallel()

	dcdtParser, _ := NewDCDTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := dcdtParser.ParseDCDTTransfers(
		nil,
		dstAddr,
		core.BuiltInFunctionDCDTNFTTransfer,
		[][]byte{[]byte("one"), []byte("two")},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = dcdtParser.ParseDCDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionDCDTNFTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), big.NewInt(10).Bytes(), dstAddr},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.DCDTTransfers[0].DCDTValue.Uint64(), big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.DCDTTransfers[0].DCDTTokenNonce, big.NewInt(10).Uint64())

	parsedData, err = dcdtParser.ParseDCDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionDCDTNFTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), big.NewInt(10).Bytes(), dstAddr, []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")

	parsedData, err = dcdtParser.ParseDCDTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionDCDTNFTTransfer,
		[][]byte{[]byte("one"), big.NewInt(10).Bytes(), big.NewInt(10).Bytes(), dstAddr, []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.DCDTTransfers[0].DCDTValue.Uint64(), big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.DCDTTransfers[0].DCDTTokenNonce, big.NewInt(10).Uint64())
}

func TestDcdtTransferParser_ParseMultiNFTTransferTransferOne(t *testing.T) {
	t.Parallel()

	dcdtParser, _ := NewDCDTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := dcdtParser.ParseDCDTTransfers(
		nil,
		sndAddr,
		core.BuiltInFunctionMultiDCDTNFTTransfer,
		[][]byte{[]byte("one"), []byte("two")},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = dcdtParser.ParseDCDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiDCDTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes()},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = dcdtParser.ParseDCDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiDCDTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.DCDTTransfers[0].DCDTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.DCDTTransfers[0].DCDTTokenNonce, big.NewInt(10).Uint64())

	parsedData, err = dcdtParser.ParseDCDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiDCDTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes(), []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")

	dcdtData := &dcdt.DCDigitalToken{Value: big.NewInt(20)}
	marshaled, _ := dcdtParser.marshaller.Marshal(dcdtData)

	parsedData, err = dcdtParser.ParseDCDTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionMultiDCDTNFTTransfer,
		[][]byte{big.NewInt(1).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), marshaled, []byte("function"), []byte("arg")})
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCDTTransfers), 1)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.DCDTTransfers[0].DCDTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.DCDTTransfers[0].DCDTTokenNonce, big.NewInt(10).Uint64())
}

func TestDcdtTransferParser_ParseMultiNFTTransferTransferMore(t *testing.T) {
	t.Parallel()

	dcdtParser, _ := NewDCDTTransferParser(&mock.MarshalizerMock{})
	parsedData, err := dcdtParser.ParseDCDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiDCDTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Equal(t, err, ErrNotEnoughArguments)
	assert.Nil(t, parsedData)

	parsedData, err = dcdtParser.ParseDCDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiDCDTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes(), []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCDTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.DCDTTransfers[0].DCDTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.DCDTTransfers[0].DCDTTokenNonce, big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.DCDTTransfers[1].DCDTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.DCDTTransfers[1].DCDTTokenNonce, uint64(0))
	assert.Equal(t, parsedData.DCDTTransfers[1].DCDTTokenType, uint32(core.Fungible))

	parsedData, err = dcdtParser.ParseDCDTTransfers(
		sndAddr,
		sndAddr,
		core.BuiltInFunctionMultiDCDTNFTTransfer,
		[][]byte{dstAddr, big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), big.NewInt(20).Bytes(), []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes(), []byte("function"), []byte("arg")},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCDTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")

	dcdtData := &dcdt.DCDigitalToken{Value: big.NewInt(20)}
	marshaled, _ := dcdtParser.marshaller.Marshal(dcdtData)
	parsedData, err = dcdtParser.ParseDCDTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionMultiDCDTNFTTransfer,
		[][]byte{big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), marshaled, []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes()},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCDTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 0)
	assert.Equal(t, parsedData.RcvAddr, dstAddr)
	assert.Equal(t, parsedData.DCDTTransfers[0].DCDTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.DCDTTransfers[0].DCDTTokenNonce, big.NewInt(10).Uint64())
	assert.Equal(t, parsedData.DCDTTransfers[1].DCDTValue.Uint64(), big.NewInt(20).Uint64())
	assert.Equal(t, parsedData.DCDTTransfers[1].DCDTTokenNonce, uint64(0))
	assert.Equal(t, parsedData.DCDTTransfers[1].DCDTTokenType, uint32(core.Fungible))

	parsedData, err = dcdtParser.ParseDCDTTransfers(
		sndAddr,
		dstAddr,
		core.BuiltInFunctionMultiDCDTNFTTransfer,
		[][]byte{big.NewInt(2).Bytes(), []byte("tokenID"), big.NewInt(10).Bytes(), marshaled, []byte("tokenID"), big.NewInt(0).Bytes(), big.NewInt(20).Bytes(), []byte("function"), []byte("arg")},
	)
	assert.Nil(t, err)
	assert.Equal(t, len(parsedData.DCDTTransfers), 2)
	assert.Equal(t, len(parsedData.CallArgs), 1)
	assert.Equal(t, parsedData.CallFunction, "function")
}
