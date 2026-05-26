package web_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dotcommander/gokart/web"
)

type bindTestUser struct {
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"gte=0,lte=130"`
}

type bindTestAddress struct {
	Street string `json:"street" validate:"required"`
	City   string `json:"city" validate:"required"`
	Zip    string `json:"zip" validate:"required,len=5"`
}

type bindTestNested struct {
	User    bindTestUser    `json:"user" validate:"required"`
	Address bindTestAddress `json:"address" validate:"required"`
}

func TestBindJSON(t *testing.T) {
	t.Parallel()

	t.Run("valid JSON", func(t *testing.T) {
		t.Parallel()
		body := `{"name":"Alice","email":"alice@example.com","age":30}`
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		var u bindTestUser
		if err := web.BindJSON(r, &u); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Name != "Alice" {
			t.Errorf("expected Name Alice, got %q", u.Name)
		}
		if u.Email != "alice@example.com" {
			t.Errorf("expected Email alice@example.com, got %q", u.Email)
		}
		if u.Age != 30 {
			t.Errorf("expected Age 30, got %d", u.Age)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("POST", "/", strings.NewReader("{bad"))
		var u bindTestUser
		if err := web.BindJSON(r, &u); err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})

	t.Run("empty body", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("POST", "/", strings.NewReader(""))
		var u bindTestUser
		if err := web.BindJSON(r, &u); err == nil {
			t.Fatal("expected error for empty body")
		}
	})
}

func TestBindAndValidate(t *testing.T) {
	t.Parallel()
	v := web.NewStandardValidator()

	// Decode errors: both cases must return (nil, non-nil error).
	for _, tc := range []struct {
		name string
		body string
	}{
		{"malformed JSON", "{bad"},
		{"empty body", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
			var u bindTestUser
			fields, err := web.BindAndValidate(r, v, &u)
			if err == nil {
				t.Fatal("expected decode error")
			}
			if fields != nil {
				t.Fatal("expected nil fields on decode error")
			}
		})
	}

	t.Run("valid JSON and valid struct", func(t *testing.T) {
		t.Parallel()
		body := `{"name":"Alice","email":"alice@example.com","age":25}`
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		var u bindTestUser
		fields, err := web.BindAndValidate(r, v, &u)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fields != nil {
			t.Fatalf("expected no field errors, got %v", fields)
		}
		if u.Name != "Alice" {
			t.Errorf("expected Name Alice, got %q", u.Name)
		}
	})

	t.Run("validation failure multiple fields", func(t *testing.T) {
		t.Parallel()
		body := `{"name":"","email":"not-an-email","age":-5}`
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		var u bindTestUser
		fields, err := web.BindAndValidate(r, v, &u)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fields == nil {
			t.Fatal("expected field errors")
		}
		for _, field := range []string{"name", "email", "age"} {
			if _, ok := fields[field]; !ok {
				t.Errorf("expected error for field %q", field)
			}
		}
	})

	t.Run("nested struct validation", func(t *testing.T) {
		t.Parallel()
		body := `{"user":{"name":"","email":"bad"},"address":{"street":"","city":"","zip":"1"}}`
		r := httptest.NewRequest("POST", "/", strings.NewReader(body))
		var n bindTestNested
		fields, err := web.BindAndValidate(r, v, &n)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fields == nil {
			t.Fatal("expected field errors for nested struct")
		}
		if len(fields) < 3 {
			t.Errorf("expected at least 3 field errors, got %d: %v", len(fields), fields)
		}
	})
}

// TestBindJSONWithLimit_RejectsOversized verifies the explicit-limit form
// wraps *http.MaxBytesError so callers can distinguish 413 from 400.
func TestBindJSONWithLimit_RejectsOversized(t *testing.T) {
	t.Parallel()

	// 128-byte cap, body of ~200 bytes — guaranteed overrun.
	body := `{"name":"` + strings.Repeat("a", 200) + `","email":"a@b","age":1}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var u bindTestUser
	err := web.BindJSONWithLimit(r, &u, 128)
	if err == nil {
		t.Fatal("expected error for oversized body, got nil")
	}
	var maxBytesErr *http.MaxBytesError
	if !errors.As(err, &maxBytesErr) {
		t.Errorf("error %v does not wrap *http.MaxBytesError; callers cannot return 413", err)
	}
}

// TestBindJSON_AppliesDefaultCap asserts that the package-level default
// cap is in force — a body larger than DefaultMaxRequestBodyBytes must
// produce a MaxBytesError without callers passing an explicit limit.
func TestBindJSON_AppliesDefaultCap(t *testing.T) {
	t.Parallel()

	// Construct a payload one byte beyond the default cap.
	pad := strings.Repeat("a", int(web.DefaultMaxRequestBodyBytes)+1)
	body := `{"name":"` + pad + `","email":"a@b.example","age":1}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var u bindTestUser
	err := web.BindJSON(r, &u)
	if err == nil {
		t.Fatal("expected error for body exceeding default cap, got nil")
	}
	var maxBytesErr *http.MaxBytesError
	if !errors.As(err, &maxBytesErr) {
		t.Errorf("error %v does not wrap *http.MaxBytesError", err)
	}
}

// TestBindJSONWithLimit_ZeroDisables verifies that passing limit<=0 bypasses
// the cap (the documented opt-out for callers who size bodies elsewhere).
func TestBindJSONWithLimit_ZeroDisables(t *testing.T) {
	t.Parallel()

	body := `{"name":"Alice","email":"a@b.example","age":1}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))

	var u bindTestUser
	if err := web.BindJSONWithLimit(r, &u, 0); err != nil {
		t.Fatalf("limit=0 should disable cap; got error %v", err)
	}
	if u.Name != "Alice" {
		t.Errorf("expected Name Alice, got %q", u.Name)
	}
}
