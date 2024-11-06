package datafield

import (
	"github.com/kalyan3104/k-chain-core-go/core"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
)

func (odp *operationDataFieldParser) parseSingleDCDTTransfer(args [][]byte, function string, sender, receiver []byte) *ResponseParseData {
	responseParse, parsedDCDTTransfers, ok := odp.extractDCDTData(args, function, sender, receiver)
	if !ok {
		return responseParse
	}

	if core.IsSmartContractAddress(receiver) && isASCIIString(parsedDCDTTransfers.CallFunction) {
		responseParse.Function = parsedDCDTTransfers.CallFunction
	}

	if len(parsedDCDTTransfers.DCDTTransfers) == 0 || !isASCIIString(string(parsedDCDTTransfers.DCDTTransfers[0].DCDTTokenName)) {
		return responseParse
	}

	firstTransfer := parsedDCDTTransfers.DCDTTransfers[0]
	responseParse.Tokens = append(responseParse.Tokens, string(firstTransfer.DCDTTokenName))
	responseParse.DCDTValues = append(responseParse.DCDTValues, firstTransfer.DCDTValue.String())

	return responseParse
}

func (odp *operationDataFieldParser) extractDCDTData(args [][]byte, function string, sender, receiver []byte) (*ResponseParseData, *vmcommon.ParsedDCDTTransfers, bool) {
	responseParse := &ResponseParseData{
		Operation: function,
	}

	parsedDCDTTransfers, err := odp.dcdtTransferParser.ParseDCDTTransfers(sender, receiver, function, args)
	if err != nil {
		return responseParse, nil, false
	}

	return responseParse, parsedDCDTTransfers, true
}
