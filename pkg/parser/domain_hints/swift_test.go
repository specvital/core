package domain_hints

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
)

func TestSwiftExtractor_Extract_ImportStatements(t *testing.T) {
	source := []byte(`
import Foundation
import XCTest
import SwiftUI
import UIKit
`)

	extractor := &SwiftExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"Foundation": true,
		"XCTest":     true,
		"SwiftUI":    true,
		"UIKit":      true,
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

func TestSwiftExtractor_Extract_TestableImport(t *testing.T) {
	source := []byte(`
import XCTest
@testable import MyApp
@testable import CoreModule
`)

	extractor := &SwiftExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"XCTest":     true,
		"MyApp":      true,
		"CoreModule": true,
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

func TestSwiftExtractor_Extract_DottedImport(t *testing.T) {
	source := []byte(`
import UIKit.UIView
import Foundation.NSObject
`)

	extractor := &SwiftExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"UIKit.UIView":       true,
		"Foundation.NSObject": true,
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

func TestSwiftExtractor_Extract_MethodCalls(t *testing.T) {
	source := []byte(`
import Foundation

class TestService {
    func testMethod() {
        userService.create(user)
        PaymentGateway.process(payment)
        notificationService.sendEmail(to: user)
    }
}
`)

	extractor := &SwiftExtractor{}
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

func TestSwiftExtractor_Extract_EmptyFile(t *testing.T) {
	source := []byte(`// empty file`)

	extractor := &SwiftExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints != nil {
		t.Errorf("expected nil for empty file, got %+v", hints)
	}
}

func TestSwiftExtractor_Extract_TestFrameworkCalls(t *testing.T) {
	source := []byte(`
import XCTest

class PaymentTests: XCTestCase {
    func testPayment() {
        XCTAssertEqual(result, expected)
        XCTAssertTrue(condition)
        print("debug output")

        paymentService.process(order)
    }
}
`)

	extractor := &SwiftExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	callSet := make(map[string]bool)
	for _, call := range hints.Calls {
		callSet[call] = true
	}

	// Test framework calls should be excluded
	excludedCalls := []string{"XCTAssertEqual", "XCTAssertTrue", "print"}
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

func TestSwiftExtractor_Extract_XCTestFile(t *testing.T) {
	source := []byte(`
import XCTest
@testable import MyApp

class UserServiceTests: XCTestCase {
    var sut: UserService!
    var mockRepository: MockUserRepository!

    override func setUp() {
        super.setUp()
        mockRepository = MockUserRepository()
        sut = UserService(repository: mockRepository)
    }

    func testCreateUser() {
        let user = User(name: "Test")

        sut.create(user)
        userNotification.send(to: user)
        analyticsService.track(event: "user_created")

        XCTAssertEqual(mockRepository.savedUsers.count, 1)
    }
}
`)

	extractor := &SwiftExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Verify imports
	importSet := make(map[string]bool)
	for _, imp := range hints.Imports {
		importSet[imp] = true
	}

	expectedImports := []string{"XCTest", "MyApp"}
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

	expectedCalls := []string{"sut.create", "userNotification.send", "analyticsService.track"}
	for _, call := range expectedCalls {
		if !callSet[call] {
			t.Errorf("expected call %q, got %v", call, hints.Calls)
		}
	}
}

func TestSwiftExtractor_Extract_Deduplication(t *testing.T) {
	source := []byte(`
import Foundation
import Foundation

func test() {
    userService.create(1)
    userService.create(2)
}
`)

	extractor := &SwiftExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Count occurrences
	importCount := 0
	for _, imp := range hints.Imports {
		if imp == "Foundation" {
			importCount++
		}
	}
	if importCount != 1 {
		t.Errorf("expected 'Foundation' to appear once, got %d times in %v", importCount, hints.Imports)
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

func TestGetExtractor_Swift(t *testing.T) {
	ext := GetExtractor(domain.LanguageSwift)
	if ext == nil {
		t.Error("expected extractor for Swift, got nil")
	}

	_, ok := ext.(*SwiftExtractor)
	if !ok {
		t.Errorf("expected SwiftExtractor, got %T", ext)
	}
}

func TestSwiftExtractor_Extract_SwiftTestingFramework(t *testing.T) {
	source := []byte(`
import Testing
@testable import MyApp

@Suite("Payment Tests")
struct PaymentTests {
    @Test("processes payment correctly")
    func testPaymentProcessing() async throws {
        let service = PaymentService()

        #expect(service.isReady)
        try #require(service.configure())

        paymentGateway.process(amount: 100)
        notificationService.sendReceipt(to: user)
    }
}
`)

	extractor := &SwiftExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Verify imports
	importSet := make(map[string]bool)
	for _, imp := range hints.Imports {
		importSet[imp] = true
	}

	expectedImports := []string{"Testing", "MyApp"}
	for _, imp := range expectedImports {
		if !importSet[imp] {
			t.Errorf("expected import %q, got %v", imp, hints.Imports)
		}
	}

	// Verify calls - domain calls should be included
	callSet := make(map[string]bool)
	for _, call := range hints.Calls {
		callSet[call] = true
	}

	expectedCalls := []string{"paymentGateway.process", "notificationService.sendReceipt"}
	for _, call := range expectedCalls {
		if !callSet[call] {
			t.Errorf("expected call %q, got %v", call, hints.Calls)
		}
	}
}
