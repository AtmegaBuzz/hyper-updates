// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

import (
	"context"

	"hyper-updates/consts"
	"hyper-updates/storage"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/platformvm/warp"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/state"
	"github.com/ava-labs/hypersdk/utils"
)

var _ chain.Action = (*CreateProject)(nil)

type CreateProject struct {
	ProjectName        []byte `json:"name"`
	ProjectDescription []byte `json:"description"`
	Logo               []byte `json:"url"`
}

func (*CreateProject) GetTypeID() uint8 {
	return createProjectID
}

func (*CreateProject) StateKeys(_ chain.Auth, txID ids.ID) []string {
	return []string{
		string(storage.ProjectKey(txID)),
	}
}

func (*CreateProject) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.ProjectDescriptionChunks}
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
		return false, CreateProjectComputeUnits, OutputProjectNameNotGiven, nil, nil
	}
	if len(c.ProjectDescription) == 0 {
		return false, CreateAssetComputeUnits, OutputProjectDescriptionNotGiven, nil, nil
	}

	owner, err := codec.AddressBech32(consts.HRP, auth.Actor())

	if err != nil {
		return false, CreateAssetComputeUnits, OutputProjectInvalidOwner, nil, nil
	}

	// It should only be possible to overwrite an existing asset if there is
	// a hash collision.
	if err := storage.SetProject(ctx, mu, txID, c.ProjectName, c.ProjectDescription, []byte(owner), c.Logo); err != nil {
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
		codec.BytesLen(c.ProjectDescription) +
		codec.BytesLen(c.Logo))

}

func (c *CreateProject) Marshal(p *codec.Packer) {
	p.PackBytes(c.ProjectName)
	p.PackBytes(c.ProjectDescription)
	p.PackBytes(c.Logo)
}

func UnmarshalCreateProject(p *codec.Packer, _ *warp.Message) (chain.Action, error) {

	var create CreateProject

	p.UnpackBytes(ProjectNameUnits, true, &create.ProjectName)
	p.UnpackBytes(ProjectDescriptionUnits, true, &create.ProjectDescription)
	p.UnpackBytes(ProjectLogoUnits, true, &create.Logo)

	return &create, p.Err()

}

func (*CreateProject) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}
