package builtInFunctions

import (
	"bytes"
	"math/big"

	"github.com/DharitriOne/drt-chain-core-go/core"
	"github.com/DharitriOne/drt-chain-core-go/core/check"
	"github.com/DharitriOne/drt-chain-core-go/data/dcdt"
	"github.com/DharitriOne/drt-chain-core-go/marshal"
	vmcommon "github.com/DharitriOne/drt-chain-vm-common-go"
)

const transfer = "transfer"

var transferAddressesKeyPrefix = []byte(core.ProtectedKeyPrefix + transfer + core.DCDTKeyIdentifier)

type dcdtTransferAddress struct {
	baseActiveHandler
	set             bool
	marshaller      vmcommon.Marshalizer
	accounts        vmcommon.AccountsAdapter
	maxNumAddresses uint32
}

// NewDCDTTransferRoleAddressFunc returns the dcdt transfer role address handler built-in function component
func NewDCDTTransferRoleAddressFunc(
	accounts vmcommon.AccountsAdapter,
	marshaller marshal.Marshalizer,
	maxNumAddresses uint32,
	set bool,
	enableEpochsHandler vmcommon.EnableEpochsHandler,
) (*dcdtTransferAddress, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}
	if check.IfNil(accounts) {
		return nil, ErrNilAccountsAdapter
	}
	if maxNumAddresses < 1 {
		return nil, ErrInvalidMaxNumAddresses
	}
	if check.IfNil(enableEpochsHandler) {
		return nil, ErrNilEnableEpochsHandler
	}

	e := &dcdtTransferAddress{
		accounts:        accounts,
		marshaller:      marshaller,
		maxNumAddresses: maxNumAddresses,
		set:             set,
	}

	e.baseActiveHandler.activeHandler = func() bool {
		return enableEpochsHandler.IsFlagEnabled(SendAlwaysFlag)
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dcdtTransferAddress) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves DCDT change roles function call
func (e *dcdtTransferAddress) ProcessBuiltinFunction(
	_, _ vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	err := checkBasicDCDTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(vmInput.CallerAddr, core.DCDTSCAddress) {
		return nil, ErrAddressIsNotDCDTSystemSC
	}
	if !vmcommon.IsSystemAccountAddress(vmInput.RecipientAddr) {
		return nil, ErrOnlySystemAccountAccepted
	}

	systemAcc, err := e.getSystemAccount()
	if err != nil {
		return nil, err
	}

	dcdtTokenTransferRoleKey := append(transferAddressesKeyPrefix, vmInput.Arguments[0]...)
	addresses, _, err := getDCDTRolesForAcnt(e.marshaller, systemAcc, dcdtTokenTransferRoleKey)
	if err != nil {
		return nil, err
	}

	if e.set {
		err = e.addNewAddresses(vmInput, addresses)
		if err != nil {
			return nil, err
		}
	} else {
		deleteRoles(addresses, vmInput.Arguments[1:])
	}

	err = saveRolesToAccount(systemAcc, dcdtTokenTransferRoleKey, addresses, e.marshaller)
	if err != nil {
		return nil, err
	}

	err = e.accounts.SaveAccount(systemAcc)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}

	logData := append([][]byte{systemAcc.AddressBytes()}, vmInput.Arguments[1:]...)
	addDCDTEntryInVMOutput(vmOutput, []byte(vmInput.Function), vmInput.Arguments[0], 0, big.NewInt(0), logData...)

	return vmOutput, nil
}

func (e *dcdtTransferAddress) addNewAddresses(vmInput *vmcommon.ContractCallInput, addresses *dcdt.DCDTRoles) error {
	for _, newAddress := range vmInput.Arguments[1:] {
		isNew := true
		for _, address := range addresses.Roles {
			if bytes.Equal(newAddress, address) {
				isNew = false
				break
			}
		}
		if isNew {
			addresses.Roles = append(addresses.Roles, newAddress)
		}
	}

	if uint32(len(addresses.Roles)) > e.maxNumAddresses {
		return ErrTooManyTransferAddresses
	}

	return nil
}

func (e *dcdtTransferAddress) getSystemAccount() (vmcommon.UserAccountHandler, error) {
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

// IsInterfaceNil returns true if underlying object in nil
func (e *dcdtTransferAddress) IsInterfaceNil() bool {
	return e == nil
}
