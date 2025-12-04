package gotesting

import (
	"context"
	"testing"

	"github.com/specvital/core/domain"
)

func TestStrategy_Name(t *testing.T) {
	s := NewStrategy()
	if got := s.Name(); got != "go-testing" {
		t.Errorf("Name() = %v, want %v", got, "go-testing")
	}
}

func TestStrategy_Priority(t *testing.T) {
	s := NewStrategy()
	if got := s.Priority(); got != 100 {
		t.Errorf("Priority() = %v, want %v", got, 100)
	}
}

func TestStrategy_Languages(t *testing.T) {
	s := NewStrategy()
	langs := s.Languages()
	if len(langs) != 1 || langs[0] != domain.LanguageGo {
		t.Errorf("Languages() = %v, want [go]", langs)
	}
}

func TestStrategy_CanHandle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"valid test file", "user_test.go", true},
		{"test file in directory", "pkg/service/user_test.go", true},
		{"non-test go file", "user.go", false},
		{"typescript test file", "user.test.ts", false},
		{"javascript test file", "user.spec.js", false},
		{"test directory", "test/main.go", false},
	}

	s := NewStrategy()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := s.CanHandle(tt.filename, nil); got != tt.want {
				t.Errorf("CanHandle(%v) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestStrategy_Parse_SimpleTestFunction(t *testing.T) {
	source := []byte(`package user

import "testing"

func TestCreate(t *testing.T) {
	// test implementation
}
`)

	s := NewStrategy()
	result, err := s.Parse(context.Background(), source, "user_test.go")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if result.Framework != "go-testing" {
		t.Errorf("Framework = %v, want go-testing", result.Framework)
	}

	if result.Language != domain.LanguageGo {
		t.Errorf("Language = %v, want go", result.Language)
	}

	if len(result.Tests) != 1 {
		t.Fatalf("len(Tests) = %v, want 1", len(result.Tests))
	}

	if result.Tests[0].Name != "TestCreate" {
		t.Errorf("Tests[0].Name = %v, want TestCreate", result.Tests[0].Name)
	}
}

func TestStrategy_Parse_MultipleTestFunctions(t *testing.T) {
	source := []byte(`package user

import "testing"

func TestCreate(t *testing.T) {}
func TestUpdate(t *testing.T) {}
func TestDelete(t *testing.T) {}
`)

	s := NewStrategy()
	result, err := s.Parse(context.Background(), source, "user_test.go")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Tests) != 3 {
		t.Fatalf("len(Tests) = %v, want 3", len(result.Tests))
	}

	expectedNames := []string{"TestCreate", "TestUpdate", "TestDelete"}
	for i, name := range expectedNames {
		if result.Tests[i].Name != name {
			t.Errorf("Tests[%d].Name = %v, want %v", i, result.Tests[i].Name, name)
		}
	}
}

func TestStrategy_Parse_Subtests(t *testing.T) {
	source := []byte(`package user

import "testing"

func TestUser(t *testing.T) {
	t.Run("create", func(t *testing.T) {})
	t.Run("update", func(t *testing.T) {})
}
`)

	s := NewStrategy()
	result, err := s.Parse(context.Background(), source, "user_test.go")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Tests) != 0 {
		t.Errorf("len(Tests) = %v, want 0 (should be in Suites)", len(result.Tests))
	}

	if len(result.Suites) != 1 {
		t.Fatalf("len(Suites) = %v, want 1", len(result.Suites))
	}

	suite := result.Suites[0]
	if suite.Name != "TestUser" {
		t.Errorf("Suites[0].Name = %v, want TestUser", suite.Name)
	}

	if len(suite.Tests) != 2 {
		t.Fatalf("len(Suites[0].Tests) = %v, want 2", len(suite.Tests))
	}

	expectedSubtests := []string{"create", "update"}
	for i, name := range expectedSubtests {
		if suite.Tests[i].Name != name {
			t.Errorf("Suites[0].Tests[%d].Name = %v, want %v", i, suite.Tests[i].Name, name)
		}
	}
}

func TestStrategy_Parse_MixedTestsAndSubtests(t *testing.T) {
	source := []byte(`package user

import "testing"

func TestSimple(t *testing.T) {}

func TestWithSubtests(t *testing.T) {
	t.Run("sub1", func(t *testing.T) {})
	t.Run("sub2", func(t *testing.T) {})
}

func TestAnotherSimple(t *testing.T) {}
`)

	s := NewStrategy()
	result, err := s.Parse(context.Background(), source, "user_test.go")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Tests) != 2 {
		t.Errorf("len(Tests) = %v, want 2", len(result.Tests))
	}

	if len(result.Suites) != 1 {
		t.Fatalf("len(Suites) = %v, want 1", len(result.Suites))
	}
}

func TestStrategy_Parse_IgnoresNonTestFunctions(t *testing.T) {
	source := []byte(`package user

import "testing"

func TestValid(t *testing.T) {}
func helperFunction() {}
func Test(t *testing.T) {} // Too short, should be ignored
func BenchmarkSomething(b *testing.B) {} // Benchmark, not test
func Test1(t *testing.T) {} // lowercase after Test, should be ignored
func Test_(t *testing.T) {} // underscore after Test, should be ignored
func TestWrongParams(i int, t *testing.T) {} // wrong parameters, should be ignored
`)

	s := NewStrategy()
	result, err := s.Parse(context.Background(), source, "user_test.go")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Tests) != 1 {
		t.Fatalf("len(Tests) = %v, want 1", len(result.Tests))
	}

	if result.Tests[0].Name != "TestValid" {
		t.Errorf("Tests[0].Name = %v, want TestValid", result.Tests[0].Name)
	}
}

