package builtInFunctions

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/kalyan3104/k-chain-core-go/core"
	"github.com/kalyan3104/k-chain-core-go/data/dcdt"
	vmcommon "github.com/kalyan3104/k-chain-vm-common-go"
	"github.com/kalyan3104/k-chain-vm-common-go/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDCDTLocalBurnFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		argsFunc func() (c uint64, m vmcommon.Marshalizer, p vmcommon.ExtendedDCDTGlobalSettingsHandler, r vmcommon.DCDTRoleHandler, e vmcommon.EnableEpochsHandler)
		exError  error
	}{
		{
			name: "NilMarshalizer",
			argsFunc: func() (c uint64, m vmcommon.Marshalizer, p vmcommon.ExtendedDCDTGlobalSettingsHandler, r vmcommon.DCDTRoleHandler, e vmcommon.EnableEpochsHandler) {
				return 0, nil, &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{}
			},
			exError: ErrNilMarshalizer,
		},
		{
			name: "NilGlobalSettingsHandler",
			argsFunc: func() (c uint64, m vmcommon.Marshalizer, p vmcommon.ExtendedDCDTGlobalSettingsHandler, r vmcommon.DCDTRoleHandler, e vmcommon.EnableEpochsHandler) {
				return 0, &mock.MarshalizerMock{}, nil, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{}
			},
			exError: ErrNilGlobalSettingsHandler,
		},
		{
			name: "NilRolesHandler",
			argsFunc: func() (c uint64, m vmcommon.Marshalizer, p vmcommon.ExtendedDCDTGlobalSettingsHandler, r vmcommon.DCDTRoleHandler, e vmcommon.EnableEpochsHandler) {
				return 0, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, nil, &mock.EnableEpochsHandlerStub{}
			},
			exError: ErrNilRolesHandler,
		},
		{
			name: "NilEnableEpochsHandler",
			argsFunc: func() (c uint64, m vmcommon.Marshalizer, p vmcommon.ExtendedDCDTGlobalSettingsHandler, r vmcommon.DCDTRoleHandler, e vmcommon.EnableEpochsHandler) {
				return 0, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, nil
			},
			exError: ErrNilEnableEpochsHandler,
		},
		{
			name: "Ok",
			argsFunc: func() (c uint64, m vmcommon.Marshalizer, p vmcommon.ExtendedDCDTGlobalSettingsHandler, r vmcommon.DCDTRoleHandler, e vmcommon.EnableEpochsHandler) {
				return 0, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{}
			},
			exError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDCDTLocalBurnFunc(tt.argsFunc())
			require.Equal(t, err, tt.exError)
		})
	}
}

func TestDcdtLocalBurn_ProcessBuiltinFunction_CalledWithValueShouldErr(t *testing.T) {
	t.Parallel()

	dcdtLocalBurnF, _ := NewDCDTLocalBurnFunc(0, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})

	_, err := dcdtLocalBurnF.ProcessBuiltinFunction(&mock.AccountWrapMock{}, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(1),
		},
	})
	require.Equal(t, ErrBuiltInFunctionCalledWithValue, err)
}

func TestDcdtLocalBurn_ProcessBuiltinFunction_CheckAllowToExecuteShouldErr(t *testing.T) {
	t.Parallel()

	localErr := errors.New("local err")
	dcdtLocalBurnF, _ := NewDCDTLocalBurnFunc(0, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return localErr
		},
	}, &mock.EnableEpochsHandlerStub{})

	_, err := dcdtLocalBurnF.ProcessBuiltinFunction(&mock.AccountWrapMock{}, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("arg1"), []byte("arg2")},
		},
	})
	require.Equal(t, localErr, err)
}

func TestDcdtLocalBurn_ProcessBuiltinFunction_CannotAddToDcdtBalanceShouldErr(t *testing.T) {
	t.Parallel()

	dcdtLocalBurnF, _ := NewDCDTLocalBurnFunc(0, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return nil
		},
	}, &mock.EnableEpochsHandlerStub{})

	localErr := errors.New("local err")
	_, err := dcdtLocalBurnF.ProcessBuiltinFunction(&mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					return nil, 0, localErr
				},
			}
		},
	}, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue: big.NewInt(0),
			Arguments: [][]byte{[]byte("arg1"), []byte("arg2")},
		},
	})
	require.Equal(t, ErrInsufficientFunds, err)
}

