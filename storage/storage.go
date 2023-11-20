// Copyright (C) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package storage

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	tconsts "hyper-updates/consts"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	smath "github.com/ava-labs/avalanchego/utils/math"
	"github.com/ava-labs/hypersdk/chain"
	"github.com/ava-labs/hypersdk/codec"
	"github.com/ava-labs/hypersdk/consts"
	"github.com/ava-labs/hypersdk/state"
)

type ReadState func(context.Context, [][]byte) ([][]byte, []error)

// Metadata
// 0x0/ (tx)
//   -> [txID] => timestamp
//
// State
// 0x0/ (balance)
//   -> [owner|asset] => balance
// 0x1/ (assets)
//   -> [asset] => metadataLen|metadata|supply|owner|warp
// 0x2/ (orders)
//   -> [txID] => in|out|rate|remaining|owner
// 0x3/ (loans)
//   -> [assetID|destination] => amount
// 0x4/ (hypersdk-height)
// 0x5/ (hypersdk-timestamp)
// 0x6/ (hypersdk-fee)
// 0x7/ (hypersdk-incoming warp)
// 0x8/ (hypersdk-outgoing warp)

const (
	// metaDB
	txPrefix = 0x0

	// stateDB
	balancePrefix      = 0x0
	assetPrefix        = 0x1
	orderPrefix        = 0x2
	loanPrefix         = 0x3
	heightPrefix       = 0x4
	timestampPrefix    = 0x5
	feePrefix          = 0x6
	incomingWarpPrefix = 0x7
	outgoingWarpPrefix = 0x8
	projectPrefix      = 0x9
	updatePrefix       = 0xA
)

const (
	BalanceChunks uint16 = 1
	AssetChunks   uint16 = 5
	OrderChunks   uint16 = 2
	LoanChunks    uint16 = 1

	ProjectNameChunks        uint16 = 32
	ProjectLogoChunks        uint16 = 100
	ProjectDescriptionChunks uint16 = 100
	ProjectOwnerChunks       uint16 = 500

	ProjectTxIDChunks             = 100
	UpdateExecutableHashChunks    = 100
	UpdateExecutableIPFSUrlChunks = 100
	ForDeviceNameChunks           = 100
	UpdateVersionUnitsChunks      = 1
	SuccessCountUnitsChunks       = 1
)

var (
	failureByte  = byte(0x0)
	successByte  = byte(0x1)
	heightKey    = []byte{heightPrefix}
	timestampKey = []byte{timestampPrefix}
	feeKey       = []byte{feePrefix}

	balanceKeyPool = sync.Pool{
		New: func() any {
			return make([]byte, 1+codec.AddressLen+consts.IDLen+consts.Uint16Len)
		},
	}
)

// [txPrefix] + [txID]
func TxKey(id ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen)
	k[0] = txPrefix
	copy(k[1:], id[:])
	return
}

func StoreTransaction(
	_ context.Context,
	db database.KeyValueWriter,
	id ids.ID,
	t int64,
	success bool,
	units chain.Dimensions,
	fee uint64,
) error {
	k := TxKey(id)
	v := make([]byte, consts.Uint64Len+1+chain.DimensionsLen+consts.Uint64Len)
	binary.BigEndian.PutUint64(v, uint64(t))
	if success {
		v[consts.Uint64Len] = successByte
	} else {
		v[consts.Uint64Len] = failureByte
	}
	copy(v[consts.Uint64Len+1:], units.Bytes())
	binary.BigEndian.PutUint64(v[consts.Uint64Len+1+chain.DimensionsLen:], fee)
	return db.Put(k, v)
}

func GetTransaction(
	_ context.Context,
	db database.KeyValueReader,
	id ids.ID,
) (bool, int64, bool, chain.Dimensions, uint64, error) {
	k := TxKey(id)
	v, err := db.Get(k)
	if errors.Is(err, database.ErrNotFound) {
		return false, 0, false, chain.Dimensions{}, 0, nil
	}
	if err != nil {
		return false, 0, false, chain.Dimensions{}, 0, err
	}
	t := int64(binary.BigEndian.Uint64(v))
	success := true
	if v[consts.Uint64Len] == failureByte {
		success = false
	}
	d, err := chain.UnpackDimensions(v[consts.Uint64Len+1 : consts.Uint64Len+1+chain.DimensionsLen])
	if err != nil {
		return false, 0, false, chain.Dimensions{}, 0, err
	}
	fee := binary.BigEndian.Uint64(v[consts.Uint64Len+1+chain.DimensionsLen:])
	return true, t, success, d, fee, nil
}

