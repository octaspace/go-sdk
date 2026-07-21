package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	octaspace "github.com/octaspace/go-sdk"
)

func TestRunCheck_SkipsWithoutAuth(t *testing.T) {
	chk := check{
		id:           "auth-only",
		label:        "auth-only",
		method:       "GET",
		path:         "/auth-only",
		requiresAuth: true,
		fn: func(context.Context, *octaspace.Client) (any, error) {
			t.Fatal("check function should not run without auth")
			return nil, nil
		},
	}

	r := runCheck(context.Background(), nil, chk, false)
	if r.Status != "skipped" {
		t.Fatalf("Status = %q, want skipped", r.Status)
	}
	if r.Summary != "no auth" {
		t.Fatalf("Summary = %q, want no auth", r.Summary)
	}
}

func TestSummarizeDoesNotExposeScalarOrAccountValues(t *testing.T) {
	if got := summarize("104064765413502726951"); got != "string" {
		t.Errorf("summarize(string) = %q, want string", got)
	}
	got := summarize(map[string]any{"email": "person@example.test", "account_uuid": "secret-uuid"})
	if strings.Contains(got, "person@example.test") || strings.Contains(got, "secret-uuid") {
		t.Errorf("summary exposes account data: %q", got)
	}
}

func TestRunCheck_SkipsSentinelErrors(t *testing.T) {
	chk := check{
		id:     "recent-logs",
		label:  "recent-logs",
		method: "GET",
		path:   "/services/:uuid/logs?recent=true",
		fn: func(context.Context, *octaspace.Client) (any, error) {
			return nil, skippedError("no recent sessions")
		},
	}

	r := runCheck(context.Background(), nil, chk, true)
	if r.Status != "skipped" {
		t.Fatalf("Status = %q, want skipped", r.Status)
	}
	if r.Summary != "no recent sessions" {
		t.Fatalf("Summary = %q, want no recent sessions", r.Summary)
	}
}

func TestRunCheck_FailuresAreNotSkippedOrPassed(t *testing.T) {
	chk := check{
		id:     "broken",
		label:  "broken",
		method: "GET",
		path:   "/broken",
		fn: func(context.Context, *octaspace.Client) (any, error) {
			return nil, errors.New("boom")
		},
	}

	r := runCheck(context.Background(), nil, chk, true)
	if r.Status != "failed" {
		t.Fatalf("Status = %q, want failed", r.Status)
	}
	if r.Error != "boom" {
		t.Fatalf("Error = %q, want boom", r.Error)
	}
}

func TestCountResults(t *testing.T) {
	got := countResults([]result{
		{Status: "passed"},
		{Status: "failed"},
		{Status: "skipped"},
		{Status: "passed"},
	})
	want := summary{Total: 4, Passed: 2, Failed: 1, Skipped: 1}
	if got != want {
		t.Fatalf("countResults = %+v, want %+v", got, want)
	}
}

func TestSmokePublicChecksDoNotRequireAuth(t *testing.T) {
	publicChecks := map[string]bool{
		"network.info":      true,
		"apps.list":         true,
		"services.mr.list":  true,
		"services.vpn.list": true,
	}

	for _, chk := range checks {
		if !publicChecks[chk.id] {
			continue
		}
		if chk.requiresAuth {
			t.Fatalf("%s is public but requiresAuth=true", chk.id)
		}
		delete(publicChecks, chk.id)
	}
	for id := range publicChecks {
		t.Fatalf("public check %s not found", id)
	}
}
