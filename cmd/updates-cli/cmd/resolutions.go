// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cmd

import (
	"context"
	"fmt"
	"reflect"

	"hyper-updates/actions"
	tconsts "hyper-updates/consts"
	trpc "hyper-updates/rpc"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/cli"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/rpc"
	"github.com/ava-labs/hypersdk/utils"
)

// sendAndWait may not be used concurrently
func sendAndWait(
	ctx context.Context, warpMsg *warp.Message, action chain.Action, cli *rpc.JSONRPCClient,
	scli *rpc.WebSocketClient, tcli *trpc.JSONRPCClient, factory chain.AuthFactory, printStatus bool,
) (bool, ids.ID, error) {
	parser, err := tcli.Parser(ctx)
	if err != nil {
		return false, ids.Empty, err
	}
	_, tx, _, err := cli.GenerateTransaction(ctx, parser, warpMsg, action, factory)
	if err != nil {
		return false, ids.Empty, err
	}

	if err := scli.RegisterTx(tx); err != nil {
		return false, ids.Empty, err
	}
	var res *chain.Result
	for {
		txID, dErr, result, err := scli.ListenTx(ctx)
		if dErr != nil {
			return false, ids.Empty, dErr
		}
		if err != nil {
			return false, ids.Empty, err
		}
		if txID == tx.ID() {
			res = result
			break
		}
		utils.Outf("{{yellow}}skipping unexpected transaction:{{/}} %s\n", tx.ID())
	}
	if printStatus {
		handler.Root().PrintStatus(tx.ID(), res.Success)
	}
	return res.Success, tx.ID(), nil
}

func handleTx(c *trpc.JSONRPCClient, tx *chain.Transaction, result *chain.Result) {
	summaryStr := string(result.Output)
	actor := tx.Auth.Actor()
	status := "⚠️"
	if result.Success {
		status = "✅"
		switch action := tx.Action.(type) {

		case *actions.CreateProject:
			summaryStr += fmt.Sprintf("Project added successfullt | Project Id: %s Project Name: %s", tx.ID(), action.ProjectName)
			fmt.Sprintf("Project Id: %s Project Name: %s", tx.ID(), action.ProjectName)
			utils.Outf(summaryStr)

		case *actions.CreateUpdate:
			summaryStr += fmt.Sprintf("Update added with Update Id: %s for Project: %s", tx.ID(), action.ProjectTxID)
			fmt.Sprintf("Update added with Update Id: %s for Project: %s", tx.ID(), action.ProjectTxID)
			utils.Outf(summaryStr)

		}
	}
	utils.Outf(
		"%s {{yellow}}%s{{/}} {{yellow}}actor:{{/}} %s {{yellow}}summary (%s):{{/}} [%s] {{yellow}}fee (max %.2f%%):{{/}} %s %s {{yellow}}consumed:{{/}} [%s]\n",
		status,
		tx.ID(),
		codec.MustAddressBech32(tconsts.HRP, actor),
		reflect.TypeOf(tx.Action),
		summaryStr,
		float64(result.Fee)/float64(tx.Base.MaxFee)*100,
		utils.FormatBalance(result.Fee, tconsts.Decimals),
		tconsts.Symbol,
		cli.ParseDimensions(result.Consumed),
	)
}
