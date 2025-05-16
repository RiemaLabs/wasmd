package cli

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/ioutils"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/testdata"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestParseVerificationFlags(t *testing.T) {
	mySender := sdk.MustAccAddressFromBech32("bc1p0xlxvlhemja6c4dqv22uapctqupfhlxm9h8z3k2e72q4k9hcz7vqzk5jj0")

	specs := map[string]struct {
		srcPath     string
		args        []string
		expErr      bool
		expSource   string
		expBuilder  string
		expCodeHash string
	}{
		"gov store zipped": {
			srcPath: "../../keeper/testdata/hackatom.wasm.gzip",
			args: []string{
				"--instantiate-everybody=true", "--code-hash=" + testdata.ChecksumHackatom,
				"--code-source-url=https://example.com", "--builder=cosmwasm/workspace-optimizer:0.12.11",
			},
			expBuilder:  "cosmwasm/workspace-optimizer:0.12.11",
			expSource:   "https://example.com",
			expCodeHash: testdata.ChecksumHackatom,
		},
		"gov store raw": {
			srcPath: "../../keeper/testdata/hackatom.wasm",
			args: []string{
				"--instantiate-everybody=true", "--code-hash=" + testdata.ChecksumHackatom,
				"--code-source-url=https://example.com", "--builder=cosmwasm/workspace-optimizer:0.12.11",
			},
			expBuilder:  "cosmwasm/workspace-optimizer:0.12.11",
			expSource:   "https://example.com",
			expCodeHash: testdata.ChecksumHackatom,
		},
		"gov store checksum mismatch": {
			srcPath: "../../keeper/testdata/hackatom.wasm",
			args: []string{
				"--instantiate-everybody=true", "--code-hash=0000de5e9b93b52e514c74ce87ccddb594b9bcd33b7f1af1bb6da63fc883917b",
				"--code-source-url=https://example.com", "--builder=cosmwasm/workspace-optimizer:0.12.11",
			},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			flagSet := ProposalStoreAndInstantiateContractCmd().Flags()
			require.NoError(t, flagSet.Parse(spec.args))

			gotMsg, err := parseStoreCodeArgs(spec.srcPath, mySender.String(), flagSet)
			require.NoError(t, err)
			require.True(t, ioutils.IsGzip(gotMsg.WASMByteCode))

			gotSource, gotBuilder, gotCodeHash, gotErr := parseVerificationFlags(gotMsg.WASMByteCode, flagSet)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expSource, gotSource)
			assert.Equal(t, spec.expBuilder, gotBuilder)
			assert.Equal(t, spec.expCodeHash, hex.EncodeToString(gotCodeHash))
		})
	}
}

func TestParseAccessConfigFlags(t *testing.T) {
	specs := map[string]struct {
		args   []string
		expCfg *types.AccessConfig
		expErr bool
	}{
		"nobody": {
			args:   []string{"--instantiate-nobody=true"},
			expCfg: &types.AccessConfig{Permission: types.AccessTypeNobody},
		},
		"everybody": {
			args:   []string{"--instantiate-everybody=true"},
			expCfg: &types.AccessConfig{Permission: types.AccessTypeEverybody},
		},
		"only address": {
			args:   []string{"--instantiate-only-address=bc1pcqx3f8qg4w3mrvmq3ndcx225tng2tgqgpw0dquc3eyyef5y55nnqecqml7"},
			expErr: true,
		},
		"only address - invalid": {
			args:   []string{"--instantiate-only-address=foo"},
			expErr: true,
		},
		"any of address": {
			args:   []string{"--instantiate-anyof-addresses=bc1pcqx3f8qg4w3mrvmq3ndcx225tng2tgqgpw0dquc3eyyef5y55nnqecqml7,bc1p8eumq4zlwmvr8dtau69fq8hcv3lvmna85j2xycqlm5l6zvwtkh2q24frkg"},
			expCfg: &types.AccessConfig{Permission: types.AccessTypeAnyOfAddresses, Addresses: []string{"bc1pcqx3f8qg4w3mrvmq3ndcx225tng2tgqgpw0dquc3eyyef5y55nnqecqml7", "bc1p8eumq4zlwmvr8dtau69fq8hcv3lvmna85j2xycqlm5l6zvwtkh2q24frkg"}},
		},
		"any of address - invalid": {
			args:   []string{"--instantiate-anyof-addresses=bc1pcqx3f8qg4w3mrvmq3ndcx225tng2tgqgpw0dquc3eyyef5y55nnqecqml7,foo"},
			expErr: true,
		},
		"not set": {
			args: []string{},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			flags := StoreCodeCmd().Flags()
			require.NoError(t, flags.Parse(spec.args))
			gotCfg, gotErr := parseAccessConfigFlags(flags)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expCfg, gotCfg)
		})
	}
}

