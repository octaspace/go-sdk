package octaspace

import (
	"context"
	"math/big"
)

// Account holds OctaSpace account information.
type Account struct {
	AccountUUID string `json:"account_uuid"`
	// Address is returned by POST /accounts when a wallet is generated.
	Address string `json:"address,omitempty"`
	Avatar  string `json:"avatar"`
	Balance BigInt `json:"balance"`
	Email   string `json:"email"`
	RefCode string `json:"ref_code"`
	UID     int64  `json:"uid"`
	Wallet  string `json:"wallet"`
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

// ProfileRaw returns the unmodified JSON body from the account-profile endpoint.
//
// GET /accounts
func (r *AccountsResource) ProfileRaw(ctx context.Context) ([]byte, error) {
	return r.c.getRaw(ctx, "/accounts")
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

// GenerateWallet creates a new wallet for the authenticated account. The
// documented POST response stores the generated wallet in Account.Address;
// full account responses continue to use Account.Wallet.
//
// POST /accounts
func (r *AccountsResource) GenerateWallet(ctx context.Context) (*Account, error) {
	var account Account
	return &account, r.c.post(ctx, "/accounts", nil, &account)
}
