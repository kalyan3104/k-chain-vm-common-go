package datafield

import (
	"bytes"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/core/sharding"
)

func (odp *operationDataFieldParser) parseSingleDCDTNFTTransfer(args [][]byte, function string, sender, receiver []byte, numOfShards uint32) *ResponseParseData {
	responseParse, parsedDCDTTransfers, ok := odp.extractDCDTData(args, function, sender, receiver)
	if !ok {
		return responseParse
	}

	if core.IsSmartContractAddress(parsedDCDTTransfers.RcvAddr) && isASCIIString(parsedDCDTTransfers.CallFunction) {
		responseParse.Function = parsedDCDTTransfers.CallFunction
	}

	if len(parsedDCDTTransfers.DCDTTransfers) == 0 || !isASCIIString(string(parsedDCDTTransfers.DCDTTransfers[0].DCDTTokenName)) {
		return responseParse
	}

	rcvAddr := receiver
	if bytes.Equal(sender, receiver) {
		rcvAddr = parsedDCDTTransfers.RcvAddr
	}

	dcdtNFTTransfer := parsedDCDTTransfers.DCDTTransfers[0]
	receiverShardID := sharding.ComputeShardID(rcvAddr, numOfShards)
	token := computeTokenIdentifier(string(dcdtNFTTransfer.DCDTTokenName), dcdtNFTTransfer.DCDTTokenNonce)

	responseParse.Tokens = append(responseParse.Tokens, token)
	responseParse.DCDTValues = append(responseParse.DCDTValues, dcdtNFTTransfer.DCDTValue.String())

	if len(rcvAddr) != len(sender) {
		return responseParse
	}

	responseParse.Receivers = append(responseParse.Receivers, rcvAddr)
	responseParse.ReceiversShardID = append(responseParse.ReceiversShardID, receiverShardID)

	return responseParse
}