// [accountPrefix] + [address] + [asset]
func BalanceKey(addr codec.Address, asset ids.ID) (k []byte) {
	k = balanceKeyPool.Get().([]byte)
	k[0] = balancePrefix
	copy(k[1:], addr[:])
	copy(k[1+codec.AddressLen:], asset[:])
	binary.BigEndian.PutUint16(k[1+codec.AddressLen+consts.IDLen:], BalanceChunks)
	return
}

// If locked is 0, then account does not exist
func GetBalance(
	ctx context.Context,
	im state.Immutable,
	addr codec.Address,
	asset ids.ID,
) (uint64, error) {
	key, bal, _, err := getBalance(ctx, im, addr, asset)
	balanceKeyPool.Put(key)
	return bal, err
}

func getBalance(
	ctx context.Context,
	im state.Immutable,
	addr codec.Address,
	asset ids.ID,
) ([]byte, uint64, bool, error) {
	k := BalanceKey(addr, asset)
	bal, exists, err := innerGetBalance(im.GetValue(ctx, k))
	return k, bal, exists, err
}

// Used to serve RPC queries
func GetBalanceFromState(
	ctx context.Context,
	f ReadState,
	addr codec.Address,
	asset ids.ID,
) (uint64, error) {
	k := BalanceKey(addr, asset)
	values, errs := f(ctx, [][]byte{k})
	bal, _, err := innerGetBalance(values[0], errs[0])
	balanceKeyPool.Put(k)
	return bal, err
}

func innerGetBalance(
	v []byte,
	err error,
) (uint64, bool, error) {
	if errors.Is(err, database.ErrNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return binary.BigEndian.Uint64(v), true, nil
}

func SetBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
	asset ids.ID,
	balance uint64,
) error {
	k := BalanceKey(addr, asset)
	return setBalance(ctx, mu, k, balance)
}

func setBalance(
	ctx context.Context,
	mu state.Mutable,
	key []byte,
	balance uint64,
) error {
	return mu.Insert(ctx, key, binary.BigEndian.AppendUint64(nil, balance))
}

func DeleteBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
	asset ids.ID,
) error {
	return mu.Remove(ctx, BalanceKey(addr, asset))
}

func AddBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
	asset ids.ID,
	amount uint64,
	create bool,
) error {
	key, bal, exists, err := getBalance(ctx, mu, addr, asset)
	if err != nil {
		return err
	}
	// Don't add balance if account doesn't exist. This
	// can be useful when processing fee refunds.
	if !exists && !create {
		return nil
	}
	nbal, err := smath.Add64(bal, amount)
	if err != nil {
		return fmt.Errorf(
			"%w: could not add balance (asset=%s, bal=%d, addr=%v, amount=%d)",
			ErrInvalidBalance,
			asset,
			bal,
			codec.MustAddressBech32(tconsts.HRP, addr),
			amount,
		)
	}
	return setBalance(ctx, mu, key, nbal)
}

func SubBalance(
	ctx context.Context,
	mu state.Mutable,
	addr codec.Address,
	asset ids.ID,
	amount uint64,
) error {
	key, bal, _, err := getBalance(ctx, mu, addr, asset)
	if err != nil {
		return err
	}
	nbal, err := smath.Sub(bal, amount)
	if err != nil {
		return fmt.Errorf(
			"%w: could not subtract balance (asset=%s, bal=%d, addr=%v, amount=%d)",
			ErrInvalidBalance,
			asset,
			bal,
			codec.MustAddressBech32(tconsts.HRP, addr),
			amount,
		)
	}
	if nbal == 0 {
		// If there is no balance left, we should delete the record instead of
		// setting it to 0.
		return mu.Remove(ctx, key)
	}
	return setBalance(ctx, mu, key, nbal)
}

