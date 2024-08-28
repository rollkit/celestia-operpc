package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/suite"

	"github.com/celestiaorg/celestia-openrpc/types/blob"
	"github.com/celestiaorg/celestia-openrpc/types/share"
)

type TestSuite struct {
	suite.Suite

	pool     *dockertest.Pool
	resource *dockertest.Resource

	token string
}

func (t *TestSuite) SetupSuite() {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Failf("Could not construct docker pool", "error: %v\n", err)
	}
	t.pool = pool

	// uses pool to try to connect to Docker
	err = pool.Client.Ping()
	if err != nil {
		t.Failf("Could not connect to Docker", "error: %v\n", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.Run("ghcr.io/rollkit/local-celestia-devnet", "a7aa7c3", []string{})
	if err != nil {
		t.Failf("Could not start resource", "error: %v\n", err)
	}
	t.resource = resource

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	pool.MaxWait = 60 * time.Second
	if err := pool.Retry(func() error {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/balance", resource.GetPort("26659/tcp")))
		if err != nil {
			return err
		}
		bz, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return err
		}
		if strings.Contains(string(bz), "error") {
			return errors.New(string(bz))
		}
		return nil
	}); err != nil {
		log.Fatalf("Could not start local-celestia-devnet: %s", err)
	}

	opts := dockertest.ExecOptions{}
	buf := new(bytes.Buffer)
	opts.StdOut = buf
	opts.StdErr = buf
	_, err = resource.Exec([]string{"/bin/celestia", "bridge", "auth", "admin", "--node.store", "/home/celestia/bridge"}, opts)
	if err != nil {
		t.Failf("Could not get auth token", "error: %v\n", err)
	}

	t.token = strings.TrimSpace(buf.String())
}

func (t *TestSuite) TearDownSuite() {
	if err := t.pool.Purge(t.resource); err != nil {
		t.Failf("failed to purge docker resource", "error: %v\n", err)
	}
}

func TestIntegrationTestSuite(t *testing.T) {
	t.Skip("skipping integration test suite until we have a better way to run it")
	suite.Run(t, new(TestSuite))
}

// TestClient is a basic smoke test,  ensuring that client can execute simple methods.
func (t *TestSuite) TestClient() {
	client, err := NewClient(context.Background(), t.getRPCAddress(), t.token)
	t.NoError(err)
	defer client.Close()

	t.NotNil(client)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	resp, err := client.Header.NetworkHead(ctx)
	t.NotZero(resp)
	t.NoError(err)

	info, err := client.Node.Info(ctx)
	t.NoError(err)
	t.NotEmpty(info.APIVersion)
}

// TestRoundTrip tests
func (t *TestSuite) TestRoundTrip() {
	client, err := NewClient(context.Background(), t.getRPCAddress(), t.token)
	t.Require().NoError(err)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	namespace, err := share.NewBlobNamespaceV0([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	t.Require().NoError(err)
	t.Require().NotEmpty(namespace)

	data := []byte("hello world")
	blobBlob, err := blob.NewBlobV0(namespace, data)
	t.Require().NoError(err)

	// write blob to DA
	height, err := client.Blob.Submit(ctx, []*blob.Blob{blobBlob}, blob.NewSubmitOptions())
	t.Require().NoError(err)
	t.Require().NotZero(height)

	ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = client.Header.WaitForHeight(context.Background(), height)
	t.Require().NoError(err)

	// retrieve data back from DA
	daBlob, err := client.Blob.Get(ctx, height, namespace, blobBlob.Commitment)
	t.Require().NoError(err)
	t.Require().NotNil(daBlob)
	t.Equal(data, daBlob.Data)
}

func (t *TestSuite) getRPCAddress() string {
	return fmt.Sprintf("http://localhost:%s", t.resource.GetPort("26658/tcp"))
}