func TestDcdtLocalBurn_ProcessBuiltinFunction_ValueTooLong(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	dcdtRoleHandler := &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.DCDTRoleLocalBurn, string(action))
			return nil
		},
	}
	dcdtLocalBurnF, _ := NewDCDTLocalBurnFunc(50, marshaller, &mock.GlobalSettingsHandlerStub{}, dcdtRoleHandler, &mock.EnableEpochsHandlerStub{})

	sndAccount := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					dcdtData := &dcdt.DCDigitalToken{Value: big.NewInt(100)}
					serializedDcdtData, err := marshaller.Marshal(dcdtData)
					return serializedDcdtData, 0, err
				},
			}
		},
	}

	bigValueStr := "1" + strings.Repeat("0", 1000)
	bigValue, _ := big.NewInt(0).SetString(bigValueStr, 10)
	vmOutput, err := dcdtLocalBurnF.ProcessBuiltinFunction(sndAccount, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), bigValue.Bytes()},
			GasProvided: 500,
		},
	})
	require.Equal(t, "insufficient funds", err.Error()) // before the activation of the flag
	require.Empty(t, vmOutput)

	// try again with the flag enabled
	dcdtLocalBurnF.enableEpochsHandler = &mock.EnableEpochsHandlerStub{
		IsFlagEnabledCalled: func(flag core.EnableEpochFlag) bool {
			return flag == ConsistentTokensValuesLengthCheckFlag
		},
	}
	vmOutput, err = dcdtLocalBurnF.ProcessBuiltinFunction(sndAccount, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), bigValue.Bytes()},
			GasProvided: 500,
		},
	})
	require.Equal(t, "invalid arguments to process built-in function: max length for dcdt local burn value is 100", err.Error())
	require.Empty(t, vmOutput)
}

func TestDcdtLocalBurn_ProcessBuiltinFunction_ShouldWork(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	dcdtRoleHandler := &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			assert.Equal(t, core.DCDTRoleLocalBurn, string(action))
			return nil
		},
	}
	dcdtLocalBurnF, _ := NewDCDTLocalBurnFunc(50, marshaller, &mock.GlobalSettingsHandlerStub{}, dcdtRoleHandler, &mock.EnableEpochsHandlerStub{})

	sndAccout := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					dcdtData := &dcdt.DCDigitalToken{Value: big.NewInt(100)}
					serializedDcdtData, err := marshaller.Marshal(dcdtData)
					return serializedDcdtData, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					dcdtData := &dcdt.DCDigitalToken{}
					_ = marshaller.Unmarshal(dcdtData, value)
					require.Equal(t, big.NewInt(99), dcdtData.Value)
					return nil
				},
			}
		},
	}
	vmOutput, err := dcdtLocalBurnF.ProcessBuiltinFunction(sndAccout, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), big.NewInt(1).Bytes()},
			GasProvided: 500,
		},
	})
	require.Equal(t, nil, err)

	expectedVMOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: 450,
		Logs: []*vmcommon.LogEntry{
			{
				Identifier: []byte("DCDTLocalBurn"),
				Address:    nil,
				Topics:     [][]byte{[]byte("arg1"), big.NewInt(0).Bytes(), big.NewInt(1).Bytes()},
				Data:       nil,
			},
		},
	}
	require.Equal(t, expectedVMOutput, vmOutput)
}

