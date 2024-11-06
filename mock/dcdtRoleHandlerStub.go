package mock

import vmcommon "github.com/DharitriOne/drt-chain-vm-common-go"

// DCDTRoleHandlerStub -
type DCDTRoleHandlerStub struct {
	CheckAllowedToExecuteCalled func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error
}

// CheckAllowedToExecute -
func (e *DCDTRoleHandlerStub) CheckAllowedToExecute(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
	if e.CheckAllowedToExecuteCalled != nil {
		return e.CheckAllowedToExecuteCalled(account, tokenID, action)
	}

	return nil
}

// IsInterfaceNil -
func (e *DCDTRoleHandlerStub) IsInterfaceNil() bool {
	return e == nil
}
