package domain_hints

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
)

func TestCSharpExtractor_Extract_Usings(t *testing.T) {
	source := []byte(`
using System;
using System.Collections.Generic;
using NUnit.Framework;
using MyApp.Services;
using MyApp.Models;

namespace MyApp.Tests
{
    public class CalculatorTests
    {
    }
}
`)

	extractor := &CSharpExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedUsings := map[string]bool{
		"System":                     true,
		"System.Collections.Generic": true,
		"NUnit.Framework":            true,
		"MyApp.Services":             true,
		"MyApp.Models":               true,
	}

	usingSet := make(map[string]bool)
	for _, u := range hints.Imports {
		usingSet[u] = true
	}

	for u := range expectedUsings {
		if !usingSet[u] {
			t.Errorf("expected using %q to be included, got %v", u, hints.Imports)
		}
	}
}

func TestCSharpExtractor_Extract_Calls(t *testing.T) {
	source := []byte(`
using NUnit.Framework;

namespace MyApp.Tests
{
    public class CalculatorTests
    {
        [Test]
        public void TestAdd()
        {
            var calculator = new Calculator();
            var result = calculator.Add(1, 2);
            userService.FindById(123);
            paymentGateway.Process(order);
        }
    }
}
`)

	extractor := &CSharpExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedCalls := map[string]bool{
		"calculator.Add":       true,
		"userService.FindById": true,
		"paymentGateway.Process": true,
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

func TestCSharpExtractor_Extract_EmptyFile(t *testing.T) {
	source := []byte(`// empty file`)

	extractor := &CSharpExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints != nil {
		t.Errorf("expected nil for empty file, got %+v", hints)
	}
}

func TestCSharpExtractor_Extract_TestFrameworkCalls(t *testing.T) {
	source := []byte(`
using NUnit.Framework;

namespace MyApp.Tests
{
    public class CalculatorTests
    {
        [Test]
        public void TestAdd()
        {
            var calculator = new Calculator();
            Assert.AreEqual(2, calculator.Add(1, 1));
            Assert.IsTrue(calculator.IsPositive(5));
            Assert.Throws<Exception>(() => calculator.Divide(1, 0));
            userService.Validate(user);
        }
    }
}
`)

	extractor := &CSharpExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	callSet := make(map[string]bool)
	for _, call := range hints.Calls {
		callSet[call] = true
	}

	// Test framework calls should be excluded
	excludedCalls := []string{"Assert.AreEqual", "Assert.IsTrue", "Assert.Throws"}
	for _, call := range excludedCalls {
		if callSet[call] {
			t.Errorf("expected test framework call %q to be excluded", call)
		}
	}

	// Domain calls should be included
	if !callSet["calculator.Add"] {
		t.Errorf("expected calculator.Add call, got %v", hints.Calls)
	}
	if !callSet["userService.Validate"] {
		t.Errorf("expected userService.Validate call, got %v", hints.Calls)
	}
}

func TestCSharpExtractor_Extract_UsingAlias(t *testing.T) {
	source := []byte(`
using System;
using Env = System.Environment;
using Console = System.Console;

namespace MyApp {}
`)

	extractor := &CSharpExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	usingSet := make(map[string]bool)
	for _, u := range hints.Imports {
		usingSet[u] = true
	}

	// Should include the aliased namespaces
	if !usingSet["System"] {
		t.Errorf("expected System using, got %v", hints.Imports)
	}
	if !usingSet["System.Environment"] {
		t.Errorf("expected System.Environment using (aliased), got %v", hints.Imports)
	}
	if !usingSet["System.Console"] {
		t.Errorf("expected System.Console using (aliased), got %v", hints.Imports)
	}
}

func TestCSharpExtractor_Extract_ChainedCalls(t *testing.T) {
	source := []byte(`
namespace MyApp.Tests
{
    public class Test
    {
        void TestMethod()
        {
            // Long chains should be normalized to 2 segments
            client.Api.Users.FindAll();
            response.Data.Items.First().GetValue();
        }
    }
}
`)

	extractor := &CSharpExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	callSet := make(map[string]bool)
	for _, call := range hints.Calls {
		callSet[call] = true
	}

	// Should be normalized to 2 segments
	expectedCalls := []string{"client.Api", "response.Data"}
	for _, call := range expectedCalls {
		if !callSet[call] {
			t.Errorf("expected %q call (2-segment normalized), got %v", call, hints.Calls)
		}
	}
}

func TestCSharpExtractor_Extract_StaticUsing(t *testing.T) {
	source := []byte(`
using System;
using static System.Console;
using static System.Math;

namespace MyApp {}
`)

	extractor := &CSharpExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	usingSet := make(map[string]bool)
	for _, u := range hints.Imports {
		usingSet[u] = true
	}

	// Static usings should be included
	if !usingSet["System"] {
		t.Errorf("expected System using, got %v", hints.Imports)
	}
	if !usingSet["System.Console"] {
		t.Errorf("expected System.Console using (static), got %v", hints.Imports)
	}
	if !usingSet["System.Math"] {
		t.Errorf("expected System.Math using (static), got %v", hints.Imports)
	}
}

func TestCSharpExtractor_Extract_GlobalUsing(t *testing.T) {
	source := []byte(`
global using System;
global using System.Linq;

namespace MyApp {}
`)

	extractor := &CSharpExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	usingSet := make(map[string]bool)
	for _, u := range hints.Imports {
		usingSet[u] = true
	}

	// Global usings should be included
	if !usingSet["System"] {
		t.Errorf("expected System using (global), got %v", hints.Imports)
	}
	if !usingSet["System.Linq"] {
		t.Errorf("expected System.Linq using (global), got %v", hints.Imports)
	}
}

func TestCSharpExtractor_Extract_XUnitTest(t *testing.T) {
	source := []byte(`
using Xunit;
using FluentAssertions;
using MyApp.Services;

namespace MyApp.Tests
{
    public class UserServiceTests
    {
        [Fact]
        public void GetUser_ReturnsUser()
        {
            var service = new UserService();
            var result = service.GetUser(1);
            result.Should().NotBeNull();
            orderService.CreateOrder(result);
        }
    }
}
`)

	extractor := &CSharpExtractor{}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Check usings
	usingSet := make(map[string]bool)
	for _, u := range hints.Imports {
		usingSet[u] = true
	}

	expectedUsings := []string{"Xunit", "FluentAssertions", "MyApp.Services"}
	for _, u := range expectedUsings {
		if !usingSet[u] {
			t.Errorf("expected using %q, got %v", u, hints.Imports)
		}
	}

	// Check calls
	callSet := make(map[string]bool)
	for _, call := range hints.Calls {
		callSet[call] = true
	}

	// Domain calls should be included
	if !callSet["service.GetUser"] {
		t.Errorf("expected service.GetUser call, got %v", hints.Calls)
	}
	if !callSet["orderService.CreateOrder"] {
		t.Errorf("expected orderService.CreateOrder call, got %v", hints.Calls)
	}

	// FluentAssertions calls should be excluded (Should is in the filter)
	if callSet["Should.NotBeNull"] {
		t.Errorf("expected FluentAssertions call to be excluded, got %v", hints.Calls)
	}
}

func TestGetExtractor_CSharp(t *testing.T) {
	ext := GetExtractor(domain.LanguageCSharp)
	if ext == nil {
		t.Error("expected extractor for CSharp, got nil")
	}

	_, ok := ext.(*CSharpExtractor)
	if !ok {
		t.Errorf("expected CSharpExtractor, got %T", ext)
	}
}