func TestParseStoreCodeGrants(t *testing.T) {
	specs := map[string]struct {
		src    []string
		exp    []types.CodeGrant
		expErr bool
	}{
		"wildcard : nobody": {
			src: []string{"*:nobody"},
			exp: []types.CodeGrant{{
				CodeHash:              []byte("*"),
				InstantiatePermission: &types.AccessConfig{Permission: types.AccessTypeNobody},
			}},
		},
		"wildcard : wildcard": {
			src: []string{"*:*"},
			exp: []types.CodeGrant{{
				CodeHash: []byte("*"),
			}},
		},
		"wildcard : everybody": {
			src: []string{"*:everybody"},
			exp: []types.CodeGrant{{
				CodeHash:              []byte("*"),
				InstantiatePermission: &types.AccessConfig{Permission: types.AccessTypeEverybody},
			}},
		},
		"wildcard : any of addresses - single": {
			src: []string{"*:bc1pcqx3f8qg4w3mrvmq3ndcx225tng2tgqgpw0dquc3eyyef5y55nnqecqml7"},
			exp: []types.CodeGrant{
				{
					CodeHash: []byte("*"),
					InstantiatePermission: &types.AccessConfig{
						Permission: types.AccessTypeAnyOfAddresses,
						Addresses:  []string{"bc1pcqx3f8qg4w3mrvmq3ndcx225tng2tgqgpw0dquc3eyyef5y55nnqecqml7"},
					},
				},
			},
		},
		"wildcard : any of addresses - multiple": {
			src: []string{"*:bc1pcqx3f8qg4w3mrvmq3ndcx225tng2tgqgpw0dquc3eyyef5y55nnqecqml7,bc1p8eumq4zlwmvr8dtau69fq8hcv3lvmna85j2xycqlm5l6zvwtkh2q24frkg"},
			exp: []types.CodeGrant{
				{
					CodeHash: []byte("*"),
					InstantiatePermission: &types.AccessConfig{
						Permission: types.AccessTypeAnyOfAddresses,
						Addresses:  []string{"bc1pcqx3f8qg4w3mrvmq3ndcx225tng2tgqgpw0dquc3eyyef5y55nnqecqml7", "bc1p8eumq4zlwmvr8dtau69fq8hcv3lvmna85j2xycqlm5l6zvwtkh2q24frkg"},
					},
				},
			},
		},
		"multiple code hashes with different permissions": {
			src: []string{"any_checksum_1:bc1pcqx3f8qg4w3mrvmq3ndcx225tng2tgqgpw0dquc3eyyef5y55nnqecqml7,bc1p8eumq4zlwmvr8dtau69fq8hcv3lvmna85j2xycqlm5l6zvwtkh2q24frkg", "any_checksum_2:nobody"},
			exp: []types.CodeGrant{
				{
					CodeHash: []byte("any_checksum_1"),
					InstantiatePermission: &types.AccessConfig{
						Permission: types.AccessTypeAnyOfAddresses,
						Addresses:  []string{"bc1pcqx3f8qg4w3mrvmq3ndcx225tng2tgqgpw0dquc3eyyef5y55nnqecqml7", "bc1p8eumq4zlwmvr8dtau69fq8hcv3lvmna85j2xycqlm5l6zvwtkh2q24frkg"},
					},
				}, {
					CodeHash: []byte("any_checksum_2"),
					InstantiatePermission: &types.AccessConfig{
						Permission: types.AccessTypeNobody,
					},
				},
			},
		},
		"code hash : wildcard": {
			src: []string{"any_checksum_1:*"},
			exp: []types.CodeGrant{{
				CodeHash: []byte("any_checksum_1"),
			}},
		},
		"code hash : any of addresses - empty list": {
			src:    []string{"any_checksum_1:"},
			expErr: true,
		},
		"code hash : any of addresses - invalid address": {
			src:    []string{"any_checksum_1:foo"},
			expErr: true,
		},
		"code hash : any of addresses - duplicate address": {
			src:    []string{"any_checksum_1:bc1pcqx3f8qg4w3mrvmq3ndcx225tng2tgqgpw0dquc3eyyef5y55nnqecqml7,bc1pcqx3f8qg4w3mrvmq3ndcx225tng2tgqgpw0dquc3eyyef5y55nnqecqml7"},
			expErr: true,
		},
		"empty code hash": {
			src:    []string{":everyone"},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			got, gotErr := parseStoreCodeGrants(spec.src)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.exp, got)
		})
	}
}
