package octaspace

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"
)

// --- BigInt ---

func TestBigIntUnmarshalJSON(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"123456789", "123456789", false},
		{"0", "0", false},
		{"999999999999999999999", "999999999999999999999", false},
		{"abc", "", true},
		{`"quoted"`, "", true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			var b BigInt
			err := json.Unmarshal([]byte(tc.input), &b)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if b.String() != tc.want {
				t.Errorf("got %q, want %q", b.String(), tc.want)
			}
		})
	}
}

func TestBigIntMarshalJSON(t *testing.T) {
	b := BigInt{Int: big.NewInt(42)}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(data) != "42" {
		t.Fatalf("got %q, want \"42\"", data)
	}
}

func TestBigIntMarshalJSON_Nil(t *testing.T) {
	var zero BigInt
	data, err := json.Marshal(zero)
	if err != nil {
		t.Fatalf("MarshalJSON nil: %v", err)
	}
	if string(data) != "0" {
		t.Fatalf("got %q, want \"0\"", data)
	}
}

func TestBigIntRoundTrip(t *testing.T) {
	original := new(big.Int).Mul(big.NewInt(1e18), big.NewInt(999))
	b := BigInt{Int: original}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var b2 BigInt
	if err := json.Unmarshal(data, &b2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if b.Cmp(b2.Int) != 0 {
		t.Fatalf("round-trip mismatch: got %s, want %s", b2.Int, b.Int)
	}
}

// --- AccountsResource ---

func TestAccounts_Profile(t *testing.T) {
	want := Account{
		AccountUUID: "uuid-001",
		Email:       "user@example.com",
		UID:         42,
		Balance:     BigInt{Int: big.NewInt(1000)},
	}
	c, _ := newTestClient(t, jsonHandler(want))
	got, err := c.Accounts.Profile(context.Background())
	if err != nil {
		t.Fatalf("Profile: %v", err)
	}
	if got.AccountUUID != want.AccountUUID {
		t.Errorf("AccountUUID = %q, want %q", got.AccountUUID, want.AccountUUID)
	}
	if got.Email != want.Email {
		t.Errorf("Email = %q, want %q", got.Email, want.Email)
	}
}

func TestAccounts_Profile_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(401, `{"error":"unauthorized"}`))
	_, err := c.Accounts.Profile(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsAuthError(err) {
		t.Errorf("expected AuthenticationError, got %T", err)
	}
}

func TestAccounts_Balance(t *testing.T) {
	want := big.NewInt(500000)
	payload := map[string]json.RawMessage{
		"balance": json.RawMessage(want.String()),
	}
	c, _ := newTestClient(t, jsonHandler(payload))
	got, err := c.Accounts.Balance(context.Background())
	if err != nil {
		t.Fatalf("Balance: %v", err)
	}
	if got.Cmp(want) != 0 {
		t.Errorf("Balance = %s, want %s", got, want)
	}
}

func TestAccounts_Balance_Error(t *testing.T) {
	c, _ := newTestClient(t, statusHandler(500, `{"error":"server error"}`))
	_, err := c.Accounts.Balance(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAccounts_GenerateWallet(t *testing.T) {
	want := Account{Wallet: "0xDEADBEEF", AccountUUID: "uuid-gen"}
	c, _ := newTestClient(t, jsonHandler(want))
	got, err := c.Accounts.GenerateWallet(context.Background())
	if err != nil {
		t.Fatalf("GenerateWallet: %v", err)
	}
	if got.Wallet != want.Wallet {
		t.Errorf("Wallet = %q, want %q", got.Wallet, want.Wallet)
	}
}