// [assetPrefix] + [address]
func AssetKey(asset ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen+consts.Uint16Len)
	k[0] = assetPrefix
	copy(k[1:], asset[:])
	binary.BigEndian.PutUint16(k[1+consts.IDLen:], AssetChunks)
	return
}

// Used to serve RPC queries
func GetAssetFromState(
	ctx context.Context,
	f ReadState,
	asset ids.ID,
) (bool, []byte, uint8, []byte, uint64, codec.Address, bool, error) {
	values, errs := f(ctx, [][]byte{AssetKey(asset)})
	return innerGetAsset(values[0], errs[0])
}

func GetAsset(
	ctx context.Context,
	im state.Immutable,
	asset ids.ID,
) (bool, []byte, uint8, []byte, uint64, codec.Address, bool, error) {
	k := AssetKey(asset)
	return innerGetAsset(im.GetValue(ctx, k))
}

func innerGetAsset(
	v []byte,
	err error,
) (bool, []byte, uint8, []byte, uint64, codec.Address, bool, error) {
	if errors.Is(err, database.ErrNotFound) {
		return false, nil, 0, nil, 0, codec.EmptyAddress, false, nil
	}
	if err != nil {
		return false, nil, 0, nil, 0, codec.EmptyAddress, false, err
	}
	symbolLen := binary.BigEndian.Uint16(v)
	symbol := v[consts.Uint16Len : consts.Uint16Len+symbolLen]
	decimals := v[consts.Uint16Len+symbolLen]
	metadataLen := binary.BigEndian.Uint16(v[consts.Uint16Len+symbolLen+consts.Uint8Len:])
	metadata := v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len : consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen]
	supply := binary.BigEndian.Uint64(v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen:])
	var addr codec.Address
	copy(addr[:], v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len:])
	warp := v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len+codec.AddressLen] == 0x1
	return true, symbol, decimals, metadata, supply, addr, warp, nil
}

func SetAsset(
	ctx context.Context,
	mu state.Mutable,
	asset ids.ID,
	symbol []byte,
	decimals uint8,
	metadata []byte,
	supply uint64,
	owner codec.Address,
	warp bool,
) error {
	k := AssetKey(asset)
	symbolLen := len(symbol)
	metadataLen := len(metadata)
	v := make([]byte, consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len+codec.AddressLen+1)
	binary.BigEndian.PutUint16(v, uint16(symbolLen))
	copy(v[consts.Uint16Len:], symbol)
	v[consts.Uint16Len+symbolLen] = decimals
	binary.BigEndian.PutUint16(v[consts.Uint16Len+symbolLen+consts.Uint8Len:], uint16(metadataLen))
	copy(v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len:], metadata)
	binary.BigEndian.PutUint64(v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen:], supply)
	copy(v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len:], owner[:])
	b := byte(0x0)
	if warp {
		b = 0x1
	}
	v[consts.Uint16Len+symbolLen+consts.Uint8Len+consts.Uint16Len+metadataLen+consts.Uint64Len+codec.AddressLen] = b
	return mu.Insert(ctx, k, v)
}

func DeleteAsset(ctx context.Context, mu state.Mutable, asset ids.ID) error {
	k := AssetKey(asset)
	return mu.Remove(ctx, k)
}

// [orderPrefix] + [txID]
func OrderKey(txID ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen+consts.Uint16Len)
	k[0] = orderPrefix
	copy(k[1:], txID[:])
	binary.BigEndian.PutUint16(k[1+consts.IDLen:], OrderChunks)
	return
}

func SetOrder(
	ctx context.Context,
	mu state.Mutable,
	txID ids.ID,
	in ids.ID,
	inTick uint64,
	out ids.ID,
	outTick uint64,
	supply uint64,
	owner codec.Address,
) error {
	k := OrderKey(txID)
	v := make([]byte, consts.IDLen*2+consts.Uint64Len*3+codec.AddressLen)
	copy(v, in[:])
	binary.BigEndian.PutUint64(v[consts.IDLen:], inTick)
	copy(v[consts.IDLen+consts.Uint64Len:], out[:])
	binary.BigEndian.PutUint64(v[consts.IDLen*2+consts.Uint64Len:], outTick)
	binary.BigEndian.PutUint64(v[consts.IDLen*2+consts.Uint64Len*2:], supply)
	copy(v[consts.IDLen*2+consts.Uint64Len*3:], owner[:])
	return mu.Insert(ctx, k, v)
}

