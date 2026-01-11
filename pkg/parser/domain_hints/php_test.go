package domain_hints

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
)

func TestPHPExtractor_Extract_UseStatements(t *testing.T) {
	source := []byte(`<?php
namespace App\Tests;

use PHPUnit\Framework\TestCase;
use App\Services\PaymentService;
use Stripe\PaymentIntent;
use App\Models\User as UserModel;
`)

	extractor := &PHPExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"PHPUnit\\Framework\\TestCase": true,
		"App\\Services\\PaymentService": true,
		"Stripe\\PaymentIntent":         true,
		"App\\Models\\User":             true,
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

func TestPHPExtractor_Extract_MethodCalls(t *testing.T) {
	source := []byte(`<?php
namespace App\Tests;

use PHPUnit\Framework\TestCase;

class PaymentTest extends TestCase
{
    public function testCreatePayment(): void
    {
        $paymentService->createIntent($amount);
        $stripeClient->processPayment($order);
        PaymentGateway::configure($config);
    }
}
`)

	extractor := &PHPExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedCalls := map[string]bool{
		"paymentService.createIntent": true,
		"stripeClient.processPayment": true,
		"PaymentGateway.configure":    true,
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

func TestPHPExtractor_Extract_EmptyFile(t *testing.T) {
	source := []byte(`<?php
// empty file
`)

	extractor := &PHPExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints != nil {
		t.Errorf("expected nil for empty file, got %+v", hints)
	}
}

func TestPHPExtractor_Extract_TestFrameworkCalls(t *testing.T) {
	source := []byte(`<?php
use PHPUnit\Framework\TestCase;

class UserTest extends TestCase
{
    public function testUser(): void
    {
        $this->assertEquals($expected, $actual);
        $this->assertTrue($result);
        $userService->findById($id);
    }
}
`)

	extractor := &PHPExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	callSet := make(map[string]bool)
	for _, call := range hints.Calls {
		callSet[call] = true
	}

	// Test framework calls should be excluded
	excludedCalls := []string{"this.assertEquals", "this.assertTrue"}
	for _, call := range excludedCalls {
		if callSet[call] {
			t.Errorf("expected test framework call %q to be excluded, got %v", call, hints.Calls)
		}
	}

	// Domain calls should be included
	if !callSet["userService.findById"] {
		t.Errorf("expected userService.findById call, got %v", hints.Calls)
	}
}

func TestPHPExtractor_Extract_PHPUnitFile(t *testing.T) {
	source := []byte(`<?php
namespace App\Tests\Payment;

use PHPUnit\Framework\TestCase;
use App\Services\PaymentService;
use Stripe\StripeClient;
use App\Models\Order;

class PaymentServiceTest extends TestCase
{
    public function testCreatePayment(): void
    {
        StripeClient::setApiKey($key);
        NotificationService::sendConfirmation($user);
        $orderService->process($data);
    }
}
`)

	extractor := &PHPExtractor{}
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
		"PHPUnit\\Framework\\TestCase",
		"App\\Services\\PaymentService",
		"Stripe\\StripeClient",
		"App\\Models\\Order",
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

	expectedCalls := []string{"StripeClient.setApiKey", "NotificationService.sendConfirmation", "orderService.process"}
	for _, call := range expectedCalls {
		if !callSet[call] {
			t.Errorf("expected call %q, got %v", call, hints.Calls)
		}
	}
}

func TestPHPExtractor_Extract_IncludeRequire(t *testing.T) {
	source := []byte(`<?php
include 'helpers.php';
include_once 'utils.php';
require 'config.php';
require_once 'bootstrap.php';
`)

	extractor := &PHPExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"helpers.php":   true,
		"utils.php":     true,
		"config.php":    true,
		"bootstrap.php": true,
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

func TestPHPExtractor_Extract_StaticMethodCalls(t *testing.T) {
	source := []byte(`<?php
use App\Models\User;

class Test
{
    public function test(): void
    {
        User::find(1);
        Cache::remember('key', function() {});
        DB::table('users')->get();
    }
}
`)

	extractor := &PHPExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedCalls := map[string]bool{
		"User.find":      true,
		"Cache.remember": true,
		"DB.table":       true,
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

func TestPHPExtractor_Extract_Deduplication(t *testing.T) {
	source := []byte(`<?php
use App\Models\User;

class Test
{
    public function test(): void
    {
        User::find(1);
        User::find(2);
        $service->process();
        $service->process();
    }
}
`)

	extractor := &PHPExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Count occurrences of 'User.find' in calls
	callCount := 0
	for _, call := range hints.Calls {
		if call == "User.find" {
			callCount++
		}
	}
	if callCount != 1 {
		t.Errorf("expected 'User.find' to appear once, got %d times in %v", callCount, hints.Calls)
	}

	// Count occurrences of service.process in calls
	processCount := 0
	for _, call := range hints.Calls {
		if call == "service.process" {
			processCount++
		}
	}
	if processCount != 1 {
		t.Errorf("expected 'service.process' to appear once, got %d times in %v", processCount, hints.Calls)
	}
}

func TestGetExtractor_PHP(t *testing.T) {
	ext := GetExtractor(domain.LanguagePHP)
	if ext == nil {
		t.Error("expected extractor for PHP, got nil")
	}

	_, ok := ext.(*PHPExtractor)
	if !ok {
		t.Errorf("expected PHPExtractor, got %T", ext)
	}
}

func TestTrimPHPQuotes(t *testing.T) {
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
		{"path with slashes", "'path/to/file.php'", "path/to/file.php"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimPHPQuotes(tt.input)
			if got != tt.want {
				t.Errorf("trimPHPQuotes(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
