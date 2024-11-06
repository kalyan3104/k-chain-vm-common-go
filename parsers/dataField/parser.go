package datafield

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/core/check"
	"github.com/kalyan3104/k-chain-core-go/core/sharding"
	"github.com/kalyan3104/k-chain-core-go/data/transaction"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
	"github.com/kalyan3104/k-chain-vm-common-go/parsers"
)

const (
	// OperationTransfer is the const for the transfer operation
	OperationTransfer = `transfer`
	operationDeploy   = `scDeploy`

	minArgumentsQuantityOperationDCDT = 2
	minArgumentsQuantityOperationNFT  = 3
	numArgsRelayedV2                  = 4
	receiverAddressIndexRelayedV2     = 0
	dataFieldIndexRelayedV2           = 2

	argsTokenPosition                   = 0
	argsNoncePosition                   = 1
	argsValuePositionNonAndSemiFungible = 2
	argsValuePositionFungible           = 1
)

var errInvalidAddressLength = errors.New("invalid address length")

type operationDataFieldParser struct {
	builtInFunctionsList []string

	addressLength      int
	argsParser         vmcommon.CallArgsParser
	dcdtTransferParser vmcommon.DCDTTransferParser
}

// NewOperationDataFieldParser will return a new instance of operationDataFieldParser
func NewOperationDataFieldParser(args *ArgsOperationDataFieldParser) (*operationDataFieldParser, error) {
	if check.IfNil(args.Marshalizer) {
		return nil, core.ErrNilMarshalizer
	}
	if args.AddressLength == 0 {
		return nil, errInvalidAddressLength
	}

	argsParser := parsers.NewCallArgsParser()
	dcdtTransferParser, err := parsers.NewDCDTTransferParser(args.Marshalizer)
	if err != nil {
		return nil, err
	}

	return &operationDataFieldParser{
		argsParser:           argsParser,
		dcdtTransferParser:   dcdtTransferParser,
		addressLength:        args.AddressLength,
		builtInFunctionsList: getAllBuiltInFunctions(),
	}, nil
}

// Parse will parse the provided data field
func (odp *operationDataFieldParser) Parse(dataField []byte, sender, receiver []byte, numOfShards uint32) *ResponseParseData {
	return odp.parse(dataField, sender, receiver, false, numOfShards)
}

func (odp *operationDataFieldParser) parse(dataField []byte, sender, receiver []byte, ignoreRelayed bool, numOfShards uint32) *ResponseParseData {
	responseParse := &ResponseParseData{
		Operation: OperationTransfer,
	}

	isSCDeploy := len(dataField) > 0 && isEmptyAddr(odp.addressLength, receiver)
	if isSCDeploy {
		responseParse.Operation = operationDeploy
		return responseParse
	}

	function, args, err := odp.argsParser.ParseData(string(dataField))
	if err != nil {
		return responseParse
	}

	switch function {
	case core.BuiltInFunctionDCDTTransfer:
		return odp.parseSingleDCDTTransfer(args, function, sender, receiver)
	case core.BuiltInFunctionDCDTNFTTransfer:
		return odp.parseSingleDCDTNFTTransfer(args, function, sender, receiver, numOfShards)
	case core.BuiltInFunctionMultiDCDTNFTTransfer:
		return odp.parseMultiDCDTNFTTransfer(args, function, sender, receiver, numOfShards)
	case core.BuiltInFunctionDCDTLocalBurn, core.BuiltInFunctionDCDTLocalMint:
		return parseQuantityOperationDCDT(args, function)
	case core.BuiltInFunctionDCDTWipe, core.BuiltInFunctionDCDTFreeze, core.BuiltInFunctionDCDTUnFreeze:
		return parseBlockingOperationDCDT(args, function)
	case core.BuiltInFunctionDCDTNFTCreate, core.BuiltInFunctionDCDTNFTBurn, core.BuiltInFunctionDCDTNFTAddQuantity:
		return parseQuantityOperationNFT(args, function)
	case core.RelayedTransaction, core.RelayedTransactionV2:
		if ignoreRelayed {
			return NewResponseParseDataAsRelayed()
		}
		return odp.parseRelayed(function, args, receiver, numOfShards)
	}

	isBuiltInFunc := isBuiltInFunction(odp.builtInFunctionsList, function)
	if isBuiltInFunc {
		responseParse.Operation = function
	}

	if function != "" && core.IsSmartContractAddress(receiver) && isASCIIString(function) {
		responseParse.Function = function
	}

	return responseParse
}

