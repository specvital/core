package domain_hints

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
)

func TestRubyExtractor_Extract_RequireStatements(t *testing.T) {
	source := []byte(`
require 'tmpdir'
require "json"
require 'rspec/core'
require_relative './helpers'
require_relative '../spec_helper'
`)

	extractor := &RubyExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"tmpdir":           true,
		"json":             true,
		"rspec/core":       true,
		"./helpers":        true,
		"../spec_helper":   true,
	}

	importSet := make(map[string]bool)
	for _, imp := range hints.Imports {
		importSet[imp] = true
	}

	for imp := range expectedImports {
		if !importSet[imp] {
			t.Errorf("expected import %q to be included, got %v", imp, hints.Imports)
		}
	}
}

func TestRubyExtractor_Extract_MethodCalls(t *testing.T) {
	source := []byte(`
require 'rspec'

RSpec.describe UserService do
  it 'creates a user' do
    user_service.create(params)
    payment_gateway.process_payment(amount)
    NotificationService.send_email(user)
  end
end
`)

	extractor := &RubyExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedCalls := map[string]bool{
		"user_service.create":           true,
		"payment_gateway.process_payment": true,
		"NotificationService.send_email": true,
	}

	callSet := make(map[string]bool)
	for _, call := range hints.Calls {
		callSet[call] = true
	}

	for call := range expectedCalls {
		if !callSet[call] {
			t.Errorf("expected call %q to be included, got %v", call, hints.Calls)
		}
	}
}

func TestRubyExtractor_Extract_EmptyFile(t *testing.T) {
	source := []byte(`# empty file`)

	extractor := &RubyExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints != nil {
		t.Errorf("expected nil for empty file, got %+v", hints)
	}
}

func TestRubyExtractor_Extract_TestFrameworkCalls(t *testing.T) {
	source := []byte(`
require 'rspec'

RSpec.describe 'Payment' do
  let(:user) { create(:user) }
  before { setup_mocks }

  it 'processes payment' do
    expect(result).to be_valid
    payment_service.charge(user)
  end
end
`)

	extractor := &RubyExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	callSet := make(map[string]bool)
	for _, call := range hints.Calls {
		callSet[call] = true
	}

	// RSpec framework calls should be excluded
	excludedCalls := []string{"RSpec.describe", "expect.to", "create"}
	for _, call := range excludedCalls {
		if callSet[call] {
			t.Errorf("expected test framework call %q to be excluded, got %v", call, hints.Calls)
		}
	}

	// Domain calls should be included
	if !callSet["payment_service.charge"] {
		t.Errorf("expected payment_service.charge call, got %v", hints.Calls)
	}
}

func TestRubyExtractor_Extract_RSpecFile(t *testing.T) {
	source := []byte(`
require 'rails_helper'
require_relative './support/auth_helpers'

RSpec.describe PaymentController, type: :controller do
  include AuthHelpers

  let(:user) { create(:user) }
  let(:order) { Order.new(amount: 100) }

  describe '#create' do
    before do
      stripe_client.configure(api_key)
    end

    it 'creates a payment intent' do
      payment_gateway.create_intent(order)
      notification_service.send_confirmation(user)
      expect(response).to have_http_status(:ok)
    end
  end
end
`)

	extractor := &RubyExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Verify imports
	importSet := make(map[string]bool)
	for _, imp := range hints.Imports {
		importSet[imp] = true
	}

	expectedImports := []string{"rails_helper", "./support/auth_helpers"}
	for _, imp := range expectedImports {
		if !importSet[imp] {
			t.Errorf("expected import %q, got %v", imp, hints.Imports)
		}
	}

	// Verify calls
	callSet := make(map[string]bool)
	for _, call := range hints.Calls {
		callSet[call] = true
	}

	expectedCalls := []string{"stripe_client.configure", "payment_gateway.create_intent", "notification_service.send_confirmation", "Order.new"}
	for _, call := range expectedCalls {
		if !callSet[call] {
			t.Errorf("expected call %q, got %v", call, hints.Calls)
		}
	}
}

func TestRubyExtractor_Extract_ChainedCalls(t *testing.T) {
	source := []byte(`
require 'rspec'

describe 'Chained calls' do
  it 'handles chains' do
    client.api.users.list
    response.data.items.first
  end
end
`)

	extractor := &RubyExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	callSet := make(map[string]bool)
	for _, call := range hints.Calls {
		callSet[call] = true
	}

	// Should be normalized to 2 segments
	expectedCalls := []string{"client.api", "response.data"}
	for _, call := range expectedCalls {
		if !callSet[call] {
			t.Errorf("expected %q call (2-segment normalized), got %v", call, hints.Calls)
		}
	}
}

func TestRubyExtractor_Extract_DifferentQuoteStyles(t *testing.T) {
	source := []byte(`
require 'single_quoted'
require "double_quoted"
`)

	extractor := &RubyExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"single_quoted": true,
		"double_quoted": true,
	}

	importSet := make(map[string]bool)
	for _, imp := range hints.Imports {
		importSet[imp] = true
	}

	for imp := range expectedImports {
		if !importSet[imp] {
			t.Errorf("expected import %q to be included, got %v", imp, hints.Imports)
		}
	}
}

func TestRubyExtractor_Extract_Deduplication(t *testing.T) {
	source := []byte(`
require 'json'
require 'json'
require "json"

user_service.create(1)
user_service.create(2)
`)

	extractor := &RubyExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Count occurrences of 'json' in imports
	jsonCount := 0
	for _, imp := range hints.Imports {
		if imp == "json" {
			jsonCount++
		}
	}
	if jsonCount != 1 {
		t.Errorf("expected 'json' to appear once, got %d times in %v", jsonCount, hints.Imports)
	}

	// Count occurrences of user_service.create in calls
	callCount := 0
	for _, call := range hints.Calls {
		if call == "user_service.create" {
			callCount++
		}
	}
	if callCount != 1 {
		t.Errorf("expected 'user_service.create' to appear once, got %d times in %v", callCount, hints.Calls)
	}
}

func TestGetExtractor_Ruby(t *testing.T) {
	ext := GetExtractor(domain.LanguageRuby)
	if ext == nil {
		t.Error("expected extractor for Ruby, got nil")
	}

	_, ok := ext.(*RubyExtractor)
	if !ok {
		t.Errorf("expected RubyExtractor, got %T", ext)
	}
}

func TestTrimRubyQuotes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"single quotes", "'hello'", "hello"},
		{"double quotes", `"hello"`, "hello"},
		{"no quotes", "hello", "hello"},
		{"empty", "", ""},
		{"single char", "a", "a"},
		{"only quotes", `""`, ""},
		{"path with slashes", "'path/to/file'", "path/to/file"},
		{"percent q parens", "%q(hello)", "hello"},
		{"percent Q parens", "%Q(hello)", "hello"},
		{"percent q brackets", "%q[hello]", "hello"},
		{"percent q braces", "%q{hello}", "hello"},
		{"percent q angle", "%q<hello>", "hello"},
		{"percent q pipe", "%q|hello|", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimRubyQuotes(tt.input)
			if got != tt.want {
				t.Errorf("trimRubyQuotes(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
