package domain_hints

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
)

func TestGoExtractor_Extract(t *testing.T) {
	source := []byte(`package order

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"myapp/repository"
	"myapp/services/inventory"
)

func TestCreateOrder(t *testing.T) {
	mockCart := Cart{Items: []Item{{ID: 1, Qty: 2}}}

	t.Run("should create order from cart", func(t *testing.T) {
		result, err := orderService.CreateFromCart(mockCart)
		assert.NoError(t, err)
		assert.Equal(t, "pending", result.Status)
	})
}
`)

	extractor := &GoExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	t.Run("imports", func(t *testing.T) {
		expectedImports := []string{
			"testing",
			"github.com/stretchr/testify/assert",
			"myapp/repository",
			"myapp/services/inventory",
		}
		if len(hints.Imports) != len(expectedImports) {
			t.Errorf("imports count: got %d, want %d", len(hints.Imports), len(expectedImports))
		}
		for i, expected := range expectedImports {
			if i >= len(hints.Imports) {
				break
			}
			if hints.Imports[i] != expected {
				t.Errorf("imports[%d]: got %q, want %q", i, hints.Imports[i], expected)
			}
		}
	})

	t.Run("variables", func(t *testing.T) {
		found := false
		for _, v := range hints.Variables {
			if v == "mockCart" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected mockCart in variables, got %v", hints.Variables)
		}
	})
}

func TestGoExtractor_Extract_EmptyFile(t *testing.T) {
	source := []byte(`package empty`)

	extractor := &GoExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints != nil {
		t.Errorf("expected nil for empty file, got %+v", hints)
	}
}

func TestGoExtractor_Extract_Calls(t *testing.T) {
	source := []byte(`package test

import "testing"

func TestSomething(t *testing.T) {
	authService.ValidateToken("token")
	userRepo.FindByID(1)
	result, err := orderService.Create(order)
	doSomething()
}
`)

	extractor := &GoExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedCalls := map[string]bool{
		"authService.ValidateToken": true,
		"userRepo.FindByID":         true,
		"orderService.Create":       true,
		"doSomething":               true,
	}

	for _, call := range hints.Calls {
		delete(expectedCalls, call)
	}

	if len(expectedCalls) > 0 {
		t.Errorf("missing calls: %v, got: %v", expectedCalls, hints.Calls)
	}
}

func TestGoExtractor_Extract_VariableFiltering(t *testing.T) {
	source := []byte(`package test

import "testing"

func TestFiltering(t *testing.T) {
	mockUser := User{}
	fakeClient := &Client{}
	stubRepo := newStubRepo()
	testData := getData()
	expectedResult := "ok"
	wantValue := 42
	gotResult := doSomething()
	fixtureOrder := Order{}
	regularVar := "ignored"
	count := 10
}
`)

	extractor := &GoExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	shouldMatch := []string{"mockUser", "fakeClient", "stubRepo", "testData", "expectedResult", "wantValue", "gotResult", "fixtureOrder"}
	shouldNotMatch := []string{"regularVar", "count"}

	varSet := make(map[string]bool)
	for _, v := range hints.Variables {
		varSet[v] = true
	}

	for _, v := range shouldMatch {
		if !varSet[v] {
			t.Errorf("expected %q to be included in variables", v)
		}
	}

	for _, v := range shouldNotMatch {
		if varSet[v] {
			t.Errorf("expected %q to be excluded from variables", v)
		}
	}
}

func TestGetExtractor(t *testing.T) {
	tests := []struct {
		lang    domain.Language
		wantNil bool
	}{
		{domain.LanguageGo, false},
		{domain.LanguageJavaScript, true},
		{domain.LanguagePython, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			ext := GetExtractor(tt.lang)
			if tt.wantNil && ext != nil {
				t.Errorf("expected nil extractor for %s", tt.lang)
			}
			if !tt.wantNil && ext == nil {
				t.Errorf("expected extractor for %s, got nil", tt.lang)
			}
		})
	}
}
