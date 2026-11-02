package octaspace

import "context"

// AccountsResource provides the account of the API key.
// Obtain via [Client.Accounts].
type AccountsResource struct{ c *Client }

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
