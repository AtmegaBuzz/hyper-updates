// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

import (
	"context"

	"hyper-updates/storage"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/state"
	"github.com/ava-labs/hypersdk/utils"
)

var _ chain.Action = (*CreateAsset)(nil)

type CreateProject struct {
	ProjectName        []byte `json:"name"`
	ProjectDescription []byte `json:"description"`
	Owner              []byte `json:"owner"`
	Logo               []byte `json:"url"`
}

func (*CreateProject) GetTypeID() uint8 {
	return createProjectID
}

func (*CreateProject) StateKeys(_ chain.Auth, txID ids.ID) []string {
	return []string{
		string(storage.AssetKey(txID)),
	}
}

func (*CreateProject) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.AssetChunks}
}

func (*CreateProject) OutputsWarpMessage() bool {
	return false
}

func (c *CreateProject) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	auth chain.Auth,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {
	if len(c.ProjectName) == 0 {
		return false, CreateProjectComputeUnits, OutputSymbolEmpty, nil, nil
	}
	if len(c.ProjectDescription) == 0 {
		return false, CreateAssetComputeUnits, OutputSymbolTooLarge, nil, nil
	}
	if len(c.Owner) == 0 {
		return false, CreateProjectComputeUnits, OutputDecimalsTooLarge, nil, nil
	}

	// It should only be possible to overwrite an existing asset if there is
	// a hash collision.
	if err := storage.SetAsset(ctx, mu, txID, c.Symbol, c.Decimals, c.Metadata, 0, auth.Actor(), false); err != nil {
		return false, CreateProjectComputeUnits, utils.ErrBytes(err), nil, nil
	}
	return true, CreateProjectComputeUnits, nil, nil, nil
}

func (*CreateProject) MaxComputeUnits(chain.Rules) uint64 {
	return CreateProjectComputeUnits
}

func (c *CreateProject) Size() int {
	// TODO: add small bytes (smaller int prefix)
	return (codec.BytesLen(c.ProjectName) +
		consts.Uint8Len +
		codec.BytesLen(c.ProjectDescription) +
		consts.Uint8Len +
		codec.BytesLen(c.Owner) +
		consts.Uint8Len +
		codec.BytesLen(c.Logo))

}

func (c *CreateProject) Marshal(p *codec.Packer) {
	p.PackBytes(c.ProjectName)
	p.PackBytes(c.ProjectDescription)
	p.PackBytes(c.Owner)
	p.PackBytes(c.Logo)
}

func UnmarshalCreateProject(p *codec.Packer, _ *warp.Message) (chain.Action, error) {

	var create CreateProject

	p.UnpackBytes(ProjectNameUnits, true, &create.ProjectName)
	p.UnpackBytes(ProjectDescriptionUnits, true, &create.ProjectDescription)
	p.UnpackBytes(ProjectOwnerUnits, true, &create.Owner)
	p.UnpackBytes(ProjectLogoUnits, true, &create.Logo)

	return &create, p.Err()

}

func (*CreateProject) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}
