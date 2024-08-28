package state

import (
	"context"

	"github.com/celestiaorg/celestia-openrpc/types/blob"
)

type API struct {
	// AccountAddress retrieves the address of the node's account/signer
	AccountAddress func(ctx context.Context) (Address, error) `perm:"read"`
	// Balance retrieves the Celestia coin balance for the node's account/signer
	// and verifies it against the corresponding block's AppHash.
	Balance func(ctx context.Context) (*Balance, error) `perm:"read"`
	// BalanceForAddress retrieves the Celestia coin balance for the given address and verifies
	// the returned balance against the corresponding block's AppHash.
	//
	// NOTE: the balance returned is the balance reported by the block right before
	// the node's current head (head-1). This is due to the fact that for block N, the block's
	// `AppHash` is the result of applying the previous block's transaction list.
	BalanceForAddress func(ctx context.Context, addr Address) (*Balance, error) `perm:"read"`
	// Transfer sends the given amount of coins from default wallet of the node to the given account
	// address.
	Transfer func(
		ctx context.Context,
		to AccAddress,
		amount Int,
		config *TxConfig,
	) (*TxResponse, error) `perm:"write"`
	// SubmitPayForBlob builds, signs and submits a PayForBlob transaction.
	SubmitPayForBlob func(
		ctx context.Context,
		blobs []*blob.Blob,
		config *TxConfig,
	) (*TxResponse, error) `perm:"write"`
	// CancelUnbondingDelegation cancels a user's pending undelegation from a validator.
	CancelUnbondingDelegation func(
		ctx context.Context,
		valAddr ValAddress,
		amount, height Int,
		config *TxConfig,
	) (*TxResponse, error) `perm:"write"`
	// BeginRedelegate sends a user's delegated tokens to a new validator for redelegation.
	BeginRedelegate func(
		ctx context.Context,
		srcValAddr,
		dstValAddr ValAddress,
		amount Int,
		config *TxConfig,
	) (*TxResponse, error) `perm:"write"`
	// Undelegate undelegates a user's delegated tokens, unbonding them from the current validator.
	Undelegate func(
		ctx context.Context,
		delAddr ValAddress,
		amount Int,
		config *TxConfig,
	) (*TxResponse, error) `perm:"write"`
	// Delegate sends a user's liquid tokens to a validator for delegation.
	Delegate func(
		ctx context.Context,
		delAddr ValAddress,
		amount Int,
		config *TxConfig,
	) (*TxResponse, error) `perm:"write"`
	// QueryDelegation retrieves the delegation information between a delegator and a validator.
	QueryDelegation func(
		ctx context.Context,
		valAddr ValAddress,
	) (*QueryDelegationResponse, error) `perm:"read"`
	// QueryUnbonding retrieves the unbonding status between a delegator and a validator.
	QueryUnbonding func(
		ctx context.Context,
		valAddr ValAddress,
	) (*QueryUnbondingDelegationResponse, error) `perm:"read"`
	// QueryRedelegations retrieves the status of the redelegations between a delegator and a validator.
	QueryRedelegations func(
		ctx context.Context,
		srcValAddr,
		dstValAddr ValAddress,
	) (*QueryRedelegationsResponse, error) `perm:"read"`
	// GrantFee grants the given amount of fee to the given grantee.
	GrantFee func(
		ctx context.Context,
		grantee AccAddress,
		amount Int,
		config *TxConfig,
	) (*TxResponse, error) `perm:"write"`
	// RevokeGrantFee revokes the granted fee from the given grantee.
	RevokeGrantFee func(
		ctx context.Context,
		grantee AccAddress,
		config *TxConfig,
	) (*TxResponse, error) `perm:"write"`
}
