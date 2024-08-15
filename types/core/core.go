package core

import (
	"bytes"
	"fmt"
	"time"

	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/merkle"
	cmbytes "github.com/cometbft/cometbft/libs/bytes"
	cmversion "github.com/cometbft/cometbft/proto/tendermint/version"

	"github.com/celestiaorg/rsmt2d"
)

// Header defines the structure of a Tendermint block header.
// NOTE: changes to the Header should be duplicated in:
// - header.Hash()
// - abci.Header
// - https://github.com/tendermint/tendermint/blob/v0.34.x/spec/blockchain/blockchain.md
type Header struct {
	// basic block info
	Version cmversion.Consensus `json:"version"`
	ChainID string              `json:"chain_id"`
	Height  int64               `json:"height,string"`
	Time    time.Time           `json:"time"`

	// prev block info
	LastBlockID BlockID `json:"last_block_id"`

	// hashes of block data
	LastCommitHash cmbytes.HexBytes `json:"last_commit_hash"` // commit from validators from the last block
	DataHash       cmbytes.HexBytes `json:"data_hash"`        // transactions

	// hashes from the app output from the prev block
	ValidatorsHash     cmbytes.HexBytes `json:"validators_hash"`      // validators for the current block
	NextValidatorsHash cmbytes.HexBytes `json:"next_validators_hash"` // validators for the next block
	ConsensusHash      cmbytes.HexBytes `json:"consensus_hash"`       // consensus params for current block
	AppHash            cmbytes.HexBytes `json:"app_hash"`             // state after txs from the previous block
	// root hash of all results from the txs from the previous block
	// see `deterministicResponseDeliverTx` to understand which parts of a tx is hashed into here
	LastResultsHash cmbytes.HexBytes `json:"last_results_hash"`

	// consensus info
	EvidenceHash    cmbytes.HexBytes `json:"evidence_hash"`    // evidence included in the block
	ProposerAddress Address          `json:"proposer_address"` // original proposer of the block
}

// BlockID
type BlockID struct {
	Hash          cmbytes.HexBytes `json:"hash"`
	PartSetHeader PartSetHeader    `json:"parts"`
}

type PartSetHeader struct {
	Total uint32           `json:"total"`
	Hash  cmbytes.HexBytes `json:"hash"`
}

type CoreBlob struct {
	// NamespaceVersion is the version of the namespace. Used in conjunction
	// with NamespaceID to determine the namespace of this blob.
	NamespaceVersion uint8

	// NamespaceID defines the namespace ID of this blob. Used in conjunction
	// with NamespaceVersion to determine the namespace of this blob.
	NamespaceID []byte

	// Data is the actual data of the blob.
	// (e.g. a block of a virtual sidechain).
	Data []byte

	// ShareVersion is the version of the share format that this blob should use
	// when encoded into shares.
	ShareVersion uint8
}

// Address is hex bytes.
type Address = crypto.Address

// Commit contains the evidence that a block was committed by a set of validators.
// NOTE: Commit is empty for height 1, but never nil.
type Commit struct {
	// NOTE: The signatures are in order of address to preserve the bonded
	// ValidatorSet order.
	// Any peer with a block can gossip signatures by index with a peer without
	// recalculating the active ValidatorSet.
	Height     int64       `json:"height"`
	Round      int32       `json:"round"`
	BlockID    BlockID     `json:"block_id"`
	Signatures []CommitSig `json:"signatures"`
}

// CommitSig is a part of the Vote included in a Commit.
type CommitSig struct {
	BlockIDFlag      BlockIDFlag `json:"block_id_flag"`
	ValidatorAddress Address     `json:"validator_address"`
	Timestamp        time.Time   `json:"timestamp"`
	Signature        []byte      `json:"signature"`
}

// BlockIDFlag indicates which BlockID the signature is for.
type BlockIDFlag byte

