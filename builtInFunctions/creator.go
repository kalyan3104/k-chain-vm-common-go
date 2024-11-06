package builtInFunctions

import (
	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/core/check"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
	"github.com/mitchellh/mapstructure"
)

var _ vmcommon.BuiltInFunctionFactory = (*builtInFuncCreator)(nil)

var trueHandler = func() bool { return true }
var falseHandler = func() bool { return false }

const deleteUserNameFuncName = "DeleteUserName" // all builtInFunction names are upper case

// ArgsCreateBuiltInFunctionContainer defines the input arguments to create built in functions container
type ArgsCreateBuiltInFunctionContainer struct {
	GasMap                           map[string]map[string]uint64
	MapDNSAddresses                  map[string]struct{}
	MapDNSV2Addresses                map[string]struct{}
	EnableUserNameChange             bool
	Marshalizer                      vmcommon.Marshalizer
	Accounts                         vmcommon.AccountsAdapter
	ShardCoordinator                 vmcommon.Coordinator
	EnableEpochsHandler              vmcommon.EnableEpochsHandler
	GuardedAccountHandler            vmcommon.GuardedAccountHandler
	MaxNumOfAddressesForTransferRole uint32
	ConfigAddress                    []byte
}

type builtInFuncCreator struct {
	mapDNSAddresses                  map[string]struct{}
	mapDNSV2Addresses                map[string]struct{}
	enableUserNameChange             bool
	marshaller                       vmcommon.Marshalizer
	accounts                         vmcommon.AccountsAdapter
	builtInFunctions                 vmcommon.BuiltInFunctionContainer
	gasConfig                        *vmcommon.GasCost
	shardCoordinator                 vmcommon.Coordinator
	dcdtStorageHandler               vmcommon.DCDTNFTStorageHandler
	dcdtGlobalSettingsHandler        vmcommon.DCDTGlobalSettingsHandler
	enableEpochsHandler              vmcommon.EnableEpochsHandler
	guardedAccountHandler            vmcommon.GuardedAccountHandler
	maxNumOfAddressesForTransferRole uint32
	configAddress                    []byte
}

