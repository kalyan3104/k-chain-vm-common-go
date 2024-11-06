package builtInFunctions

import (
	"bytes"
	"math"
	"math/big"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/core/check"
	"github.com/kalyan3104/k-chain-core-go/data/dcdt"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
)

var roleKeyPrefix = []byte(core.ProtectedKeyPrefix + core.DCDTRoleIdentifier + core.DCDTKeyIdentifier)

type dcdtRoles struct {
	baseAlwaysActiveHandler
	set        bool
	marshaller vmcommon.Marshalizer
}

// NewDCDTRolesFunc returns the dcdt change roles built-in function component
func NewDCDTRolesFunc(
	marshaller vmcommon.Marshalizer,
	set bool,
) (*dcdtRoles, error) {
	if check.IfNil(marshaller) {
		return nil, ErrNilMarshalizer
	}

	e := &dcdtRoles{
		set:        set,
		marshaller: marshaller,
	}

	return e, nil
}

// SetNewGasConfig is called whenever gas cost is changed
func (e *dcdtRoles) SetNewGasConfig(_ *vmcommon.GasCost) {
}

// ProcessBuiltinFunction resolves DCDT change roles function call
func (e *dcdtRoles) ProcessBuiltinFunction(
	_, acntDst vmcommon.UserAccountHandler,
	vmInput *vmcommon.ContractCallInput,
) (*vmcommon.VMOutput, error) {
	err := checkBasicDCDTArguments(vmInput)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(vmInput.CallerAddr, core.DCDTSCAddress) {
		return nil, ErrAddressIsNotDCDTSystemSC
	}
	if check.IfNil(acntDst) {
		return nil, ErrNilUserAccount
	}

	dcdtTokenRoleKey := append(roleKeyPrefix, vmInput.Arguments[0]...)

	roles, _, err := getDCDTRolesForAcnt(e.marshaller, acntDst, dcdtTokenRoleKey)
	if err != nil {
		return nil, err
	}

	if e.set {
		roles.Roles = append(roles.Roles, vmInput.Arguments[1:]...)
	} else {
		deleteRoles(roles, vmInput.Arguments[1:])
	}

	for _, arg := range vmInput.Arguments[1:] {
		if !bytes.Equal(arg, []byte(core.DCDTRoleNFTCreateMultiShard)) {
			continue
		}

		err = saveLatestNonce(acntDst, vmInput.Arguments[0], computeStartNonce(vmInput.RecipientAddr))
		if err != nil {
			return nil, err
		}

		break
	}

	err = saveRolesToAccount(acntDst, dcdtTokenRoleKey, roles, e.marshaller)
	if err != nil {
		return nil, err
	}

	vmOutput := &vmcommon.VMOutput{ReturnCode: vmcommon.Ok}

	logData := append([][]byte{acntDst.AddressBytes()}, vmInput.Arguments[1:]...)
	addDCDTEntryInVMOutput(vmOutput, []byte(vmInput.Function), vmInput.Arguments[0], 0, big.NewInt(0), logData...)

	return vmOutput, nil
}

// Nonces on multi shard NFT create are from (LastByte * MaxUint64 / 256), this is in order to differentiate them
// even like this, if one contract makes 1000 NFT create on each block, it would need 14 million years to occupy the whole space
// 2 ^ 64 / 256 / 1000 / 14400 / 365 ~= 14 million
func computeStartNonce(destAddress []byte) uint64 {
	lastByteOfAddress := uint64(destAddress[len(destAddress)-1])
	startNonce := (math.MaxUint64 / 256) * lastByteOfAddress
	return startNonce
}

func deleteRoles(roles *dcdt.DCDTRoles, deleteRoles [][]byte) {
	for _, deleteRole := range deleteRoles {
		index, exist := doesRoleExist(roles, deleteRole)
		if !exist {
			continue
		}

		copy(roles.Roles[index:], roles.Roles[index+1:])
		roles.Roles[len(roles.Roles)-1] = nil
		roles.Roles = roles.Roles[:len(roles.Roles)-1]
	}
}

func doesRoleExist(roles *dcdt.DCDTRoles, role []byte) (int, bool) {
	for i, currentRole := range roles.Roles {
		if bytes.Equal(currentRole, role) {
			return i, true
		}
	}
	return -1, false
}

func getDCDTRolesForAcnt(
	marshaller vmcommon.Marshalizer,
	acnt vmcommon.UserAccountHandler,
	key []byte,
) (*dcdt.DCDTRoles, bool, error) {
	roles := &dcdt.DCDTRoles{
		Roles: make([][]byte, 0),
	}

	marshaledData, _, err := acnt.AccountDataHandler().RetrieveValue(key)
	if core.IsGetNodeFromDBError(err) {
		return nil, false, err
	}
	if err != nil || len(marshaledData) == 0 {
		return roles, true, nil
	}

	err = marshaller.Unmarshal(roles, marshaledData)
	if err != nil {
		return nil, false, err
	}

	return roles, false, nil
}

// CheckAllowedToExecute returns error if the account is not allowed to execute the given action
func (e *dcdtRoles) CheckAllowedToExecute(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
	if check.IfNil(account) {
		return ErrNilUserAccount
	}

	dcdtTokenRoleKey := append(roleKeyPrefix, tokenID...)
	roles, isNew, err := getDCDTRolesForAcnt(e.marshaller, account, dcdtTokenRoleKey)
	if err != nil {
		return err
	}
	if isNew {
		return ErrActionNotAllowed
	}
	_, exist := doesRoleExist(roles, action)
	if !exist {
		return ErrActionNotAllowed
	}

	return nil
}

// IsInterfaceNil returns true if underlying object in nil
func (e *dcdtRoles) IsInterfaceNil() bool {
	return e == nil
}
