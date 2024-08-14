package main

import (
	"bytes"
	"context"
	"fmt"

	client "github.com/celestiaorg/celestia-openrpc"
	"github.com/celestiaorg/celestia-openrpc/types/blob"
	"github.com/celestiaorg/celestia-openrpc/types/share"
	"github.com/celestiaorg/rsmt2d"
)

func main() {
	ctx := context.Background()
	url := "ws://localhost:26658"
	token := ""

	err := SubmitBlob(ctx, url, token)
	if err != nil {
		fmt.Println(err)
	}

	eds, err := GetEDS(ctx, url, token, 500)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(eds.FlattenedODS())

	SubscribeHeaders(ctx, url, token)
}

// SubmitBlob submits a blob containing "Hello, World!" to the 0xDEADBEEF namespace. It uses the default signer on the running node.
func SubmitBlob(ctx context.Context, url string, token string) error {
	client, err := client.NewClient(ctx, url, token)
	if err != nil {
		return err
	}

	// let's post to 0xDEADBEEF namespace
	namespace, err := share.NewBlobNamespaceV0([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	if err != nil {
		return err
	}

	// create a blob
	helloWorldBlob, err := blob.NewBlobV0(namespace, []byte("Hello, World!"))
	if err != nil {
		return err
	}

	// submit the blob to the network
	height, err := client.Blob.Submit(ctx, []*blob.Blob{helloWorldBlob}, blob.NewSubmitOptions())
	if err != nil {
		return err
	}

	fmt.Printf("Blob was included at height %d\n", height)

	// fetch the blob back from the network
	retrievedBlobs, err := client.Blob.GetAll(ctx, height, []share.Namespace{namespace})
	if err != nil {
		return err
	}

	fmt.Printf("Blobs are equal? %v\n", bytes.Equal(helloWorldBlob.Commitment, retrievedBlobs[0].Commitment))
	return nil
}

// SubscribeHeaders subscribes to new headers and fetches all blobs at the height of the new header in the 0xDEADBEEF namespace.
func SubscribeHeaders(ctx context.Context, url string, token string) error {
	client, err := client.NewClient(ctx, url, token)
	if err != nil {
		return err
	}

	// create a namespace to filter blobs with
	namespace, err := share.NewBlobNamespaceV0([]byte{0xDE, 0xAD, 0xBE, 0xEF})
	if err != nil {
		return err
	}

	// subscribe to new headers using a <-chan *header.ExtendedHeader channel
	headerChan, err := client.Header.Subscribe(ctx)
	if err != nil {
		return err
	}

	for {
		select {
		case header := <-headerChan:
			// fetch all blobs at the height of the new header
			blobs, err := client.Blob.GetAll(context.TODO(), header.Height(), []share.Namespace{namespace})
			if err != nil {
				fmt.Printf("Error fetching blobs: %v\n", err)
			}

			fmt.Printf("Found %d blobs at height %d in 0xDEADBEEF namespace\n", len(blobs), header.Height())
		case <-ctx.Done():
			return nil
		}
	}
}

// GetEDS fetches the EDS at the given height.
func GetEDS(ctx context.Context, url string, token string, height uint64) (*rsmt2d.ExtendedDataSquare, error) {
	client, err := client.NewClient(ctx, url, token)
	if err != nil {
		return nil, err
	}

	// First get the header of the block you want to fetch the EDS from
	header, err := client.Header.GetByHeight(ctx, height)
	if err != nil {
		return nil, err
	}

	// Fetch the EDS
	return client.Share.GetEDS(ctx, header)
}
