// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package actions

// Note: Registry will error during initialization if a duplicate ID is assigned. We explicitly assign IDs to avoid accidental remapping.
const (
	burnAssetID     uint8 = 0
	closeOrderID    uint8 = 1
	createAssetID   uint8 = 2
	exportAssetID   uint8 = 3
	importAssetID   uint8 = 4
	createOrderID   uint8 = 5
	fillOrderID     uint8 = 6
	mintAssetID     uint8 = 7
	transferID      uint8 = 8
	createProjectID uint8 = 9
	createUpdateID  uint8 = 10
)

const (
	// TODO: tune this
	BurnComputeUnits        = 2
	CloseOrderComputeUnits  = 5
	CreateAssetComputeUnits = 10
	ExportAssetComputeUnits = 10
	ImportAssetComputeUnits = 10
	CreateOrderComputeUnits = 5
	NoFillOrderComputeUnits = 5
	FillOrderComputeUnits   = 15
	MintAssetComputeUnits   = 2
	TransferComputeUnits    = 1

	MaxSymbolSize   = 8
	MaxMemoSize     = 256
	MaxMetadataSize = 256
	MaxDecimals     = 9
)

// Project storage constants
const (
	ProjectNameUnits          = 32
	ProjectLogoUnits          = 100
	ProjectDescriptionUnits   = 100
	ProjectOwnerUnits         = 500
	CreateProjectComputeUnits = 5
)

// Update storage constants
const (
	ProjectTxIDUnits          = 100
	UpdateExecutableHashUnits = 100
	UpdateExecutableIPFSUrl   = 100
	ForDeviceNameUnits        = 100
	UpdateVersionUnits        = 1
	SuccessCountUnits         = 1
	CreateUpdateComputeUnits  = 5
)
