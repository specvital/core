package domain_hints

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
)

func TestCppExtractor_Extract_IncludeStatements(t *testing.T) {
	source := []byte(`
#include <iostream>
#include <vector>
#include "myheader.h"
#include <gtest/gtest.h>
`)

	extractor := &CppExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"iostream":        true,
		"vector":          true,
		"myheader.h":      true,
		"gtest/gtest.h":   true,
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

func TestCppExtractor_Extract_SystemHeaders(t *testing.T) {
	source := []byte(`
#include <string>
#include <memory>
#include <algorithm>
#include <functional>
`)

	extractor := &CppExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"string":     true,
		"memory":     true,
		"algorithm":  true,
		"functional": true,
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

func TestCppExtractor_Extract_LocalHeaders(t *testing.T) {
	source := []byte(`
#include "services/payment.h"
#include "models/user.h"
#include "../common/utils.h"
`)

	extractor := &CppExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"services/payment.h": true,
		"models/user.h":      true,
		"../common/utils.h":  true,
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

func TestCppExtractor_Extract_MethodCalls(t *testing.T) {
	source := []byte(`
#include <iostream>

void testFunction() {
    userService.create(user);
    PaymentGateway::process(payment);
    notificationService->sendEmail(user);
}
`)

	extractor := &CppExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedCalls := map[string]bool{
		"userService.create":          true,
		"PaymentGateway.process":      true,
		"notificationService.sendEmail": true,
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

func TestCppExtractor_Extract_EmptyFile(t *testing.T) {
	source := []byte(`// empty file`)

	extractor := &CppExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints != nil {
		t.Errorf("expected nil for empty file, got %+v", hints)
	}
}

func TestCppExtractor_Extract_TestFrameworkCalls(t *testing.T) {
	source := []byte(`
#include <gtest/gtest.h>

TEST(PaymentTest, ProcessPayment) {
    EXPECT_EQ(result, expected);
    ASSERT_TRUE(condition);

    paymentService.process(order);
}
`)

	extractor := &CppExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	callSet := make(map[string]bool)
	for _, call := range hints.Calls {
		callSet[call] = true
	}

	// Test framework calls should be excluded
	excludedCalls := []string{"EXPECT_EQ", "ASSERT_TRUE", "TEST"}
	for _, call := range excludedCalls {
		if callSet[call] {
			t.Errorf("expected test framework call %q to be excluded, got %v", call, hints.Calls)
		}
	}

	// Domain calls should be included
	if !callSet["paymentService.process"] {
		t.Errorf("expected paymentService.process call, got %v", hints.Calls)
	}
}

func TestCppExtractor_Extract_GTestFile(t *testing.T) {
	source := []byte(`
#include <gtest/gtest.h>
#include "services/payment.h"
#include "models/order.h"

class PaymentTest : public ::testing::Test {
protected:
    void SetUp() override {
        gateway = std::make_unique<PaymentGateway>();
    }

    std::unique_ptr<PaymentGateway> gateway;
};

TEST_F(PaymentTest, ProcessPayment) {
    Order order(100);

    gateway->process(order);
    notificationService->sendConfirmation(order.id);

    EXPECT_TRUE(gateway->isComplete());
}
`)

	extractor := &CppExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Verify imports
	importSet := make(map[string]bool)
	for _, imp := range hints.Imports {
		importSet[imp] = true
	}

	expectedImports := []string{"gtest/gtest.h", "services/payment.h", "models/order.h"}
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

	expectedCalls := []string{"gateway.process", "notificationService.sendConfirmation"}
	for _, call := range expectedCalls {
		if !callSet[call] {
			t.Errorf("expected call %q, got %v", call, hints.Calls)
		}
	}
}

func TestCppExtractor_Extract_Catch2File(t *testing.T) {
	source := []byte(`
#include <catch2/catch_test_macros.hpp>
#include "services/user.h"

TEST_CASE("User creation", "[user]") {
    SECTION("valid user") {
        userService.create(validData);
        repository.save(user);

        REQUIRE(user.isValid());
    }
}
`)

	extractor := &CppExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Verify imports
	importSet := make(map[string]bool)
	for _, imp := range hints.Imports {
		importSet[imp] = true
	}

	expectedImports := []string{"catch2/catch_test_macros.hpp", "services/user.h"}
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

	expectedCalls := []string{"userService.create", "repository.save"}
	for _, call := range expectedCalls {
		if !callSet[call] {
			t.Errorf("expected call %q, got %v", call, hints.Calls)
		}
	}

	// REQUIRE should be excluded
	if callSet["REQUIRE"] {
		t.Errorf("expected REQUIRE to be excluded, got %v", hints.Calls)
	}
}

func TestCppExtractor_Extract_Deduplication(t *testing.T) {
	source := []byte(`
#include <iostream>
#include <iostream>

void test() {
    userService.create(1);
    userService.create(2);
}
`)

	extractor := &CppExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Count occurrences
	importCount := 0
	for _, imp := range hints.Imports {
		if imp == "iostream" {
			importCount++
		}
	}
	if importCount != 1 {
		t.Errorf("expected 'iostream' to appear once, got %d times in %v", importCount, hints.Imports)
	}

	callCount := 0
	for _, call := range hints.Calls {
		if call == "userService.create" {
			callCount++
		}
	}
	if callCount != 1 {
		t.Errorf("expected 'userService.create' to appear once, got %d times in %v", callCount, hints.Calls)
	}
}

func TestGetExtractor_Cpp(t *testing.T) {
	ext := GetExtractor(domain.LanguageCpp)
	if ext == nil {
		t.Error("expected extractor for C++, got nil")
	}

	_, ok := ext.(*CppExtractor)
	if !ok {
		t.Errorf("expected CppExtractor, got %T", ext)
	}
}

func TestCppExtractor_Extract_NamespacedCalls(t *testing.T) {
	source := []byte(`
#include <vector>

void test() {
    std::vector<int> v;
    MyNamespace::Service::getInstance();
    payment::gateway::process(order);
}
`)

	extractor := &CppExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	callSet := make(map[string]bool)
	for _, call := range hints.Calls {
		callSet[call] = true
	}

	// Should be normalized to 2 segments
	expectedCalls := []string{"MyNamespace.Service", "payment.gateway"}
	for _, call := range expectedCalls {
		if !callSet[call] {
			t.Errorf("expected %q call (2-segment normalized), got %v", call, hints.Calls)
		}
	}
}
