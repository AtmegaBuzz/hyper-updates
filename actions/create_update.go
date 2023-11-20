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
	"github.com/ava-labs/hypersdk/state"
	"github.com/ava-labs/hypersdk/utils"
)

var _ chain.Action = (*CreateUpdate)(nil)

type CreateUpdate struct {
	ProjectTxID          []byte `json:"project_id"` // reference to Project
	UpdateExecutableHash []byte `json:"executable_hash"`
	UpdateIPFSUrl        []byte `json:"executable_ipfs_url"`
	ForDeviceName        []byte `json:"for_device_name"`
	UpdateVersion        uint8  `json:"version"`
	SuccessCount         uint8  `json:"success_count"`
}

func (*CreateUpdate) GetTypeID() uint8 {
	return createUpdateID
}

func (*CreateUpdate) StateKeys(_ chain.Auth, txID ids.ID) []string {
	return []string{
		string(storage.UpdateKey(txID)),
	}
}

func (*CreateUpdate) StateKeysMaxChunks() []uint16 {
	return []uint16{storage.UpdateExecutableHashChunks}
}

func (*CreateUpdate) OutputsWarpMessage() bool {
	return false
}

func (c *CreateUpdate) Execute(
	ctx context.Context,
	_ chain.Rules,
	mu state.Mutable,
	_ int64,
	auth chain.Auth,
	txID ids.ID,
	_ bool,
) (bool, uint64, []byte, *warp.UnsignedMessage, error) {

	if len(c.ProjectTxID) == 0 {
		return false, CreateUpdateComputeUnits, OutputProjectTxIdNotProvided, nil, nil
	}
	if len(c.UpdateExecutableHash) == 0 {
		return false, CreateUpdateComputeUnits, OutputUpdateExecutableHashNotProvided, nil, nil
	}

	if len(c.ForDeviceName) == 0 {
		return false, CreateAssetComputeUnits, OutputForDeviceNameNotProvided, nil, nil
	}

	if len(c.UpdateIPFSUrl) == 0 {
		return false, CreateAssetComputeUnits, OutputUpdateExecutableIPFSNotProvided, nil, nil
	}

	if c.UpdateVersion == 0 {
		return false, CreateAssetComputeUnits, OutputUpdateVersionNotProvided, nil, nil
	}

	// It should only be possible to overwrite an existing asset if there is
	// a hash collision.
	if err := storage.SetUpdate(ctx, mu, txID, c.ProjectTxID, c.UpdateExecutableHash, c.UpdateIPFSUrl, c.ForDeviceName, byte(c.UpdateVersion), byte(c.SuccessCount)); err != nil {
		return false, CreateUpdateComputeUnits, utils.ErrBytes(err), nil, nil
	}
	return true, CreateUpdateComputeUnits, nil, nil, nil
}

func (*CreateUpdate) MaxComputeUnits(chain.Rules) uint64 {
	return CreateUpdateComputeUnits
}

func (c *CreateUpdate) Size() int {

	return (codec.BytesLen(c.ProjectTxID) +
		codec.BytesLen(c.UpdateExecutableHash) +
		codec.BytesLen(c.UpdateIPFSUrl) +
		codec.BytesLen(c.ForDeviceName) +
		UpdateVersionUnits +
		SuccessCountUnits)

}

func (c *CreateUpdate) Marshal(p *codec.Packer) {
	p.PackBytes(c.ProjectTxID)
	p.PackBytes(c.UpdateExecutableHash)
	p.PackBytes(c.UpdateIPFSUrl)
	p.PackBytes(c.ForDeviceName)
	p.PackByte(c.UpdateVersion)
	p.PackByte(c.SuccessCount)

}

func UnmarshalCreateUpdate(p *codec.Packer, _ *warp.Message) (chain.Action, error) {

	var create CreateUpdate

	p.UnpackBytes(ProjectTxIDUnits, true, &create.ProjectTxID)
	p.UnpackBytes(UpdateExecutableHashUnits, true, &create.UpdateExecutableHash)
	p.UnpackBytes(UpdateExecutableIPFSUrl, true, &create.UpdateIPFSUrl)
	p.UnpackBytes(ForDeviceNameUnits, true, &create.ForDeviceName)

	create.UpdateVersion = uint8(p.UnpackByte())
	create.SuccessCount = uint8(p.UnpackByte())

	return &create, p.Err()

}

func (*CreateUpdate) ValidRange(chain.Rules) (int64, int64) {
	// Returning -1, -1 means that the action is always valid.
	return -1, -1
}