func TestStrategy_Parse_Location(t *testing.T) {
	source := []byte(`package user

import "testing"

func TestCreate(t *testing.T) {
	// test
}
`)

	s := NewStrategy()
	result, err := s.Parse(context.Background(), source, "user_test.go")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Tests) != 1 {
		t.Fatalf("len(Tests) = %v, want 1", len(result.Tests))
	}

	loc := result.Tests[0].Location
	if loc.File != "user_test.go" {
		t.Errorf("Location.File = %v, want user_test.go", loc.File)
	}
	if loc.StartLine != 5 {
		t.Errorf("Location.StartLine = %v, want 5", loc.StartLine)
	}
}

func TestStrategy_Parse_VerificationExample(t *testing.T) {
	// Test from plan.md verification method
	source := []byte(`package user

import "testing"

func TestUser(t *testing.T) {
	t.Run("create", func(t *testing.T) {})
}
`)

	s := NewStrategy()
	result, err := s.Parse(context.Background(), source, "user_test.go")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Should have TestUser suite + create subtest
	if len(result.Suites) != 1 {
		t.Fatalf("len(Suites) = %v, want 1", len(result.Suites))
	}

	if result.Suites[0].Name != "TestUser" {
		t.Errorf("Suites[0].Name = %v, want TestUser", result.Suites[0].Name)
	}

	if len(result.Suites[0].Tests) != 1 {
		t.Fatalf("len(Suites[0].Tests) = %v, want 1", len(result.Suites[0].Tests))
	}

	if result.Suites[0].Tests[0].Name != "create" {
		t.Errorf("Suites[0].Tests[0].Name = %v, want create", result.Suites[0].Tests[0].Name)
	}
}

func TestStrategy_Parse_RawStringLiteral(t *testing.T) {
	source := []byte(`package user

import "testing"

func TestUser(t *testing.T) {
	t.Run(` + "`" + `raw string name` + "`" + `, func(t *testing.T) {})
}
`)

	s := NewStrategy()
	result, err := s.Parse(context.Background(), source, "user_test.go")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Suites) != 1 {
		t.Fatalf("len(Suites) = %v, want 1", len(result.Suites))
	}

	if len(result.Suites[0].Tests) != 1 {
		t.Fatalf("len(Suites[0].Tests) = %v, want 1", len(result.Suites[0].Tests))
	}

	if result.Suites[0].Tests[0].Name != "raw string name" {
		t.Errorf("subtest name = %v, want 'raw string name'", result.Suites[0].Tests[0].Name)
	}
}

func TestStrategy_Parse_NestedSubtests(t *testing.T) {
	source := []byte(`package user

import "testing"

func TestUser(t *testing.T) {
	t.Run("level1", func(t *testing.T) {
		t.Run("level2", func(t *testing.T) {})
	})
}
`)

	s := NewStrategy()
	result, err := s.Parse(context.Background(), source, "user_test.go")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Suites) != 1 {
		t.Fatalf("len(Suites) = %v, want 1", len(result.Suites))
	}

	// Both subtests should be extracted (flat structure)
	if len(result.Suites[0].Tests) != 2 {
		t.Fatalf("len(Suites[0].Tests) = %v, want 2", len(result.Suites[0].Tests))
	}
}

func TestStrategy_Parse_EmptySource(t *testing.T) {
	source := []byte(``)

	s := NewStrategy()
	result, err := s.Parse(context.Background(), source, "empty_test.go")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Tests) != 0 {
		t.Errorf("len(Tests) = %v, want 0", len(result.Tests))
	}
	if len(result.Suites) != 0 {
		t.Errorf("len(Suites) = %v, want 0", len(result.Suites))
	}
}

func TestStrategy_Parse_NoTestFunctions(t *testing.T) {
	source := []byte(`package user

func helperFunction() {}
func anotherHelper(s string) int { return len(s) }
`)

	s := NewStrategy()
	result, err := s.Parse(context.Background(), source, "user_test.go")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Tests) != 0 {
		t.Errorf("len(Tests) = %v, want 0", len(result.Tests))
	}
}

func TestStrategy_Parse_TestFunctionWithNoBody(t *testing.T) {
	// Interface method declaration style (no body)
	source := []byte(`package user

import "testing"

type Tester interface {
	TestSomething(t *testing.T)
}

func TestReal(t *testing.T) {}
`)

	s := NewStrategy()
	result, err := s.Parse(context.Background(), source, "user_test.go")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Should only find TestReal, not interface method
	if len(result.Tests) != 1 {
		t.Fatalf("len(Tests) = %v, want 1", len(result.Tests))
	}
	if result.Tests[0].Name != "TestReal" {
		t.Errorf("Tests[0].Name = %v, want TestReal", result.Tests[0].Name)
	}
}
