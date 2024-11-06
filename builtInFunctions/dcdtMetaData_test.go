package builtInFunctions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDCDTGlobalMetaData_ToBytesWhenPaused(t *testing.T) {
	t.Parallel()

	dcdtMetaData := &DCDTGlobalMetadata{
		Paused: true,
	}

	expected := make([]byte, lengthOfDCDTMetadata)
	expected[0] = 1
	actual := dcdtMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestDCDTGlobalMetaData_ToBytesWhenTransfer(t *testing.T) {
	t.Parallel()

	dcdtMetaData := &DCDTGlobalMetadata{
		LimitedTransfer: true,
	}

	expected := make([]byte, lengthOfDCDTMetadata)
	expected[0] = 2
	actual := dcdtMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestDCDTGlobalMetaData_ToBytesWhenTransferAndPause(t *testing.T) {
	t.Parallel()

	dcdtMetaData := &DCDTGlobalMetadata{
		Paused:          true,
		LimitedTransfer: true,
	}

	expected := make([]byte, lengthOfDCDTMetadata)
	expected[0] = 3
	actual := dcdtMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestDCDTGlobalMetaData_ToBytesWhenNotPaused(t *testing.T) {
	t.Parallel()

	dcdtMetaData := &DCDTGlobalMetadata{
		Paused: false,
	}

	expected := make([]byte, lengthOfDCDTMetadata)
	expected[0] = 0
	actual := dcdtMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestDCDTGlobalMetadataFromBytes_InvalidLength(t *testing.T) {
	t.Parallel()

	emptyDcdtGlobalMetaData := DCDTGlobalMetadata{}

	invalidLengthByteSlice := make([]byte, lengthOfDCDTMetadata+1)

	result := DCDTGlobalMetadataFromBytes(invalidLengthByteSlice)
	require.Equal(t, emptyDcdtGlobalMetaData, result)
}

func TestDCDTGlobalMetadataFromBytes_ShouldSetPausedToTrue(t *testing.T) {
	t.Parallel()

	input := make([]byte, lengthOfDCDTMetadata)
	input[0] = 1

	result := DCDTGlobalMetadataFromBytes(input)
	require.True(t, result.Paused)
}

func TestDCDTGlobalMetadataFromBytes_ShouldSetPausedToFalse(t *testing.T) {
	t.Parallel()

	input := make([]byte, lengthOfDCDTMetadata)
	input[0] = 0

	result := DCDTGlobalMetadataFromBytes(input)
	require.False(t, result.Paused)
}

func TestDCDTUserMetaData_ToBytesWhenFrozen(t *testing.T) {
	t.Parallel()

	dcdtMetaData := &DCDTUserMetadata{
		Frozen: true,
	}

	expected := make([]byte, lengthOfDCDTMetadata)
	expected[0] = 1
	actual := dcdtMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestDCDTUserMetaData_ToBytesWhenNotFrozen(t *testing.T) {
	t.Parallel()

	dcdtMetaData := &DCDTUserMetadata{
		Frozen: false,
	}

	expected := make([]byte, lengthOfDCDTMetadata)
	expected[0] = 0
	actual := dcdtMetaData.ToBytes()
	require.Equal(t, expected, actual)
}

func TestDCDTUserMetadataFromBytes_InvalidLength(t *testing.T) {
	t.Parallel()

	emptyDcdtUserMetaData := DCDTUserMetadata{}

	invalidLengthByteSlice := make([]byte, lengthOfDCDTMetadata+1)

	result := DCDTUserMetadataFromBytes(invalidLengthByteSlice)
	require.Equal(t, emptyDcdtUserMetaData, result)
}

func TestDCDTUserMetadataFromBytes_ShouldSetFrozenToTrue(t *testing.T) {
	t.Parallel()

	input := make([]byte, lengthOfDCDTMetadata)
	input[0] = 1

	result := DCDTUserMetadataFromBytes(input)
	require.True(t, result.Frozen)
}

func TestDCDTUserMetadataFromBytes_ShouldSetFrozenToFalse(t *testing.T) {
	t.Parallel()

	input := make([]byte, lengthOfDCDTMetadata)
	input[0] = 0

	result := DCDTUserMetadataFromBytes(input)
	require.False(t, result.Frozen)
}

func TestDCDTGlobalMetadata_FromBytes(t *testing.T) {
	require.True(t, DCDTGlobalMetadataFromBytes([]byte{1, 0}).Paused)
	require.False(t, DCDTGlobalMetadataFromBytes([]byte{1, 0}).LimitedTransfer)
	require.True(t, DCDTGlobalMetadataFromBytes([]byte{2, 0}).LimitedTransfer)
	require.False(t, DCDTGlobalMetadataFromBytes([]byte{2, 0}).Paused)
	require.False(t, DCDTGlobalMetadataFromBytes([]byte{0, 0}).LimitedTransfer)
	require.False(t, DCDTGlobalMetadataFromBytes([]byte{0, 0}).Paused)
	require.True(t, DCDTGlobalMetadataFromBytes([]byte{3, 0}).Paused)
	require.True(t, DCDTGlobalMetadataFromBytes([]byte{3, 0}).LimitedTransfer)
}
