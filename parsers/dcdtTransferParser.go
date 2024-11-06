package parsers

import (
	"bytes"
	"math/big"

	"github.com/DharitriOne/drt-chain-core-go/core"
	"github.com/DharitriOne/drt-chain-core-go/core/check"
	"github.com/DharitriOne/drt-chain-core-go/data/dcdt"
	vmcommon "github.com/DharitriOne/drt-chain-vm-common-go"
)

// MinArgsForDCDTTransfer defines the minimum arguments needed for an dcdt transfer
const MinArgsForDCDTTransfer = 2

// MinArgsForDCDTNFTTransfer defines the minimum arguments needed for an nft transfer
const MinArgsForDCDTNFTTransfer = 4

// MinArgsForMultiDCDTNFTTransfer defines the minimum arguments needed for a multi transfer
const MinArgsForMultiDCDTNFTTransfer = 4

// ArgsPerTransfer defines the number of arguments per transfer in multi transfer
const ArgsPerTransfer = 3

type dcdtTransferParser struct {
	marshaller vmcommon.Marshalizer
}

// NewDCDTTransferParser creates a new dcdt transfer parser
func NewDCDTTransferParser(
	marshaller vmcommon.Marshalizer,
) (*dcdtTransferParser, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}

	return &dcdtTransferParser{marshaller: marshaller}, nil
}

// ParseDCDTTransfers returns the list of dcdt transfers, the callFunction and callArgs from the given arguments
func (e *dcdtTransferParser) ParseDCDTTransfers(
	sndAddr []byte,
	rcvAddr []byte,
	function string,
	args [][]byte,
) (*vmcommon.ParsedDCDTTransfers, error) {
	switch function {
	case core.BuiltInFunctionDCDTTransfer:
		return e.parseSingleDCDTTransfer(rcvAddr, args)
	case core.BuiltInFunctionDCDTNFTTransfer:
		return e.parseSingleDCDTNFTTransfer(sndAddr, rcvAddr, args)
	case core.BuiltInFunctionMultiDCDTNFTTransfer:
		return e.parseMultiDCDTNFTTransfer(rcvAddr, args)
	default:
		return nil, ErrNotDCDTTransferInput
	}
}

func (e *dcdtTransferParser) parseSingleDCDTTransfer(rcvAddr []byte, args [][]byte) (*vmcommon.ParsedDCDTTransfers, error) {
	if len(args) < MinArgsForDCDTTransfer {
		return nil, ErrNotEnoughArguments
	}
	dcdtTransfers := &vmcommon.ParsedDCDTTransfers{
		DCDTTransfers: make([]*vmcommon.DCDTTransfer, 1),
		RcvAddr:       rcvAddr,
		CallArgs:      make([][]byte, 0),
		CallFunction:  "",
	}
	if len(args) > MinArgsForDCDTTransfer {
		dcdtTransfers.CallFunction = string(args[MinArgsForDCDTTransfer])
	}
	if len(args) > MinArgsForDCDTTransfer+1 {
		dcdtTransfers.CallArgs = append(dcdtTransfers.CallArgs, args[MinArgsForDCDTTransfer+1:]...)
	}
	dcdtTransfers.DCDTTransfers[0] = &vmcommon.DCDTTransfer{
		DCDTValue:      big.NewInt(0).SetBytes(args[1]),
		DCDTTokenName:  args[0],
		DCDTTokenType:  uint32(core.Fungible),
		DCDTTokenNonce: 0,
	}

	return dcdtTransfers, nil
}

