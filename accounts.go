package octaspace

import (
	"context"
	"math/big"
)

// Account holds OctaSpace account information.
type Account struct {
	AccountUUID string `json:"account_uuid"`
	Avatar      string `json:"avatar"`
	Balance     BigInt `json:"balance"`
	Email       string `json:"email"`
	RefCode     string `json:"ref_code"`
	UID         int64  `json:"uid"`
	Wallet      string `json:"wallet"`
}

// AccountsResource provides access to account endpoints.
// Obtain via [Client.Accounts].
type AccountsResource struct{ c *Client }

// Profile returns the authenticated account's information.
//
// GET /accounts
func (r *AccountsResource) Profile(ctx context.Context) (*Account, error) {
	var account Account
	return &account, r.c.get(ctx, "/accounts", nil, &account)
}

// Balance returns the authenticated account's balance in Wei.
//
// GET /accounts/balance
func (r *AccountsResource) Balance(ctx context.Context) (*big.Int, error) {
	var resp struct {
		Balance BigInt `json:"balance"`
	}
	if err := r.c.get(ctx, "/accounts/balance", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Balance.Int, nil
}

// GenerateWallet creates a new wallet for the authenticated account.
//
// POST /accounts
func (r *AccountsResource) GenerateWallet(ctx context.Context) (*Account, error) {
	var account Account
	return &account, r.c.post(ctx, "/accounts", nil, &account)
}
