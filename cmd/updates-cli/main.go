// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// "token-cli" implements tokenvm client operation interface.
package main

import (
	"os"

	"hyper-updates/cmd/updates-cli/cmd"

	"github.com/ava-labs/hypersdk/utils"
)

func main() {
	if err := cmd.Execute(); err != nil {
		utils.Outf("{{red}}Updates-cli exited with error:{{/}} %+v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
