package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/spf13/cobra"
)

type HubSnapshot struct {
	Accounts map[string]HubSnapshotAccount `json:"accounts"`
}

// HubSnapshotAccount provide fields of snapshot per account
type HubSnapshotAccount struct {
	AtomAddress       string `json:"atom_address"`
	AtomStaker        bool   `json:"atom_staker"`
	StargazeDelegator bool   `json:"stargaze_delegator"`
}

func isIn(s string, ss []string) bool {
	for _, t := range ss {
		if t == s {
			return true
		}
	}
	return false
}

// ExportHubSnapshotCmd generates a snapshot.json from a provided Cosmos Hub genesis export.
func ExportHubSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export-hub-snapshot [airdrop-to-denom] [input-genesis-file] [output-snapshot-json]",
		Short: "Export snapshot from a provided Cosmos Hub genesis export",
		Long: `Export snapshot from a provided Cosmos Hub genesis export
Example:
	starsd export-hub-snapshot genesis.json hub-snapshot.json
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			serverCtx := server.GetServerContextFromCmd(cmd)
			config := serverCtx.Config

			config.SetRoot(clientCtx.HomeDir)

			genesisFile := args[0]
			snapshotOutput := args[1]

			// Read genesis file
			genesisJSON, err := os.Open(genesisFile)
			if err != nil {
				return err
			}
			defer genesisJSON.Close()

			// Produce the map of address to total atom balance, both staked and unstaked
			snapshotAccs := make(map[string]HubSnapshotAccount)

			cdc := clientCtx.Codec

			appState, _, error := genutiltypes.GenesisStateFromGenFile(genesisFile)
			if error != nil {
				return fmt.Errorf("failed to unmarshal genesis state: %w", err)
			}

			exchanges := strings.Split(strings.TrimSpace(os.Getenv("EXCHANGES")), ",")
			fmt.Println("exchanges", len(exchanges))
			if len(exchanges) == 0 || strings.TrimSpace(os.Getenv("EXCHANGES")) == "" {
				panic("provide list of addresses")
			}
			stakingGenState := stakingtypes.GetGenesisStateFromAppState(cdc, appState)

			// Make a map from validator operator address to the validator type
			validators := make(map[string]stakingtypes.Validator)
			for _, validator := range stakingGenState.Validators {
				validators[validator.OperatorAddress] = validator
			}
			amounts := make(map[string]sdk.Dec)
			stakers := 0
			delegators := 0
			for _, delegation := range stakingGenState.Delegations {
				if isIn(delegation.ValidatorAddress, exchanges) {
					continue
				}

				val, ok := validators[delegation.ValidatorAddress]
				if !ok {
					panic(fmt.Sprintf("missing validator %s ", delegation.GetValidatorAddr()))
				}

				address := delegation.DelegatorAddress
				delegationAmount := val.TokensFromShares(delegation.Shares).Quo(sdk.NewDec(1_000_000))
				current, ok := amounts[address]
				if !ok {
					current = sdk.ZeroDec()
				}
				newAmount := current.Add(delegationAmount)
				amounts[address] = newAmount

				acc, ok := snapshotAccs[address]
				if !ok {
					acc = HubSnapshotAccount{
						AtomAddress: address,
					}
				}
				staker := false
				stargazer := false
				// MIN 5ATOM
				if newAmount.GTE(sdk.NewDec(5)) {
					acc.AtomStaker = true
					staker = true
					stakers++
				}

				if delegation.ValidatorAddress == "cosmosvaloper1et77usu8q2hargvyusl4qzryev8x8t9wwqkxfs" && delegationAmount.GTE(sdk.NewDec(5)) {
					acc.StargazeDelegator = true
					stargazer = true
					delegators++
				}
				if stargazer || staker {
					snapshotAccs[address] = acc
				}
			}

			snapshot := HubSnapshot{
				Accounts: snapshotAccs,
			}

			fmt.Printf("accounts: %d\n", len(snapshotAccs))
			fmt.Printf("stakers: %d\n", stakers)
			fmt.Printf("delegators: %d\n", delegators)

			// export snapshot json
			snapshotJSON, err := json.MarshalIndent(snapshot, "", "    ")
			if err != nil {
				return fmt.Errorf("failed to marshal snapshot: %w", err)
			}

			err = ioutil.WriteFile(snapshotOutput, snapshotJSON, 0600)
			return err
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