func (odp *operationDataFieldParser) parseRelayed(function string, args [][]byte, receiver []byte, numOfShards uint32) *ResponseParseData {
	if len(args) == 0 {
		return &ResponseParseData{
			IsRelayed: true,
		}
	}

	tx, ok := extractInnerTx(function, args, receiver)
	if !ok {
		return &ResponseParseData{
			IsRelayed: true,
		}
	}

	res := odp.parse(tx.Data, tx.SndAddr, tx.RcvAddr, true, numOfShards)
	if res.IsRelayed {
		return &ResponseParseData{
			IsRelayed: true,
		}
	}

	receivers := [][]byte{tx.RcvAddr}
	receiversShardID := []uint32{sharding.ComputeShardID(tx.RcvAddr, numOfShards)}
	if res.Operation == core.BuiltInFunctionMultiDCDTNFTTransfer || res.Operation == core.BuiltInFunctionDCDTNFTTransfer {
		receivers = res.Receivers
		receiversShardID = res.ReceiversShardID
	}

	return &ResponseParseData{
		Operation:        res.Operation,
		Function:         res.Function,
		DCDTValues:       res.DCDTValues,
		Tokens:           res.Tokens,
		Receivers:        receivers,
		ReceiversShardID: receiversShardID,
		IsRelayed:        true,
	}
}

func extractInnerTx(function string, args [][]byte, receiver []byte) (*transaction.Transaction, bool) {
	tx := &transaction.Transaction{}

	if function == core.RelayedTransaction {
		err := json.Unmarshal(args[0], &tx)

		return tx, err == nil
	}

	if len(args) != numArgsRelayedV2 {
		return nil, false
	}

	// sender of the inner tx is the receiver of the relayed tx
	tx.SndAddr = receiver
	tx.RcvAddr = args[receiverAddressIndexRelayedV2]
	tx.Data = args[dataFieldIndexRelayedV2]

	return tx, true
}

func parseBlockingOperationDCDT(args [][]byte, funcName string) *ResponseParseData {
	responseData := &ResponseParseData{
		Operation: funcName,
	}

	if len(args) == 0 {
		return responseData
	}

	token, nonce := extractTokenAndNonce(args[argsTokenPosition])
	if !isASCIIString(token) {
		return responseData
	}

	if nonce != 0 {
		token = computeTokenIdentifier(token, nonce)
	}

	responseData.Tokens = append(responseData.Tokens, token)
	return responseData
}

func parseQuantityOperationDCDT(args [][]byte, funcName string) *ResponseParseData {
	responseData := &ResponseParseData{
		Operation: funcName,
	}

	if len(args) < minArgumentsQuantityOperationDCDT {
		return responseData
	}

	token := string(args[argsTokenPosition])
	if !isASCIIString(token) {
		return responseData
	}

	responseData.Tokens = append(responseData.Tokens, token)
	responseData.DCDTValues = append(responseData.DCDTValues, big.NewInt(0).SetBytes(args[argsValuePositionFungible]).String())

	return responseData
}

func parseQuantityOperationNFT(args [][]byte, funcName string) *ResponseParseData {
	responseData := &ResponseParseData{
		Operation: funcName,
	}

	if len(args) < minArgumentsQuantityOperationNFT {
		return responseData
	}

	token := string(args[argsTokenPosition])
	if !isASCIIString(token) {
		return responseData
	}

	nonce := big.NewInt(0).SetBytes(args[argsNoncePosition]).Uint64()
	tokenIdentifier := computeTokenIdentifier(token, nonce)

	value := big.NewInt(0).SetBytes(args[argsValuePositionNonAndSemiFungible]).String()
	if funcName == core.BuiltInFunctionDCDTNFTCreate {
		value = big.NewInt(0).SetBytes(args[argsValuePositionNonAndSemiFungible-1]).String()
		tokenIdentifier = token
	}

	responseData.DCDTValues = append(responseData.DCDTValues, value)
	responseData.Tokens = append(responseData.Tokens, tokenIdentifier)

	return responseData
}
