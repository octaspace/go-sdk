package sdk

import (
	"encoding/json"
	"math/big"
	"net/http"
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
		// JSON numbers without quotes are acceptable
		{"999999999999999999999", "999999999999999999999", false},
		{"abc", "", true},
		{`"quoted"`, "", true},
	}

	for _, tc := range tests {
		var b BigInt
		err := json.Unmarshal([]byte(tc.input), &b)
		if tc.wantErr {
			if err == nil {
				t.Errorf("Unmarshal(%q): expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("Unmarshal(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if b.Int.String() != tc.want {
			t.Errorf("Unmarshal(%q) = %q, want %q", tc.input, b.Int.String(), tc.want)
		}
	}
}

func TestBigIntMarshalJSON(t *testing.T) {
	b := BigInt{Int: big.NewInt(42)}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if string(data) != "42" {
		t.Fatalf("MarshalJSON = %q, want %q", data, "42")
	}

	// nil Int should marshal as "0"
	var zero BigInt
	data, err = json.Marshal(zero)
	if err != nil {
		t.Fatalf("MarshalJSON nil: %v", err)
	}
	if string(data) != "0" {
		t.Fatalf("MarshalJSON nil = %q, want \"0\"", data)
	}
}

func TestBigIntRoundTrip(t *testing.T) {
	original := big.NewInt(0).Mul(big.NewInt(1e18), big.NewInt(999))
	b := BigInt{Int: original}

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var b2 BigInt
	if err := json.Unmarshal(data, &b2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if b.Int.Cmp(b2.Int) != 0 {
		t.Fatalf("round-trip mismatch: got %s, want %s", b2.Int, b.Int)
	}
}

// --- GetAccount ---

func TestGetAccount(t *testing.T) {
	payload := Account{
		AccountUUID: "uuid-001",
		Email:       "user@example.com",
		UID:         42,
		Balance:     BigInt{Int: big.NewInt(1000)},
	}
	c, cleanup := withServer(t, jsonHandler(http.StatusOK, payload))
	defer cleanup()

	got, err := c.GetAccount()
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if got.AccountUUID != payload.AccountUUID {
		t.Errorf("AccountUUID = %q, want %q", got.AccountUUID, payload.AccountUUID)
	}
	if got.Email != payload.Email {
		t.Errorf("Email = %q, want %q", got.Email, payload.Email)
	}
	if got.UID != payload.UID {
		t.Errorf("UID = %d, want %d", got.UID, payload.UID)
	}
}

func TestGetAccountError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusUnauthorized, `{"error":"unauthorized"}`))
	defer cleanup()

	_, err := c.GetAccount()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- GetBalance ---

func TestGetBalance(t *testing.T) {
	want := big.NewInt(500000)
	// BigInt.UnmarshalJSON expects a bare decimal number, not a quoted string.
	payload := map[string]json.RawMessage{"balance": json.RawMessage(want.String())}
	c, cleanup := withServer(t, jsonHandler(http.StatusOK, payload))
	defer cleanup()

	got, err := c.GetBalance()
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if got.Cmp(want) != 0 {
		t.Errorf("GetBalance = %s, want %s", got, want)
	}
}

func TestGetBalanceError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusInternalServerError, `err`))
	defer cleanup()

	_, err := c.GetBalance()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- ValidateToken ---

func TestValidateTokenOK(t *testing.T) {
	c, cleanup := withServer(t, jsonHandler(http.StatusOK, Account{}))
	defer cleanup()

	if err := c.ValidateToken(); err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
}

func TestValidateTokenError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusUnauthorized, `{}`))
	defer cleanup()

	if err := c.ValidateToken(); err == nil {
		t.Fatal("expected error, got nil")
	}
}
