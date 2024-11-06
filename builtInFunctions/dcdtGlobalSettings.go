package builtInFunctions

import (
	"bytes"

	"github.com/DharitriOne/drt-chain-core-go/core"
	"github.com/DharitriOne/drt-chain-core-go/core/check"
	"github.com/DharitriOne/drt-chain-core-go/marshal"
	vmcommon "github.com/DharitriOne/drt-chain-vm-common-go"
)

type dcdtGlobalSettings struct {
	baseActiveHandler
	keyPrefix  []byte
	set        bool
	accounts   vmcommon.AccountsAdapter
	marshaller marshal.Marshalizer
	function   string
}

// NewDCDTGlobalSettingsFunc returns the dcdt pause/un-pause built-in function component
func NewDCDTGlobalSettingsFunc(
	accounts vmcommon.AccountsAdapter,
	marshaller marshal.Marshalizer,
	set bool,
	function string,
	activeHandler func() bool,
) (*dcdtGlobalSettings, error) {
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if activeHandler == nil {
		return nil, ErrNilActiveHandler
	}
	if !isCorrectFunction(function) {
		return nil, ErrInvalidArguments
	}

	e := &dcdtGlobalSettings{
		keyPrefix:  []byte(baseDCDTKeyPrefix),
		set:        set,
		accounts:   accounts,
		marshaller: marshaller,
		function:   function,
	}

	e.baseActiveHandler.activeHandler = activeHandler

	return e, nil
}

func isCorrectFunction(function string) bool {
	switch function {
	case core.BuiltInFunctionDCDTPause, core.BuiltInFunctionDCDTUnPause, core.BuiltInFunctionDCDTSetLimitedTransfer, core.BuiltInFunctionDCDTUnSetLimitedTransfer:
		return true
	case vmcommon.BuiltInFunctionDCDTSetBurnRoleForAll, vmcommon.BuiltInFunctionDCDTUnSetBurnRoleForAll:
		return true
	default:
		return false
	}
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dcdtGlobalSettings) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves DCDT pause function call
func (e *dcdtGlobalSettings) ProcessBuiltinFunction(
	_, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	if vmInput == nil {
		return nil, ErrNilVmInput
	}
	if vmInput.CallValue.Cmp(zero) != 0 {
		return nil, ErrBuiltInFunctionCalledWithValue
	}
	if len(vmInput.Arguments) != 1 {
		return nil, ErrInvalidArguments
	}
	if !bytes.Equal(vmInput.CallerAddr, core.DCDTSCAddress) {
		return nil, ErrAddressIsNotDCDTSystemSC
	}
	if !vmcommon.IsSystemAccountAddress(vmInput.RecipientAddr) {
		return nil, ErrOnlySystemAccountAccepted
	}

	dcdtTokenKey := append(e.keyPrefix, vmInput.Arguments[0]...)

	err := e.toggleSetting(dcdtTokenKey)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}
	return vmOutput, nil
}

func (e *dcdtGlobalSettings) toggleSetting(dcdtTokenKey []byte) error {
	systemSCAccount, err := e.getSystemAccount()
	if err != nil {
		return err
	}

	dcdtMetaData, err := e.getGlobalMetadata(dcdtTokenKey)
	if err != nil {
		return err
	}

	switch e.function {
	case core.BuiltInFunctionDCDTSetLimitedTransfer, core.BuiltInFunctionDCDTUnSetLimitedTransfer:
		dcdtMetaData.LimitedTransfer = e.set
	case core.BuiltInFunctionDCDTPause, core.BuiltInFunctionDCDTUnPause:
		dcdtMetaData.Paused = e.set
	case vmcommon.BuiltInFunctionDCDTUnSetBurnRoleForAll, vmcommon.BuiltInFunctionDCDTSetBurnRoleForAll:
		dcdtMetaData.BurnRoleForAll = e.set
	}

	err = systemSCAccount.AccountDataHandler().SaveKeyValue(dcdtTokenKey, dcdtMetaData.ToBytes())
	if err != nil {
		return err
	}

	return e.accounts.SaveAccount(systemSCAccount)
}

func (e *dcdtGlobalSettings) getSystemAccount() (vmcommon.UserAccountHandler, error) {
	systemSCAccount, err := e.accounts.LoadAccount(vmcommon.SystemAccountAddress)
	if err != nil {
		return nil, err
	}

	userAcc, ok := systemSCAccount.(vmcommon.UserAccountHandler)
	if !ok {
		return nil, ErrWrongTypeAssertion
	}

	return userAcc, nil
}

// IsPaused returns true if the dcdtTokenKey (prefixed) is paused
func (e *dcdtGlobalSettings) IsPaused(dcdtTokenKey []byte) bool {
	dcdtMetadata, err := e.getGlobalMetadata(dcdtTokenKey)
	if err != nil {
		return false
	}

	return dcdtMetadata.Paused
}

// IsLimitedTransfer returns true if the dcdtTokenKey (prefixed) is with limited transfer
func (e *dcdtGlobalSettings) IsLimitedTransfer(dcdtTokenKey []byte) bool {
	dcdtMetadata, err := e.getGlobalMetadata(dcdtTokenKey)
	if err != nil {
		return false
	}

	return dcdtMetadata.LimitedTransfer
}

// IsBurnForAll returns true if the dcdtTokenKey (prefixed) is with burn for all
func (e *dcdtGlobalSettings) IsBurnForAll(dcdtTokenKey []byte) bool {
	dcdtMetadata, err := e.getGlobalMetadata(dcdtTokenKey)
	if err != nil {
		return false
	}

	return dcdtMetadata.BurnRoleForAll
}

// IsSenderOrDestinationWithTransferRole returns true if we have transfer role on the system account
func (e *dcdtGlobalSettings) IsSenderOrDestinationWithTransferRole(sender, destination, tokenID []byte) bool {
	if !e.activeHandler() {
		return false
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return false
	}

	dcdtTokenTransferRoleKey := append(transferAddressesKeyPrefix, tokenID...)
	addresses, _, err := getDCDTRolesForAcnt(e.marshaller, systemAcc, dcdtTokenTransferRoleKey)
	if err != nil {
		return false
	}

	for _, address := range addresses.Roles {
		if bytes.Equal(address, sender) || bytes.Equal(address, destination) {
			return true
		}
	}

	return false
}

func (e *dcdtGlobalSettings) getGlobalMetadata(dcdtTokenKey []byte) (*DCDTGlobalMetadata, error) {
	systemSCAccount, err := e.getSystemAccount()
	if err != nil {
		return nil, err
	}

	val, _, err := systemSCAccount.AccountDataHandler().RetrieveValue(dcdtTokenKey)
	if core.IsGetNodeFromDBError(err) {
		return nil, err
	}
	dcdtMetaData := DCDTGlobalMetadataFromBytes(val)
	return &dcdtMetaData, nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dcdtGlobalSettings) IsInterfaceNil() bool {
	return e == nil
}