func TestDcdtLocalBurn_ProcessBuiltinFunction_WithGlobalBurn(t *testing.T) {
	t.Parallel()

	marshaller := &mock.MarshalizerMock{}
	dcdtLocalBurnF, _ := NewDCDTLocalBurnFunc(50, marshaller, &mock.GlobalSettingsHandlerStub{
		IsBurnForAllCalled: func(token []byte) bool {
			return true
		},
	}, &mock.DCDTRoleHandlerStub{
		CheckAllowedToExecuteCalled: func(account vmcommon.UserAccountHandler, tokenID []byte, action []byte) error {
			return errors.New("no role")
		},
	}, &mock.EnableEpochsHandlerStub{})

	sndAccout := &mock.UserAccountStub{
		AccountDataHandlerCalled: func() vmcommon.AccountDataHandler {
			return &mock.DataTrieTrackerStub{
				RetrieveValueCalled: func(_ []byte) ([]byte, uint32, error) {
					dcdtData := &dcdt.DCDigitalToken{Value: big.NewInt(100)}
					serializedDcdtData, err := marshaller.Marshal(dcdtData)
					return serializedDcdtData, 0, err
				},
				SaveKeyValueCalled: func(key []byte, value []byte) error {
					dcdtData := &dcdt.DCDigitalToken{}
					_ = marshaller.Unmarshal(dcdtData, value)
					require.Equal(t, big.NewInt(99), dcdtData.Value)
					return nil
				},
			}
		},
	}
	vmOutput, err := dcdtLocalBurnF.ProcessBuiltinFunction(sndAccout, &mock.AccountWrapMock{}, &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), big.NewInt(1).Bytes()},
			GasProvided: 500,
		},
	})
	require.Equal(t, nil, err)

	expectedVMOutput := &vmcommon.VMOutput{
		ReturnCode:   vmcommon.Ok,
		GasRemaining: 450,
		Logs: []*vmcommon.LogEntry{
			{
				Identifier: []byte("DCDTLocalBurn"),
				Address:    nil,
				Topics:     [][]byte{[]byte("arg1"), big.NewInt(0).Bytes(), big.NewInt(1).Bytes()},
				Data:       nil,
			},
		},
	}
	require.Equal(t, expectedVMOutput, vmOutput)
}

func TestDcdtLocalBurn_SetNewGasConfig(t *testing.T) {
	t.Parallel()

	dcdtLocalBurnF, _ := NewDCDTLocalBurnFunc(0, &mock.MarshalizerMock{}, &mock.GlobalSettingsHandlerStub{}, &mock.DCDTRoleHandlerStub{}, &mock.EnableEpochsHandlerStub{})

	dcdtLocalBurnF.SetNewGasConfig(&vmcommon.GasCost{BuiltInCost: vmcommon.BuiltInCost{
		DCDTLocalBurn: 500},
	})

	require.Equal(t, uint64(500), dcdtLocalBurnF.funcGasCost)
}

func TestCheckInputArgumentsForLocalAction_InvalidRecipientAddr(t *testing.T) {
	t.Parallel()

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), big.NewInt(1).Bytes()},
			CallerAddr: []byte("caller"),
		},
		RecipientAddr: []byte("rec"),
	}

	err := checkInputArgumentsForLocalAction(&mock.UserAccountStub{}, vmInput, 0)
	require.Equal(t, ErrInvalidRcvAddr, err)
}

func TestCheckInputArgumentsForLocalAction_NilUserAccount(t *testing.T) {
	t.Parallel()

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:  big.NewInt(0),
			Arguments:  [][]byte{[]byte("arg1"), big.NewInt(1).Bytes()},
			CallerAddr: []byte("caller"),
		},
		RecipientAddr: []byte("caller"),
	}

	err := checkInputArgumentsForLocalAction(nil, vmInput, 0)
	require.Equal(t, ErrNilUserAccount, err)
}

func TestCheckInputArgumentsForLocalAction_NotEnoughGas(t *testing.T) {
	t.Parallel()

	vmInput := &vmcommon.ContractCallInput{
		VMInput: vmcommon.VMInput{
			CallValue:   big.NewInt(0),
			Arguments:   [][]byte{[]byte("arg1"), big.NewInt(10).Bytes()},
			CallerAddr:  []byte("caller"),
			GasProvided: 1,
		},
		RecipientAddr: []byte("caller"),
	}

	err := checkInputArgumentsForLocalAction(&mock.UserAccountStub{}, vmInput, 500)
	require.Equal(t, ErrNotEnoughGas, err)
}
