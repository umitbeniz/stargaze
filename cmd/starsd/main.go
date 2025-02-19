package main

import (
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/public-awesome/stargaze/app"
	"github.com/public-awesome/stargaze/cmd/starsd/cmd"
	"github.com/tendermint/spm/cosmoscmd"
	tmcmds "github.com/tendermint/tendermint/cmd/tendermint/commands"
)

func main() {
	rootCmd, _ := cosmoscmd.NewRootCmd(
		"stars",
		app.AccountAddressPrefix,
		app.DefaultNodeHome,
		app.Name,
		app.ModuleBasics,
		app.NewStargazeApp,
		cosmoscmd.AddSubCmd(cmd.TestnetCmd(app.ModuleBasics)),
		cosmoscmd.AddCustomInitCmd(cmd.InitCmd(app.ModuleBasics, app.DefaultNodeHome)),
		cosmoscmd.AddSubCmd(cmd.PrepareGenesisCmd(app.DefaultNodeHome, app.ModuleBasics)),
		cosmoscmd.AddSubCmd(tmcmds.RollbackStateCmd),
		// this line is used by starport scaffolding # root/arguments
	)
	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