func GetOrder(
	ctx context.Context,
	im state.Immutable,
	order ids.ID,
) (
	bool, // exists
	ids.ID, // in
	uint64, // inTick
	ids.ID, // out
	uint64, // outTick
	uint64, // remaining
	codec.Address, // owner
	error,
) {
	k := OrderKey(order)
	v, err := im.GetValue(ctx, k)
	return innerGetOrder(v, err)
}

// Used to serve RPC queries
func GetOrderFromState(
	ctx context.Context,
	f ReadState,
	order ids.ID,
) (
	bool, // exists
	ids.ID, // in
	uint64, // inTick
	ids.ID, // out
	uint64, // outTick
	uint64, // remaining
	codec.Address, // owner
	error,
) {
	values, errs := f(ctx, [][]byte{OrderKey(order)})
	return innerGetOrder(values[0], errs[0])
}

func innerGetOrder(v []byte, err error) (
	bool, // exists
	ids.ID, // in
	uint64, // inTick
	ids.ID, // out
	uint64, // outTick
	uint64, // remaining
	codec.Address, // owner
	error,
) {
	if errors.Is(err, database.ErrNotFound) {
		return false, ids.Empty, 0, ids.Empty, 0, 0, codec.EmptyAddress, nil
	}
	if err != nil {
		return false, ids.Empty, 0, ids.Empty, 0, 0, codec.EmptyAddress, err
	}
	var in ids.ID
	copy(in[:], v[:consts.IDLen])
	inTick := binary.BigEndian.Uint64(v[consts.IDLen:])
	var out ids.ID
	copy(out[:], v[consts.IDLen+consts.Uint64Len:consts.IDLen*2+consts.Uint64Len])
	outTick := binary.BigEndian.Uint64(v[consts.IDLen*2+consts.Uint64Len:])
	supply := binary.BigEndian.Uint64(v[consts.IDLen*2+consts.Uint64Len*2:])
	var owner codec.Address
	copy(owner[:], v[consts.IDLen*2+consts.Uint64Len*3:])
	return true, in, inTick, out, outTick, supply, owner, nil
}

func DeleteOrder(ctx context.Context, mu state.Mutable, order ids.ID) error {
	k := OrderKey(order)
	return mu.Remove(ctx, k)
}

// [loanPrefix] + [asset] + [destination]
func LoanKey(asset ids.ID, destination ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen*2+consts.Uint16Len)
	k[0] = loanPrefix
	copy(k[1:], asset[:])
	copy(k[1+consts.IDLen:], destination[:])
	binary.BigEndian.PutUint16(k[1+consts.IDLen*2:], LoanChunks)
	return
}

// Used to serve RPC queries
func GetLoanFromState(
	ctx context.Context,
	f ReadState,
	asset ids.ID,
	destination ids.ID,
) (uint64, error) {
	values, errs := f(ctx, [][]byte{LoanKey(asset, destination)})
	return innerGetLoan(values[0], errs[0])
}

