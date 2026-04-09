package sdk

import (
	"net/http"
	"testing"
)

func TestListApps(t *testing.T) {
	want := []App{
		{UUID: "app-1", Name: "Jupyter", Image: "jupyter:latest", Category: "ml"},
		{UUID: "app-2", Name: "VS Code", Category: "dev"},
	}
	c, cleanup := withServer(t, jsonHandler(http.StatusOK, want))
	defer cleanup()

	got, err := c.ListApps()
	if err != nil {
		t.Fatalf("ListApps: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i, a := range got {
		if a.UUID != want[i].UUID {
			t.Errorf("[%d] UUID = %q, want %q", i, a.UUID, want[i].UUID)
		}
		if a.Name != want[i].Name {
			t.Errorf("[%d] Name = %q, want %q", i, a.Name, want[i].Name)
		}
		if a.Category != want[i].Category {
			t.Errorf("[%d] Category = %q, want %q", i, a.Category, want[i].Category)
		}
	}
}

func TestListAppsEmpty(t *testing.T) {
	c, cleanup := withServer(t, jsonHandler(http.StatusOK, []App{}))
	defer cleanup()

	got, err := c.ListApps()
	if err != nil {
		t.Fatalf("ListApps: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(got))
	}
}

func TestListAppsError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusUnauthorized, `{}`))
	defer cleanup()

	_, err := c.ListApps()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestListAppsRaw(t *testing.T) {
	body := `[{"uuid":"app-1","name":"Jupyter"}]`
	c, cleanup := withServer(t, rawHandler(http.StatusOK, body))
	defer cleanup()

	got, err := c.ListAppsRaw()
	if err != nil {
		t.Fatalf("ListAppsRaw: %v", err)
	}
	if string(got) != body {
		t.Fatalf("body = %q, want %q", got, body)
	}
}

func TestListAppsRawError(t *testing.T) {
	c, cleanup := withServer(t, rawHandler(http.StatusInternalServerError, `err`))
	defer cleanup()

	_, err := c.ListAppsRaw()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
