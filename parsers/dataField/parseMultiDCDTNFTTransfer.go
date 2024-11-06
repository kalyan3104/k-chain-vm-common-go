package datafield

import (
	"github.com/DharitriOne/drt-chain-core-go/core"
	"github.com/DharitriOne/drt-chain-core-go/core/sharding"
)

func (odp *operationDataFieldParser) parseMultiDCDTNFTTransfer(args [][]byte, function string, sender, receiver []byte, numOfShards uint32) *ResponseParseData {
	responseParse, parsedDCDTTransfers, ok := odp.extractDCDTData(args, function, sender, receiver)
	if !ok {
		return responseParse
	}
	if core.IsSmartContractAddress(parsedDCDTTransfers.RcvAddr) && isASCIIString(parsedDCDTTransfers.CallFunction) {
		responseParse.Function = parsedDCDTTransfers.CallFunction
	}

	receiverShardID := sharding.ComputeShardID(parsedDCDTTransfers.RcvAddr, numOfShards)
	for _, dcdtTransferData := range parsedDCDTTransfers.DCDTTransfers {
		if !isASCIIString(string(dcdtTransferData.DCDTTokenName)) {
			return &ResponseParseData{
				Operation: function,
			}
		}

		token := string(dcdtTransferData.DCDTTokenName)
		if dcdtTransferData.DCDTTokenNonce != 0 {
			token = computeTokenIdentifier(token, dcdtTransferData.DCDTTokenNonce)
		}

		responseParse.Tokens = append(responseParse.Tokens, token)
		responseParse.DCDTValues = append(responseParse.DCDTValues, dcdtTransferData.DCDTValue.String())
		responseParse.Receivers = append(responseParse.Receivers, parsedDCDTTransfers.RcvAddr)
		responseParse.ReceiversShardID = append(responseParse.ReceiversShardID, receiverShardID)
	}

	return responseParse
}