func innerGetLoan(v []byte, err error) (uint64, error) {
	if errors.Is(err, database.ErrNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(v), nil
}

func GetLoan(
	ctx context.Context,
	im state.Immutable,
	asset ids.ID,
	destination ids.ID,
) (uint64, error) {
	k := LoanKey(asset, destination)
	v, err := im.GetValue(ctx, k)
	return innerGetLoan(v, err)
}

func SetLoan(
	ctx context.Context,
	mu state.Mutable,
	asset ids.ID,
	destination ids.ID,
	amount uint64,
) error {
	k := LoanKey(asset, destination)
	return mu.Insert(ctx, k, binary.BigEndian.AppendUint64(nil, amount))
}

func AddLoan(
	ctx context.Context,
	mu state.Mutable,
	asset ids.ID,
	destination ids.ID,
	amount uint64,
) error {
	loan, err := GetLoan(ctx, mu, asset, destination)
	if err != nil {
		return err
	}
	nloan, err := smath.Add64(loan, amount)
	if err != nil {
		return fmt.Errorf(
			"%w: could not add loan (asset=%s, destination=%s, amount=%d)",
			ErrInvalidBalance,
			asset,
			destination,
			amount,
		)
	}
	return SetLoan(ctx, mu, asset, destination, nloan)
}

func SubLoan(
	ctx context.Context,
	mu state.Mutable,
	asset ids.ID,
	destination ids.ID,
	amount uint64,
) error {
	loan, err := GetLoan(ctx, mu, asset, destination)
	if err != nil {
		return err
	}
	nloan, err := smath.Sub(loan, amount)
	if err != nil {
		return fmt.Errorf(
			"%w: could not subtract loan (asset=%s, destination=%s, amount=%d)",
			ErrInvalidBalance,
			asset,
			destination,
			amount,
		)
	}
	if nloan == 0 {
		// If there is no balance left, we should delete the record instead of
		// setting it to 0.
		return mu.Remove(ctx, LoanKey(asset, destination))
	}
	return SetLoan(ctx, mu, asset, destination, nloan)
}

func HeightKey() (k []byte) {
	return heightKey
}

func TimestampKey() (k []byte) {
	return timestampKey
}

func FeeKey() (k []byte) {
	return feeKey
}

func IncomingWarpKeyPrefix(sourceChainID ids.ID, msgID ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen*2)
	k[0] = incomingWarpPrefix
	copy(k[1:], sourceChainID[:])
	copy(k[1+consts.IDLen:], msgID[:])
	return k
}

func OutgoingWarpKeyPrefix(txID ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen)
	k[0] = outgoingWarpPrefix
	copy(k[1:], txID[:])
	return k
}

// [projectPrefix] + [address]
func ProjectKey(project ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen+consts.Uint16Len)
	k[0] = projectPrefix
	copy(k[1:], project[:])
	binary.BigEndian.PutUint16(k[1+consts.IDLen:], ProjectDescriptionChunks)
	return
}

func SetProject(
	ctx context.Context,
	mu state.Mutable,
	project ids.ID,
	project_name []byte,
	project_description []byte,
	project_owner []byte,
	logo []byte,
) error {

	k := ProjectKey(project)

	v := make([]byte, ProjectNameChunks+ProjectDescriptionChunks+ProjectOwnerChunks+ProjectLogoChunks)

	// saddr, _ := codec.AddressBech32(tconsts.HRP, owner)

	copy(v[:ProjectNameChunks], project_name[:])
	copy(v[ProjectNameChunks:ProjectNameChunks+ProjectDescriptionChunks], project_description[:])
	copy(v[ProjectNameChunks+ProjectDescriptionChunks:ProjectNameChunks+ProjectDescriptionChunks+ProjectOwnerChunks], project_owner[:])
	copy(v[ProjectNameChunks+ProjectDescriptionChunks+ProjectOwnerChunks:ProjectNameChunks+ProjectDescriptionChunks+ProjectOwnerChunks+ProjectLogoChunks], logo[:])
	fmt.Println("Project Added to the Chain State")
	return mu.Insert(ctx, k, v)
}

func GetProjectFromState(
	ctx context.Context,
	f ReadState,
	project ids.ID,
) (bool, ProjectData, error) {

	k := ProjectKey(project)
	v, errs := f(ctx, [][]byte{k})

	if errors.Is(errs[0], database.ErrNotFound) {
		return false, ProjectData{}, nil
	}
	if errs[0] != nil {
		return false, ProjectData{}, nil
	}

	return true, ProjectData{
		Key:                hex.EncodeToString(k),
		ProjectName:        v[0][:ProjectNameChunks],
		ProjectDescription: v[0][ProjectNameChunks : ProjectNameChunks+ProjectDescriptionChunks],
		ProjectOwner:       v[0][ProjectNameChunks+ProjectDescriptionChunks : ProjectNameChunks+ProjectDescriptionChunks+ProjectOwnerChunks],
		Logo:               v[0][ProjectNameChunks+ProjectDescriptionChunks+ProjectOwnerChunks : ProjectNameChunks+ProjectDescriptionChunks+ProjectOwnerChunks+ProjectLogoChunks],
	}, errs[0]
}

