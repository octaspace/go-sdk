package octaspace

import (
	"context"
	"net/http"
	"os"
	"testing"
)

func TestAccounts_Balance(t *testing.T) {
	var want AccountBalance
	decodeStrict(t, "testdata/next/account_balance.json", &want)
	fixture, err := os.ReadFile("testdata/next/account_balance.json")
	if err != nil {
		t.Fatal(err)
	}
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/accounts/balance" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write(fixture)
	})
	got, err := c.Accounts.Balance(context.Background())
	if err != nil {
		t.Fatalf("Balance: %v", err)
	}
	if *got != want || got.Balance.String() != "$1.234567" || got.Currency != "USD" || got.Decimals != 6 {
		t.Errorf("Balance = %+v", got)
	}

	cNeg, _ := newTestClient(t, statusHandler(200, `{"balance":-1250000,"currency":"USD","decimals":6}`))
	neg, err := cNeg.Accounts.Balance(context.Background())
	if err != nil || neg.Balance != -1250000 || neg.Balance.String() != "-$1.250000" {
		t.Errorf("negative balance = %+v, %v", neg, err)
	}
}

func TestAccounts_Balance_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(500, `{"error":"server error"}`))
	_, err := c.Accounts.Balance(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
