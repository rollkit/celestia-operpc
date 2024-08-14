package share

import (
	coretypes "github.com/tendermint/tendermint/types"
)

// SplitBlobs splits the provided blobs into shares.
func SplitBlobs(blobs ...coretypes.Blob) ([]AppShare, error) {
	writer := NewSparseShareSplitter()
	for _, blob := range blobs {
		if err := writer.Write(blob); err != nil {
			return nil, err
		}
	}
	return writer.Export(), nil
}
