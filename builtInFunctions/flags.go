package builtInFunctions

import "github.com/DharitriOne/drt-chain-core-go/core"

// Enable epoch flags definitions
const (
	GlobalMintBurnFlag                          core.EnableEpochFlag = "GlobalMintBurnFlag"
	DCDTTransferRoleFlag                        core.EnableEpochFlag = "DCDTTransferRoleFlag"
	CheckFunctionArgumentFlag                   core.EnableEpochFlag = "CheckFunctionArgumentFlag"
	CheckCorrectTokenIDForTransferRoleFlag      core.EnableEpochFlag = "CheckCorrectTokenIDForTransferRoleFlag"
	FixAsyncCallbackCheckFlag                   core.EnableEpochFlag = "FixAsyncCallbackCheckFlag"
	SaveToSystemAccountFlag                     core.EnableEpochFlag = "SaveToSystemAccountFlag"
	CheckFrozenCollectionFlag                   core.EnableEpochFlag = "CheckFrozenCollectionFlag"
	SendAlwaysFlag                              core.EnableEpochFlag = "SendAlwaysFlag"
	ValueLengthCheckFlag                        core.EnableEpochFlag = "ValueLengthCheckFlag"
	CheckTransferFlag                           core.EnableEpochFlag = "CheckTransferFlag"
	DCDTNFTImprovementV1Flag                    core.EnableEpochFlag = "DCDTNFTImprovementV1Flag"
	FixOldTokenLiquidityFlag                    core.EnableEpochFlag = "FixOldTokenLiquidityFlag"
	WipeSingleNFTLiquidityDecreaseFlag          core.EnableEpochFlag = "WipeSingleNFTLiquidityDecreaseFlag"
	AlwaysSaveTokenMetaDataFlag                 core.EnableEpochFlag = "AlwaysSaveTokenMetaDataFlag"
	SetGuardianFlag                             core.EnableEpochFlag = "SetGuardianFlag"
	ConsistentTokensValuesLengthCheckFlag       core.EnableEpochFlag = "ConsistentTokensValuesLengthCheckFlag"
	ChangeUsernameFlag                          core.EnableEpochFlag = "ChangeUsernameFlag"
	AutoBalanceDataTriesFlag                    core.EnableEpochFlag = "AutoBalanceDataTriesFlag"
	ScToScLogEventFlag                          core.EnableEpochFlag = "ScToScLogEventFlag"
	FixGasRemainingForSaveKeyValueFlag          core.EnableEpochFlag = "FixGasRemainingForSaveKeyValueFlag"
	IsChangeOwnerAddressCrossShardThroughSCFlag core.EnableEpochFlag = "IsChangeOwnerAddressCrossShardThroughSCFlag"
	MigrateDataTrieFlag                         core.EnableEpochFlag = "MigrateDataTrieFlag"
)

// allFlags must have all flags used by drt-chain-vm-common-go in the current version
var allFlags = []core.EnableEpochFlag{
	GlobalMintBurnFlag,
	DCDTTransferRoleFlag,
	CheckFunctionArgumentFlag,
	CheckCorrectTokenIDForTransferRoleFlag,
	FixAsyncCallbackCheckFlag,
	SaveToSystemAccountFlag,
	CheckFrozenCollectionFlag,
	SendAlwaysFlag,
	ValueLengthCheckFlag,
	CheckTransferFlag,
	DCDTNFTImprovementV1Flag,
	FixOldTokenLiquidityFlag,
	WipeSingleNFTLiquidityDecreaseFlag,
	AlwaysSaveTokenMetaDataFlag,
	SetGuardianFlag,
	ConsistentTokensValuesLengthCheckFlag,
	ChangeUsernameFlag,
	AutoBalanceDataTriesFlag,
	ScToScLogEventFlag,
	FixGasRemainingForSaveKeyValueFlag,
	IsChangeOwnerAddressCrossShardThroughSCFlag,
	MigrateDataTrieFlag,
}