// ValidatorSet represent a set of *Validator at a given height.
//
// The validators can be fetched by address or index.
// The index is in order of .VotingPower, so the indices are fixed for all
// rounds of a given blockchain height - ie. the validators are sorted by their
// voting power (descending). Secondary index - .Address (ascending).
//
// On the other hand, the .ProposerPriority of each validator and the
// designated .GetProposer() of a set changes every round, upon calling
// .IncrementProposerPriority().
//
// NOTE: Not goroutine-safe.
// NOTE: All get/set to validators should copy the value for safety.
type ValidatorSet struct {
	// NOTE: persisted via reflect, must be exported.
	Validators []*Validator `json:"validators"`
	Proposer   *Validator   `json:"proposer"`
}

// Volatile state for each Validator
// NOTE: The ProposerPriority is not included in Validator.Hash();
// make sure to update that method if changes are made here
type Validator struct {
	Address     Address       `json:"address"`
	PubKey      crypto.PubKey `json:"pub_key"`
	VotingPower int64         `json:"voting_power"`

	ProposerPriority int64 `json:"proposer_priority"`
}

// DataAvailabilityHeader (DAHeader) contains the row and column roots of the
// erasure coded version of the data in Block.Data. The original Block.Data is
// split into shares and arranged in a square of width squareSize. Then, this
// square is "extended" into an extended data square (EDS) of width 2*squareSize
// by applying Reed-Solomon encoding. For details see Section 5.2 of
// https://arxiv.org/abs/1809.09044 or the Celestia specification:
// https://github.com/celestiaorg/celestia-specs/blob/master/src/specs/data_structures.md#availabledataheader
type DataAvailabilityHeader struct {
	// RowRoot_j = root((M_{j,1} || M_{j,2} || ... || M_{j,2k} ))
	RowRoots [][]byte `json:"row_roots"`
	// ColumnRoot_j = root((M_{1,j} || M_{2,j} || ... || M_{2k,j} ))
	ColumnRoots [][]byte `json:"column_roots"`
	// hash is the Merkle root of the row and column roots. This field is the
	// memoized result from `Hash()`.
	hash []byte
}

// NewDataAvailabilityHeader generates a DataAvailability header using the
// provided extended data square.
func NewDataAvailabilityHeader(eds *rsmt2d.ExtendedDataSquare) (DataAvailabilityHeader, error) {
	rowRoots, err := eds.RowRoots()
	if err != nil {
		return DataAvailabilityHeader{}, err
	}
	colRoots, err := eds.ColRoots()
	if err != nil {
		return DataAvailabilityHeader{}, err
	}

	dah := DataAvailabilityHeader{
		RowRoots:    rowRoots,
		ColumnRoots: colRoots,
	}

	// Generate the hash of the data using the new roots
	dah.Hash()

	return dah, nil
}

// String returns hex representation of merkle hash of the DAHeader.
func (dah *DataAvailabilityHeader) String() string {
	if dah == nil {
		return "<nil DAHeader>"
	}
	return fmt.Sprintf("%X", dah.Hash())
}

// Equals checks equality of two DAHeaders.
func (dah *DataAvailabilityHeader) Equals(to *DataAvailabilityHeader) bool {
	return bytes.Equal(dah.Hash(), to.Hash())
}

// Hash computes the Merkle root of the row and column roots. Hash memoizes the
// result in `DataAvailabilityHeader.hash`.
func (dah *DataAvailabilityHeader) Hash() []byte {
	if dah == nil {
		return merkle.HashFromByteSlices(nil)
	}
	if len(dah.hash) != 0 {
		return dah.hash
	}

	rowsCount := len(dah.RowRoots)
	slices := make([][]byte, rowsCount+rowsCount)
	copy(slices[0:rowsCount], dah.RowRoots)
	copy(slices[rowsCount:], dah.ColumnRoots)
	// The single data root is computed using a simple binary merkle tree.
	// Effectively being root(rowRoots || columnRoots):
	dah.hash = merkle.HashFromByteSlices(slices)
	return dah.hash
}
