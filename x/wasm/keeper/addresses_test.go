package keeper

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func prepareCleanup(t *testing.T) {
	// preserve current Bech32 settings and restore them after test completion
	x, y := sdk.GetConfig().GetBech32AccountAddrPrefix(), sdk.GetConfig().GetBech32AccountPubPrefix()
	c := sdk.IsAddrCacheEnabled()
	t.Cleanup(func() {
		sdk.GetConfig().SetBech32PrefixForAccount(x, y)
		sdk.SetAddrCacheEnabled(c)
	})
	// set custom Bech32 settings
	sdk.GetConfig().SetBech32PrefixForAccount("purple", "purple")
	// disable address cache
	// AccAddress -> String conversion is then slower, but does not lead to errors like this:
	//   runtime error: invalid memory address or nil pointer dereference
	sdk.SetAddrCacheEnabled(false)
}

func TestBuildContractAddressClassic(t *testing.T) {
	// set cleanup function
	prepareCleanup(t)
	// prepare test data
	specs := []struct {
		codeId     uint64
		instanceId uint64
		expAddress string
	}{
		{
			codeId:     0,
			instanceId: 0,
			expAddress: "bc1pw0w8sasnut0jx0vvsnvlc8nayq0q2ej8xgrpwgel05tn6wy4r57qf8d792",
		},
		{
			codeId:     0,
			instanceId: 1,
			expAddress: "bc1p56r47kpk4va938pmtpuee4fh77847gqcw2dmpl2nnpwztwfgz04s5739dx",
		},
		{
			codeId:     1,
			instanceId: 0,
			expAddress: "bc1pmzdhwvvh22wrt07w59wxyd58822qavwkx5lcej7aqfkpqqlhaqfs5lmwgz",
		},
		{
			codeId:     1,
			instanceId: 1,
			expAddress: "bc1p4hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9sm7cwu0",
		},
	}
	// run tests
	for i, spec := range specs {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			// when
			gotAddr := BuildContractAddressClassic(spec.codeId, spec.instanceId)
			// then
			require.Equal(t, spec.expAddress, gotAddr.String())
			require.NoError(t, sdk.VerifyAddressFormat(gotAddr))
		})
	}
}

func TestBuildContractAddressPredictableShort(t *testing.T) {
	types.ContractAddrLen = 20
	// reset to default value after test completion
	defer func() { types.ContractAddrLen = 32 }()

	checksum, err := hex.DecodeString("13a1fc994cc6d1c81b746ee0c0ff6f90043875e0bf1d9be6b7d779fc978dc2a5")
	require.NoError(t, err)
	creator, err := sdk.AccAddressFromHexUnsafe("9999999999aaaaaaaaaabbbbbbbbbbcccccccccc")
	require.NoError(t, err)
	salt, err := hex.DecodeString("61")
	require.NoError(t, err)
	expAddr, err := sdk.AccAddressFromHexUnsafe("5e865d3e45ad3e961f77fd77d46543417ced44d9")
	require.NoError(t, err)

	addr := BuildContractAddressPredictable(checksum, creator, salt, []byte{})
	assert.Equal(t, expAddr, addr)
}
