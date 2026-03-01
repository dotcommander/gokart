package web_test

import (
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

	t.Run("malformed JSON", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("POST", "/", strings.NewReader("{bad"))
		var u bindTestUser
		fields, err := web.BindAndValidate(r, v, &u)
		if err == nil {
			t.Fatal("expected decode error")
		}
		if fields != nil {
			t.Fatal("expected nil fields on decode error")
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
		if _, ok := fields["name"]; !ok {
			t.Error("expected error for field 'name'")
		}
		if _, ok := fields["email"]; !ok {
			t.Error("expected error for field 'email'")
		}
		if _, ok := fields["age"]; !ok {
			t.Error("expected error for field 'age'")
		}
	})

	t.Run("empty body", func(t *testing.T) {
		t.Parallel()
		r := httptest.NewRequest("POST", "/", strings.NewReader(""))
		var u bindTestUser
		_, err := web.BindAndValidate(r, v, &u)
		if err == nil {
			t.Fatal("expected decode error for empty body")
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