// NewBuiltInFunctionsCreator creates a component which will instantiate the built in functions contracts
func NewBuiltInFunctionsCreator(args ArgsCreateBuiltInFunctionContainer) (*builtInFuncCreator, error) {
	if check.IfNil(args.Marshalizer) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(args.Accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if args.MapDNSAddresses == nil {
		return nil, ErrNilDnsAddresses
	}
	if args.MapDNSV2Addresses == nil {
		return nil, ErrNilDnsAddresses
	}
	if check.IfNil(args.ShardCoordinator) {
		return nil, ErrNilShardCoordinator
	}
	if check.IfNil(args.EnableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}
	err := core.CheckHandlerCompatibility(args.EnableEpochsHandler, allFlags)
	if err != nil {
		return nil, err
	}
	if check.IfNil(args.GuardedAccountHandler) {
		return nil, ErrNilGuardedAccountHandler
	}

	b := &builtInFuncCreator{
		mapDNSAddresses:                  args.MapDNSAddresses,
		mapDNSV2Addresses:                args.MapDNSV2Addresses,
		enableUserNameChange:             args.EnableUserNameChange,
		marshaller:                       args.Marshalizer,
		accounts:                         args.Accounts,
		shardCoordinator:                 args.ShardCoordinator,
		enableEpochsHandler:              args.EnableEpochsHandler,
		guardedAccountHandler:            args.GuardedAccountHandler,
		maxNumOfAddressesForTransferRole: args.MaxNumOfAddressesForTransferRole,
		configAddress:                    args.ConfigAddress,
	}

	b.gasConfig, err = createGasConfig(args.GasMap)
	if err != nil {
		return nil, err
	}
	b.builtInFunctions = NewBuiltInFunctionContainer()

	return b, nil
}

// GasScheduleChange is called when gas schedule is changed, thus all contracts must be updated
func (b *builtInFuncCreator) GasScheduleChange(gasSchedule map[string]map[string]uint64) {
	newGasConfig, err := createGasConfig(gasSchedule)
	if err != nil {
		return
	}

	b.gasConfig = newGasConfig
	for key := range b.builtInFunctions.Keys() {
		builtInFunc, errGet := b.builtInFunctions.Get(key)
		if errGet != nil {
			return
		}

		builtInFunc.SetNewGasConfig(b.gasConfig)
	}
}

// NFTStorageHandler will return the dcdt storage handler from the built in functions factory
func (b *builtInFuncCreator) NFTStorageHandler() vmcommon.SimpleDCDTNFTStorageHandler {
	return b.dcdtStorageHandler
}

// DCDTGlobalSettingsHandler will return the dcdt global settings handler from the built in functions factory
func (b *builtInFuncCreator) DCDTGlobalSettingsHandler() vmcommon.DCDTGlobalSettingsHandler {
	return b.dcdtGlobalSettingsHandler
}

// BuiltInFunctionContainer will return the built in function container
func (b *builtInFuncCreator) BuiltInFunctionContainer() vmcommon.BuiltInFunctionContainer {
	return b.builtInFunctions
}

// CreateBuiltInFunctionContainer will create the list of built-in functions
func (b *builtInFuncCreator) CreateBuiltInFunctionContainer() error {

	b.builtInFunctions = NewBuiltInFunctionContainer()
	var newFunc vmcommon.BuiltinFunction
	newFunc = NewClaimDeveloperRewardsFunc(b.gasConfig.BuiltInCost.ClaimDeveloperRewards)
	err := b.builtInFunctions.Add(core.BuiltInFunctionClaimDeveloperRewards, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewChangeOwnerAddressFunc(b.gasConfig.BuiltInCost.ChangeOwnerAddress, b.enableEpochsHandler)
	if err != nil {
		return err
	}

	err = b.builtInFunctions.Add(core.BuiltInFunctionChangeOwnerAddress, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewSaveUserNameFunc(b.gasConfig.BuiltInCost.SaveUserName, b.mapDNSAddresses, b.mapDNSV2Addresses, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSetUserName, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDeleteUserNameFunc(b.gasConfig.BuiltInCost.SaveUserName, b.mapDNSV2Addresses, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(deleteUserNameFuncName, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewSaveKeyValueStorageFunc(b.gasConfig.BaseOperationCost, b.gasConfig.BuiltInCost.SaveKeyValue, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSaveKeyValue, newFunc)
	if err != nil {
		return err
	}

	globalSettingsFunc, err := NewDCDTGlobalSettingsFunc(
		b.accounts,
		b.marshaller,
		true,
		core.BuiltInFunctionDCDTPause,
		trueHandler,
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTPause, globalSettingsFunc)
	if err != nil {
		return err
	}
	b.dcdtGlobalSettingsHandler = globalSettingsFunc

	setRoleFunc, err := NewDCDTRolesFunc(b.marshaller, true)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSetDCDTRole, setRoleFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTTransferFunc(
		b.gasConfig.BuiltInCost.DCDTTransfer,
		b.marshaller,
		globalSettingsFunc,
		b.shardCoordinator,
		setRoleFunc,
		b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTBurnFunc(b.gasConfig.BuiltInCost.DCDTBurn, b.marshaller, globalSettingsFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTBurn, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTGlobalSettingsFunc(
		b.accounts,
		b.marshaller,
		false,
		core.BuiltInFunctionDCDTUnPause,
		trueHandler,
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTUnPause, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTRolesFunc(b.marshaller, false)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionUnSetDCDTRole, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTLocalBurnFunc(b.gasConfig.BuiltInCost.DCDTLocalBurn, b.marshaller, globalSettingsFunc, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTLocalBurn, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTLocalMintFunc(b.gasConfig.BuiltInCost.DCDTLocalMint, b.marshaller, globalSettingsFunc, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTLocalMint, newFunc)
	if err != nil {
		return err
	}

	args := ArgsNewDCDTDataStorage{
		Accounts:              b.accounts,
		GlobalSettingsHandler: globalSettingsFunc,
		Marshalizer:           b.marshaller,
		EnableEpochsHandler:   b.enableEpochsHandler,
		ShardCoordinator:      b.shardCoordinator,
	}
	b.dcdtStorageHandler, err = NewDCDTDataStorage(args)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTNFTAddQuantityFunc(b.gasConfig.BuiltInCost.DCDTNFTAddQuantity, b.dcdtStorageHandler, globalSettingsFunc, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTNFTAddQuantity, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTNFTBurnFunc(b.gasConfig.BuiltInCost.DCDTNFTBurn, b.dcdtStorageHandler, globalSettingsFunc, setRoleFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTNFTBurn, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTNFTCreateFunc(b.gasConfig.BuiltInCost.DCDTNFTCreate, b.gasConfig.BaseOperationCost, b.marshaller, globalSettingsFunc, setRoleFunc, b.dcdtStorageHandler, b.accounts, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTNFTCreate, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTFreezeWipeFunc(b.dcdtStorageHandler, b.enableEpochsHandler, b.marshaller, true, false)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTFreeze, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTFreezeWipeFunc(b.dcdtStorageHandler, b.enableEpochsHandler, b.marshaller, false, false)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTUnFreeze, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTFreezeWipeFunc(b.dcdtStorageHandler, b.enableEpochsHandler, b.marshaller, false, true)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTWipe, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTNFTTransferFunc(b.gasConfig.BuiltInCost.DCDTNFTTransfer,
		b.marshaller,
		globalSettingsFunc,
		b.accounts,
		b.shardCoordinator,
		b.gasConfig.BaseOperationCost,
		setRoleFunc,
		b.dcdtStorageHandler,
		b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTNFTTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTNFTCreateRoleTransfer(b.marshaller, b.accounts, b.shardCoordinator)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTNFTCreateRoleTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTNFTUpdateAttributesFunc(b.gasConfig.BuiltInCost.DCDTNFTUpdateAttributes, b.gasConfig.BaseOperationCost, b.dcdtStorageHandler, globalSettingsFunc, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTNFTUpdateAttributes, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTNFTAddUriFunc(b.gasConfig.BuiltInCost.DCDTNFTAddURI, b.gasConfig.BaseOperationCost, b.dcdtStorageHandler, globalSettingsFunc, setRoleFunc, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTNFTAddURI, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTNFTMultiTransferFunc(b.gasConfig.BuiltInCost.DCDTNFTMultiTransfer,
		b.marshaller,
		globalSettingsFunc,
		b.accounts,
		b.shardCoordinator,
		b.gasConfig.BaseOperationCost,
		b.enableEpochsHandler,
		setRoleFunc,
		b.dcdtStorageHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMultiDCDTNFTTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTGlobalSettingsFunc(
		b.accounts,
		b.marshaller,
		true,
		core.BuiltInFunctionDCDTSetLimitedTransfer,
		func() bool {
			return b.enableEpochsHandler.IsFlagEnabled(DCDTTransferRoleFlag)
		},
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTSetLimitedTransfer, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTGlobalSettingsFunc(
		b.accounts,
		b.marshaller,
		false,
		core.BuiltInFunctionDCDTUnSetLimitedTransfer,
		func() bool {
			return b.enableEpochsHandler.IsFlagEnabled(DCDTTransferRoleFlag)
		},
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionDCDTUnSetLimitedTransfer, newFunc)
	if err != nil {
		return err
	}

	argsNewDeleteFunc := ArgsNewDCDTDeleteMetadata{
		FuncGasCost:         b.gasConfig.BuiltInCost.DCDTNFTBurn,
		Marshalizer:         b.marshaller,
		Accounts:            b.accounts,
		AllowedAddress:      b.configAddress,
		Delete:              true,
		EnableEpochsHandler: b.enableEpochsHandler,
	}
	newFunc, err = NewDCDTDeleteMetadataFunc(argsNewDeleteFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.DCDTDeleteMetadata, newFunc)
	if err != nil {
		return err
	}

	argsNewDeleteFunc.Delete = false
	newFunc, err = NewDCDTDeleteMetadataFunc(argsNewDeleteFunc)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.DCDTAddMetadata, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTGlobalSettingsFunc(
		b.accounts,
		b.marshaller,
		true,
		vmcommon.BuiltInFunctionDCDTSetBurnRoleForAll,
		func() bool {
			return b.enableEpochsHandler.IsFlagEnabled(SendAlwaysFlag)
		},
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionDCDTSetBurnRoleForAll, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTGlobalSettingsFunc(
		b.accounts,
		b.marshaller,
		false,
		vmcommon.BuiltInFunctionDCDTUnSetBurnRoleForAll,
		func() bool {
			return b.enableEpochsHandler.IsFlagEnabled(SendAlwaysFlag)
		},
	)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionDCDTUnSetBurnRoleForAll, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTTransferRoleAddressFunc(b.accounts, b.marshaller, b.maxNumOfAddressesForTransferRole, false, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionDCDTTransferRoleDeleteAddress, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewDCDTTransferRoleAddressFunc(b.accounts, b.marshaller, b.maxNumOfAddressesForTransferRole, true, b.enableEpochsHandler)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(vmcommon.BuiltInFunctionDCDTTransferRoleAddAddress, newFunc)
	if err != nil {
		return err
	}

	argsSetGuardian := SetGuardianArgs{
		BaseAccountGuarderArgs: b.createBaseAccountGuarderArgs(b.gasConfig.BuiltInCost.SetGuardian),
	}
	newFunc, err = NewSetGuardianFunc(argsSetGuardian)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionSetGuardian, newFunc)
	if err != nil {
		return err
	}

	argsGuardAccount := b.createGuardAccountArgs()
	newFunc, err = NewGuardAccountFunc(argsGuardAccount)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionGuardAccount, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewUnGuardAccountFunc(argsGuardAccount)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionUnGuardAccount, newFunc)
	if err != nil {
		return err
	}

	newFunc, err = NewMigrateDataTrieFunc(b.gasConfig.BuiltInCost, b.enableEpochsHandler, b.accounts)
	if err != nil {
		return err
	}
	err = b.builtInFunctions.Add(core.BuiltInFunctionMigrateDataTrie, newFunc)
	if err != nil {
		return err
	}

	return nil
}

func (b *builtInFuncCreator) createBaseAccountGuarderArgs(funcGasCost uint64) BaseAccountGuarderArgs {
	return BaseAccountGuarderArgs{
		Marshaller:            b.marshaller,
		FuncGasCost:           funcGasCost,
		GuardedAccountHandler: b.guardedAccountHandler,
		EnableEpochsHandler:   b.enableEpochsHandler,
	}
}

func (b *builtInFuncCreator) createGuardAccountArgs() GuardAccountArgs {
	return GuardAccountArgs{
		BaseAccountGuarderArgs: b.createBaseAccountGuarderArgs(b.gasConfig.BuiltInCost.GuardAccount),
	}
}

func createGasConfig(gasMap map[string]map[string]uint64) (*vmcommon.GasCost, error) {
	baseOps := &vmcommon.BaseOperationCost{}
	err := mapstructure.Decode(gasMap[core.BaseOperationCostString], baseOps)
	if err != nil {
		return nil, err
	}

	err = check.ForZeroUintFields(*baseOps)
	if err != nil {
		return nil, err
	}

	builtInOps := &vmcommon.BuiltInCost{}
	err = mapstructure.Decode(gasMap[core.BuiltInCostString], builtInOps)
	if err != nil {
		return nil, err
	}

	err = check.ForZeroUintFields(*builtInOps)
	if err != nil {
		return nil, err
	}

	gasCost := vmcommon.GasCost{
		BaseOperationCost: *baseOps,
		BuiltInCost:       *builtInOps,
	}

	return &gasCost, nil
}

// SetPayableHandler sets the payableCheck interface to the needed functions
func (b *builtInFuncCreator) SetPayableHandler(payableHandler vmcommon.PayableHandler) error {
	payableChecker, err := NewPayableCheckFunc(
		payableHandler,
		b.enableEpochsHandler,
	)
	if err != nil {
		return err
	}

	listOfTransferFunc := []string{
		core.BuiltInFunctionMultiDCDTNFTTransfer,
		core.BuiltInFunctionDCDTNFTTransfer,
		core.BuiltInFunctionDCDTTransfer}

	for _, transferFunc := range listOfTransferFunc {
		builtInFunc, err := b.builtInFunctions.Get(transferFunc)
		if err != nil {
			return err
		}

		dcdtTransferFunc, ok := builtInFunc.(vmcommon.AcceptPayableChecker)
		if !ok {
			return ErrWrongTypeAssertion
		}

		err = dcdtTransferFunc.SetPayableChecker(payableChecker)
		if err != nil {
			return err
		}
	}

	return nil
}

// IsInterfaceNil returns true if underlying object is nil
func (b *builtInFuncCreator) IsInterfaceNil() bool {
	return b == nil
}
