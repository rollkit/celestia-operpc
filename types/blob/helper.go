package blob

import (
	"bytes"
	"sort"

	"github.com/celestiaorg/celestia-openrpc/types/core"
	"github.com/celestiaorg/celestia-openrpc/types/share"
)

// BlobsToShares accepts blobs and convert them to the Shares.
func BlobsToShares(blobs ...*Blob) ([]share.Share, error) {
	b := make([]core.CoreBlob, len(blobs))
	for i, blob := range blobs {
		namespace := blob.Namespace()
		b[i] = core.CoreBlob{
			NamespaceVersion: namespace.Version,
			NamespaceID:      namespace.ID,
			Data:             blob.Data,
			//nolint:gosec
			ShareVersion: uint8(blob.ShareVersion),
		}
	}

	sort.Slice(b, func(i, j int) bool {
		val := bytes.Compare(b[i].NamespaceID, b[j].NamespaceID)
		return val < 0
	})

	rawShares, err := share.SplitBlobs(b...)
	if err != nil {
		return nil, err
	}
	return share.ToBytes(rawShares), nil
}