// [updatePrefix] + [address]
func UpdateKey(update ids.ID) (k []byte) {
	k = make([]byte, 1+consts.IDLen+consts.Uint16Len)
	k[0] = updatePrefix
	copy(k[1:], update[:])
	binary.BigEndian.PutUint16(k[1+consts.IDLen:], UpdateExecutableHashChunks)
	return
}

// ProjectTxID          []byte `json:"project_id"` // reference to Project
//
//	UpdateExecutableHash []byte `json:"executable_hash"`
//	UpdateIPFSUrl       []byte `json:"executable_ipfs_url"`
//	ForDeviceName        []byte `json:"for_device_name"`
//	UpdateVersion        uint8  `json:"version"`
//	SuccessCount         uint8  `json:"success_count"`
func SetUpdate(
	ctx context.Context,
	mu state.Mutable,
	update ids.ID,
	project_id []byte,
	executable_hash []byte,
	executable_ipfs_url []byte,
	for_device_name []byte,
	version uint8,
	success_count uint8,
) error {

	k := UpdateKey(update)

	v := make([]byte,
		ProjectTxIDChunks+
			UpdateExecutableHashChunks+
			UpdateExecutableIPFSUrlChunks+
			ForDeviceNameChunks+
			UpdateVersionUnitsChunks+
			SuccessCountUnitsChunks)

	// saddr, _ := codec.AddressBech32(tconsts.HRP, owner)

	copy(v[:ProjectTxIDChunks], project_id[:])

	copy(v[ProjectTxIDChunks:ProjectTxIDChunks+UpdateExecutableHashChunks], executable_hash[:])

	copy(v[ProjectTxIDChunks+UpdateExecutableHashChunks:ProjectTxIDChunks+UpdateExecutableHashChunks+UpdateExecutableIPFSUrlChunks], executable_ipfs_url[:])

	copy(v[ProjectTxIDChunks+UpdateExecutableHashChunks+UpdateExecutableIPFSUrlChunks:ProjectTxIDChunks+UpdateExecutableHashChunks+UpdateExecutableIPFSUrlChunks+ForDeviceNameChunks], for_device_name[:])

	v[ProjectTxIDChunks+UpdateExecutableHashChunks+UpdateExecutableIPFSUrlChunks+ForDeviceNameChunks] = version
	v[ProjectTxIDChunks+UpdateExecutableHashChunks+UpdateExecutableIPFSUrlChunks+ForDeviceNameChunks+UpdateVersionUnitsChunks] = success_count

	fmt.Println("Update Added to the Chain State")
	return mu.Insert(ctx, k, v)
}

func GetUpdateFromState(
	ctx context.Context,
	f ReadState,
	update ids.ID,
) (bool, UpdateData, error) {

	k := UpdateKey(update)
	v, errs := f(ctx, [][]byte{k})

	if errors.Is(errs[0], database.ErrNotFound) {
		return false, UpdateData{}, nil
	}
	if errs[0] != nil {
		return false, UpdateData{}, nil
	}

	return true, UpdateData{
		Key:                  hex.EncodeToString(k),
		ProjectTxID:          v[0][:ProjectTxIDChunks],
		UpdateExecutableHash: v[0][ProjectTxIDChunks : ProjectTxIDChunks+UpdateExecutableHashChunks],
		UpdateIPFSUrl:        v[0][ProjectTxIDChunks+UpdateExecutableHashChunks : ProjectTxIDChunks+UpdateExecutableHashChunks+UpdateExecutableIPFSUrlChunks],
		ForDeviceName:        v[0][ProjectTxIDChunks+UpdateExecutableHashChunks+UpdateExecutableIPFSUrlChunks : ProjectTxIDChunks+UpdateExecutableHashChunks+UpdateExecutableIPFSUrlChunks+ForDeviceNameChunks],
		UpdateVersion:        v[0][ProjectTxIDChunks+UpdateExecutableHashChunks+UpdateExecutableIPFSUrlChunks+ForDeviceNameChunks],
		SuccessCount:         v[0][ProjectTxIDChunks+UpdateExecutableHashChunks+UpdateExecutableIPFSUrlChunks+ForDeviceNameChunks+UpdateVersionUnitsChunks],
	}, errs[0]
}