func (e *dcdtTransferParser) parseSingleDCDTNFTTransfer(sndAddr, rcvAddr []byte, args [][]byte) (*vmcommon.ParsedDCDTTransfers, error) {
	if len(args) < MinArgsForDCDTNFTTransfer {
		return nil, ErrNotEnoughArguments
	}
	dcdtTransfers := &vmcommon.ParsedDCDTTransfers{
		DCDTTransfers: make([]*vmcommon.DCDTTransfer, 1),
		RcvAddr:       rcvAddr,
		CallArgs:      make([][]byte, 0),
		CallFunction:  "",
	}

	if bytes.Equal(sndAddr, rcvAddr) {
		dcdtTransfers.RcvAddr = args[3]
	}
	if len(args) > MinArgsForDCDTNFTTransfer {
		dcdtTransfers.CallFunction = string(args[MinArgsForDCDTNFTTransfer])
	}
	if len(args) > MinArgsForDCDTNFTTransfer+1 {
		dcdtTransfers.CallArgs = append(dcdtTransfers.CallArgs, args[MinArgsForDCDTNFTTransfer+1:]...)
	}
	dcdtTransfers.DCDTTransfers[0] = &vmcommon.DCDTTransfer{
		DCDTValue:      big.NewInt(0).SetBytes(args[2]),
		DCDTTokenName:  args[0],
		DCDTTokenType:  uint32(core.NonFungible),
		DCDTTokenNonce: big.NewInt(0).SetBytes(args[1]).Uint64(),
	}

	return dcdtTransfers, nil
}

func (e *dcdtTransferParser) parseMultiDCDTNFTTransfer(rcvAddr []byte, args [][]byte) (*vmcommon.ParsedDCDTTransfers, error) {
	if len(args) < MinArgsForMultiDCDTNFTTransfer {
		return nil, ErrNotEnoughArguments
	}
	dcdtTransfers := &vmcommon.ParsedDCDTTransfers{
		RcvAddr:      rcvAddr,
		CallArgs:     make([][]byte, 0),
		CallFunction: "",
	}

	numOfTransfer := big.NewInt(0).SetBytes(args[0])
	startIndex := uint64(1)
	isTxAtSender := false

	isFirstArgumentAnAddress := len(args[0]) == len(rcvAddr) && !numOfTransfer.IsUint64()
	if isFirstArgumentAnAddress {
		dcdtTransfers.RcvAddr = args[0]
		numOfTransfer.SetBytes(args[1])
		startIndex = 2
		isTxAtSender = true
	}

	minLenArgs := ArgsPerTransfer*numOfTransfer.Uint64() + startIndex
	if uint64(len(args)) < minLenArgs {
		return nil, ErrNotEnoughArguments
	}

	if uint64(len(args)) > minLenArgs {
		dcdtTransfers.CallFunction = string(args[minLenArgs])
	}
	if uint64(len(args)) > minLenArgs+1 {
		dcdtTransfers.CallArgs = append(dcdtTransfers.CallArgs, args[minLenArgs+1:]...)
	}

	var err error
	dcdtTransfers.DCDTTransfers = make([]*vmcommon.DCDTTransfer, numOfTransfer.Uint64())
	for i := uint64(0); i < numOfTransfer.Uint64(); i++ {
		tokenStartIndex := startIndex + i*ArgsPerTransfer
		dcdtTransfers.DCDTTransfers[i], err = e.createNewDCDTTransfer(tokenStartIndex, args, isTxAtSender)
		if err != nil {
			return nil, err
		}
	}

	return dcdtTransfers, nil
}

func (e *dcdtTransferParser) createNewDCDTTransfer(
	tokenStartIndex uint64,
	args [][]byte,
	isTxAtSender bool,
) (*vmcommon.DCDTTransfer, error) {
	dcdtTransfer := &vmcommon.DCDTTransfer{
		DCDTValue:      big.NewInt(0).SetBytes(args[tokenStartIndex+2]),
		DCDTTokenName:  args[tokenStartIndex],
		DCDTTokenType:  uint32(core.Fungible),
		DCDTTokenNonce: big.NewInt(0).SetBytes(args[tokenStartIndex+1]).Uint64(),
	}
	if dcdtTransfer.DCDTTokenNonce > 0 {
		dcdtTransfer.DCDTTokenType = uint32(core.NonFungible)

		if !isTxAtSender && len(args[tokenStartIndex+2]) > vmcommon.MaxLengthForValueToOptTransfer {
			transferDCDTData := &dcdt.DCDigitalToken{}
			err := e.marshaller.Unmarshal(transferDCDTData, args[tokenStartIndex+2])
			if err != nil {
				return nil, err
			}
			dcdtTransfer.DCDTValue.Set(transferDCDTData.Value)
		}
	}

	return dcdtTransfer, nil
}

// IsInterfaceNil returns true if underlying object is nil
func (e *dcdtTransferParser) IsInterfaceNil() bool {
	return e == nil
}
