package octaspace

import "context"

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

// AccountBalance is the available balance of an account.
type AccountBalance struct {
	// Balance is deposits less charges, less what running sessions have
	// accrued but not been charged yet. It is negative while a session runs
	// past what the account has.
	Balance MicroUSD `json:"balance"`
	// Currency of Balance; "USD".
	Currency string `json:"currency"`
	// Decimals of Balance: Balance / 10^Decimals is the amount in Currency
	// (6 for micro-USD). Prefer it over assuming the unit.
	Decimals int `json:"decimals"`
}

// Balance returns the available balance of the authenticated account. It is
// read from a database replica, so a charge or a deposit may show up a moment
// later.
//
// GET /accounts/balance
func (r *AccountsResource) Balance(ctx context.Context) (*AccountBalance, error) {
	var b AccountBalance
	if err := r.c.get(ctx, "/accounts/balance", nil, &b); err != nil {
		return nil, err
	}
	return &b, nil
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
