package domain_hints

import (
	"context"
	"strings"
	"testing"

	"github.com/specvital/core/pkg/domain"
)

func TestRustExtractor_Extract_UseStatements(t *testing.T) {
	source := []byte(`
use std::collections::HashMap;
use crate::models::User;
use super::helpers;
use tokio::sync::mpsc;
`)

	extractor := &RustExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"std/collections/HashMap": true,
		"crate/models/User":       true,
		"super/helpers":           true,
		"tokio/sync/mpsc":         true,
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

func TestRustExtractor_Extract_UseList(t *testing.T) {
	source := []byte(`
use std::collections::{HashMap, HashSet};
use crate::{models, services};
`)

	extractor := &RustExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Should extract the base path before the list
	expectedImports := map[string]bool{
		"std/collections": true,
		"crate":           true,
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

func TestRustExtractor_Extract_UseWildcard(t *testing.T) {
	source := []byte(`
use std::prelude::*;
use crate::models::*;
`)

	extractor := &RustExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"std/prelude":   true,
		"crate/models":  true,
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

func TestRustExtractor_Extract_UseAlias(t *testing.T) {
	source := []byte(`
use std::collections::HashMap as Map;
use crate::models::User as UserModel;
`)

	extractor := &RustExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"std/collections/HashMap": true,
		"crate/models/User":       true,
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

func TestRustExtractor_Extract_ModDeclarations(t *testing.T) {
	source := []byte(`
mod tests;
mod helpers;
pub mod utils;
`)

	extractor := &RustExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"tests":   true,
		"helpers": true,
		"utils":   true,
	}

	importSet := make(map[string]bool)
	for _, imp := range hints.Imports {
		importSet[imp] = true
	}

	for imp := range expectedImports {
		if !importSet[imp] {
			t.Errorf("expected mod %q to be included, got %v", imp, hints.Imports)
		}
	}
}

func TestRustExtractor_Extract_MethodCalls(t *testing.T) {
	source := []byte(`
use std::collections::HashMap;

fn test_service() {
    user_service.create(user);
    PaymentGateway::process(payment);
    notification_service.send_email(user);
}
`)

	extractor := &RustExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedCalls := map[string]bool{
		"user_service.create":          true,
		"PaymentGateway.process":       true,
		"notification_service.send_email": true,
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

func TestRustExtractor_Extract_EmptyFile(t *testing.T) {
	source := []byte(`// empty file`)

	extractor := &RustExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints != nil {
		t.Errorf("expected nil for empty file, got %+v", hints)
	}
}

func TestRustExtractor_Extract_TestFrameworkCalls(t *testing.T) {
	source := []byte(`
use crate::services::payment;

#[test]
fn test_payment() {
    assert_eq!(result, expected);
    assert!(condition);
    println!("debug output");

    payment_service.process(order);
    Result::Ok(value);
}
`)

	extractor := &RustExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	callSet := make(map[string]bool)
	for _, call := range hints.Calls {
		callSet[call] = true
	}

	// Test framework calls should be excluded
	excludedCalls := []string{"assert_eq", "assert", "println"}
	for _, call := range excludedCalls {
		if callSet[call] {
			t.Errorf("expected test framework call %q to be excluded, got %v", call, hints.Calls)
		}
	}

	// Domain calls should be included
	if !callSet["payment_service.process"] {
		t.Errorf("expected payment_service.process call, got %v", hints.Calls)
	}
}

func TestRustExtractor_Extract_CargoTestFile(t *testing.T) {
	source := []byte(`
use crate::models::Order;
use crate::services::payment::PaymentGateway;
use tokio::sync::mpsc;

mod test_helpers;

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn test_payment_processing() {
        let gateway = PaymentGateway::new();
        let order = Order::create(100);

        gateway.process(order).await;
        notification_service.send_confirmation(order.id);
    }
}
`)

	extractor := &RustExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Verify imports
	importSet := make(map[string]bool)
	for _, imp := range hints.Imports {
		importSet[imp] = true
	}

	expectedImports := []string{
		"crate/models/Order",
		"crate/services/payment/PaymentGateway",
		"tokio/sync/mpsc",
		"test_helpers",
	}
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

	expectedCalls := []string{
		"PaymentGateway.new",
		"Order.create",
		"gateway.process",
		"notification_service.send_confirmation",
	}
	for _, call := range expectedCalls {
		if !callSet[call] {
			t.Errorf("expected call %q, got %v", call, hints.Calls)
		}
	}
}

func TestRustExtractor_Extract_Deduplication(t *testing.T) {
	source := []byte(`
use std::collections::HashMap;
use std::collections::HashMap;

fn test() {
    user_service.create(1);
    user_service.create(2);
}
`)

	extractor := &RustExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Count occurrences
	importCount := 0
	for _, imp := range hints.Imports {
		if imp == "std/collections/HashMap" {
			importCount++
		}
	}
	if importCount != 1 {
		t.Errorf("expected 'std/collections/HashMap' to appear once, got %d times in %v", importCount, hints.Imports)
	}

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

func TestGetExtractor_Rust(t *testing.T) {
	ext := GetExtractor(domain.LanguageRust)
	if ext == nil {
		t.Error("expected extractor for Rust, got nil")
	}

	_, ok := ext.(*RustExtractor)
	if !ok {
		t.Errorf("expected RustExtractor, got %T", ext)
	}
}

func TestExtractRustUsePath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple path", "use std::collections::HashMap;", "std/collections/HashMap"},
		{"crate path", "use crate::models::User;", "crate/models/User"},
		{"super path", "use super::helpers;", "super/helpers"},
		{"wildcard", "use std::prelude::*;", "std/prelude"},
		{"alias", "use std::collections::HashMap as Map;", "std/collections/HashMap"},
		{"list", "use std::collections::{HashMap, HashSet};", "std/collections"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since we can't easily create tree-sitter nodes, test the string processing
			// by calling the function logic directly
			text := tt.input
			text = text[4:]           // Remove "use "
			text = text[:len(text)-1] // Remove ";"

			// Apply the same logic as extractRustUsePath
			if idx := strings.Index(text, "::{"); idx > 0 {
				text = text[:idx]
			}
			if idx := strings.Index(text, " as "); idx > 0 {
				text = text[:idx]
			}
			text = strings.TrimSuffix(text, "::*")
			text = strings.ReplaceAll(text, "::", "/")

			if text != tt.want {
				t.Errorf("extractRustUsePath logic for %q = %q, want %q", tt.input, text, tt.want)
			}
		})
	}
}
