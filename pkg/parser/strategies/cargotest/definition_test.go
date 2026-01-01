package cargotest

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
)

func TestCargoTestParser_Parse(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		checkFunc func(t *testing.T, file *domain.TestFile)
	}{
		{
			name: "basic test function",
			source: `
#[test]
fn test_basic() {
    assert_eq!(1 + 1, 2);
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 1 {
					t.Errorf("expected 1 test, got %d", len(file.Tests))
					return
				}
				test := file.Tests[0]
				if test.Name != "test_basic" {
					t.Errorf("expected test name 'test_basic', got %q", test.Name)
				}
				if test.Status != domain.TestStatusActive {
					t.Errorf("expected status active, got %q", test.Status)
				}
			},
		},
		{
			name: "test with ignore attribute",
			source: `
#[test]
#[ignore]
fn test_ignored() {
    assert!(true);
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 1 {
					t.Errorf("expected 1 test, got %d", len(file.Tests))
					return
				}
				test := file.Tests[0]
				if test.Name != "test_ignored" {
					t.Errorf("expected test name 'test_ignored', got %q", test.Name)
				}
				if test.Status != domain.TestStatusSkipped {
					t.Errorf("expected status skipped, got %q", test.Status)
				}
				if test.Modifier != "#[ignore]" {
					t.Errorf("expected modifier '#[ignore]', got %q", test.Modifier)
				}
			},
		},
		{
			name: "test with should_panic modifier preserved",
			source: `
#[test]
#[should_panic]
fn test_panics() {
    panic!("expected panic");
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 1 {
					t.Errorf("expected 1 test, got %d", len(file.Tests))
					return
				}
				test := file.Tests[0]
				if test.Name != "test_panics" {
					t.Errorf("expected test name 'test_panics', got %q", test.Name)
				}
				if test.Status != domain.TestStatusActive {
					t.Errorf("expected status active, got %q", test.Status)
				}
				if test.Modifier != "#[should_panic]" {
					t.Errorf("expected modifier '#[should_panic]', got %q", test.Modifier)
				}
			},
		},
		{
			name: "test with should_panic expected message",
			source: `
#[test]
#[should_panic(expected = "division by zero")]
fn test_panic_message() {
    let _ = 1 / 0;
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 1 {
					t.Errorf("expected 1 test, got %d", len(file.Tests))
					return
				}
				test := file.Tests[0]
				if test.Name != "test_panic_message" {
					t.Errorf("expected test name 'test_panic_message', got %q", test.Name)
				}
				// Should preserve full attribute including expected message
				if test.Modifier != `#[should_panic(expected = "division by zero")]` {
					t.Errorf("expected modifier with expected message, got %q", test.Modifier)
				}
			},
		},
		{
			name: "test with ignore and should_panic combined",
			source: `
#[test]
#[ignore]
#[should_panic]
fn test_ignored_panic() {
    panic!("ignored");
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 1 {
					t.Errorf("expected 1 test, got %d", len(file.Tests))
					return
				}
				test := file.Tests[0]
				if test.Status != domain.TestStatusSkipped {
					t.Errorf("expected status skipped, got %q", test.Status)
				}
				// Both modifiers should be preserved
				if test.Modifier != "#[ignore] #[should_panic]" {
					t.Errorf("expected combined modifiers, got %q", test.Modifier)
				}
			},
		},
		{
			name: "multiple test functions",
			source: `
#[test]
fn test_one() {
    assert!(true);
}

#[test]
fn test_two() {
    assert_eq!(2, 2);
}

#[test]
fn test_three() {
    assert_ne!(1, 2);
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 3 {
					t.Errorf("expected 3 tests, got %d", len(file.Tests))
					return
				}
				expectedNames := []string{"test_one", "test_two", "test_three"}
				for i, name := range expectedNames {
					if file.Tests[i].Name != name {
						t.Errorf("expected test[%d] name %q, got %q", i, name, file.Tests[i].Name)
					}
				}
			},
		},
		{
			name: "test module with cfg(test)",
			source: `
fn helper() -> i32 {
    42
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_helper() {
        assert_eq!(helper(), 42);
    }

    #[test]
    fn test_another() {
        assert!(true);
    }
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if suite.Name != "tests" {
					t.Errorf("expected suite name 'tests', got %q", suite.Name)
				}
				if len(suite.Tests) != 2 {
					t.Errorf("expected 2 tests in suite, got %d", len(suite.Tests))
					return
				}
				if suite.Tests[0].Name != "test_helper" {
					t.Errorf("expected test name 'test_helper', got %q", suite.Tests[0].Name)
				}
				if suite.Tests[1].Name != "test_another" {
					t.Errorf("expected test name 'test_another', got %q", suite.Tests[1].Name)
				}
			},
		},
		{
			name: "tests module by convention",
			source: `
mod tests {
    #[test]
    fn test_in_tests_module() {
        assert!(true);
    }
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if suite.Name != "tests" {
					t.Errorf("expected suite name 'tests', got %q", suite.Name)
				}
				if len(suite.Tests) != 1 {
					t.Errorf("expected 1 test, got %d", len(suite.Tests))
				}
			},
		},
		{
			name: "mixed ignored and active tests",
			source: `
#[test]
fn test_active() {
    assert!(true);
}

#[test]
#[ignore]
fn test_ignored() {
    assert!(true);
}

#[test]
fn test_active_too() {
    assert!(true);
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 3 {
					t.Errorf("expected 3 tests, got %d", len(file.Tests))
					return
				}

				if file.Tests[0].Status != domain.TestStatusActive {
					t.Errorf("test[0] expected active, got %q", file.Tests[0].Status)
				}
				if file.Tests[1].Status != domain.TestStatusSkipped {
					t.Errorf("test[1] expected skipped, got %q", file.Tests[1].Status)
				}
				if file.Tests[2].Status != domain.TestStatusActive {
					t.Errorf("test[2] expected active, got %q", file.Tests[2].Status)
				}
			},
		},
		{
			name: "non-test functions are ignored",
			source: `
fn helper_function() {
    println!("not a test");
}

#[test]
fn actual_test() {
    assert!(true);
}

fn another_helper() -> bool {
    true
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 1 {
					t.Errorf("expected 1 test, got %d", len(file.Tests))
					return
				}
				if file.Tests[0].Name != "actual_test" {
					t.Errorf("expected test name 'actual_test', got %q", file.Tests[0].Name)
				}
			},
		},
		{
			name: "nested test modules flatten tests",
			source: `
#[cfg(test)]
mod tests {
    mod unit {
        #[test]
        fn test_nested() {
            assert!(true);
        }
    }

    #[test]
    fn test_outer() {
        assert!(true);
    }
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 top-level suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if suite.Name != "tests" {
					t.Errorf("expected suite name 'tests', got %q", suite.Name)
				}
				// Current implementation: nested modules without #[cfg(test)] are parsed
				// but inner tests are flattened into the parent test module.
				// The "unit" module is not a test module (no #[cfg(test)]), so its tests
				// are added directly to the parent suite.
				if len(suite.Tests) != 2 {
					t.Errorf("expected 2 tests (flattened), got %d", len(suite.Tests))
					return
				}
				// Tests should be: test_nested (from unit module) and test_outer
				names := make(map[string]bool)
				for _, test := range suite.Tests {
					names[test.Name] = true
				}
				if !names["test_nested"] || !names["test_outer"] {
					t.Errorf("expected tests 'test_nested' and 'test_outer', got %v", names)
				}
			},
		},
		{
			name: "test location accuracy",
			source: `fn helper() {}

#[test]
fn test_basic() {
    assert_eq!(1, 1);
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 1 {
					t.Errorf("expected 1 test, got %d", len(file.Tests))
					return
				}
				test := file.Tests[0]
				// Line 4 is fn test_basic (0-indexed: line 3)
				// tree-sitter uses 0-indexed lines, but we add 1 in GetLocation
				if test.Location.StartLine != 4 {
					t.Errorf("expected start line 4, got %d", test.Location.StartLine)
				}
			},
		},
		{
			name: "macro-based test with test in name",
			source: `
rgtest!(basic_rgtest, |dir, cmd| {
    dir.create("test.txt", "hello");
    cmd.arg("--help");
});

rgtest!(another_test, |dir, cmd| {
    assert!(true);
});
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 2 {
					t.Errorf("expected 2 tests, got %d", len(file.Tests))
					return
				}
				if file.Tests[0].Name != "basic_rgtest" {
					t.Errorf("expected test name 'basic_rgtest', got %q", file.Tests[0].Name)
				}
				if file.Tests[0].Modifier != "rgtest!" {
					t.Errorf("expected modifier 'rgtest!', got %q", file.Tests[0].Modifier)
				}
				if file.Tests[1].Name != "another_test" {
					t.Errorf("expected test name 'another_test', got %q", file.Tests[1].Name)
				}
			},
		},
		{
			name: "macro-based test mixed with attribute-based test",
			source: `
#[test]
fn regular_test() {
    assert!(true);
}

rgtest!(macro_test, |dir, cmd| {
    cmd.arg("--version");
});

#[test]
fn another_regular() {
    assert_eq!(1, 1);
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 3 {
					t.Errorf("expected 3 tests, got %d", len(file.Tests))
					return
				}
				names := make(map[string]bool)
				for _, test := range file.Tests {
					names[test.Name] = true
				}
				if !names["regular_test"] || !names["macro_test"] || !names["another_regular"] {
					t.Errorf("expected all tests to be detected, got %v", names)
				}
			},
		},
		{
			name: "non-test macro is ignored",
			source: `
println!("hello world");

assert_eq!(1, 1);

format!("test: {}", value);
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 0 {
					t.Errorf("expected 0 tests, got %d", len(file.Tests))
				}
			},
		},
		{
			name: "test macro in test module",
			source: `
#[cfg(test)]
mod tests {
    rgtest!(module_test, |dir, cmd| {
        assert!(true);
    });
}
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Suites) != 1 {
					t.Errorf("expected 1 suite, got %d", len(file.Suites))
					return
				}
				suite := file.Suites[0]
				if len(suite.Tests) != 1 {
					t.Errorf("expected 1 test in suite, got %d", len(suite.Tests))
					return
				}
				if suite.Tests[0].Name != "module_test" {
					t.Errorf("expected test name 'module_test', got %q", suite.Tests[0].Name)
				}
			},
		},
		{
			name: "same-file macro_rules with test attribute detected",
			source: `
macro_rules! syntax {
    ($name:ident, $pat:expr, $tokens:expr) => {
        #[test]
        fn $name() {
            let pat = Glob::new($pat).unwrap();
            assert_eq!($tokens, pat.tokens.0);
        }
    };
}

syntax!(literal1, "a", vec![Literal('a')]);
syntax!(literal2, "ab", vec![Literal('a'), Literal('b')]);
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 2 {
					t.Errorf("expected 2 tests from macro invocations, got %d", len(file.Tests))
					return
				}
				if file.Tests[0].Name != "literal1" {
					t.Errorf("expected first test name 'literal1', got %q", file.Tests[0].Name)
				}
				if file.Tests[0].Modifier != "syntax!" {
					t.Errorf("expected modifier 'syntax!', got %q", file.Tests[0].Modifier)
				}
				if file.Tests[1].Name != "literal2" {
					t.Errorf("expected second test name 'literal2', got %q", file.Tests[1].Name)
				}
			},
		},
		{
			name: "macro without test attribute not detected",
			source: `
macro_rules! helper {
    ($name:ident) => {
        fn $name() {
            println!("helper function");
        }
    };
}

helper!(my_helper);
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 0 {
					t.Errorf("expected 0 tests (macro has no #[test]), got %d", len(file.Tests))
				}
			},
		},
		{
			name: "multi-arm macro with test in second arm",
			source: `
macro_rules! matches {
    ($name:ident, $pat:expr, $path:expr) => {
        matches!($name, $pat, $path, Options::default());
    };
    ($name:ident, $pat:expr, $path:expr, $options:expr) => {
        #[test]
        fn $name() {
            let matcher = create_matcher($pat, $options);
            assert!(matcher.is_match($path));
        }
    };
}

matches!(match1, "*.txt", "foo.txt");
matches!(match2, "*.rs", "lib.rs", Options::new());
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 2 {
					t.Errorf("expected 2 tests from multi-arm macro, got %d", len(file.Tests))
					return
				}
				if file.Tests[0].Name != "match1" {
					t.Errorf("expected first test name 'match1', got %q", file.Tests[0].Name)
				}
				if file.Tests[1].Name != "match2" {
					t.Errorf("expected second test name 'match2', got %q", file.Tests[1].Name)
				}
			},
		},
		{
			name: "mixed same-file macro and name-based macro",
			source: `
macro_rules! syntax {
    ($name:ident, $pat:expr) => {
        #[test]
        fn $name() {
            assert_eq!($pat, $pat);
        }
    };
}

syntax!(test_syntax, "pattern");
rgtest!(test_rg, |dir, cmd| { assert!(true); });
`,
			checkFunc: func(t *testing.T, file *domain.TestFile) {
				if len(file.Tests) != 2 {
					t.Errorf("expected 2 tests (one from same-file macro, one from name-based), got %d", len(file.Tests))
					return
				}
				names := make(map[string]bool)
				for _, test := range file.Tests {
					names[test.Name] = true
				}
				if !names["test_syntax"] {
					t.Errorf("expected test 'test_syntax' from same-file macro")
				}
				if !names["test_rg"] {
					t.Errorf("expected test 'test_rg' from name-based heuristic")
				}
			},
		},
	}

	parser := &CargoTestParser{}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := parser.Parse(ctx, []byte(tt.source), "test.rs")
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if file.Framework != frameworkName {
				t.Errorf("expected framework %q, got %q", frameworkName, file.Framework)
			}

			if file.Language != domain.LanguageRust {
				t.Errorf("expected language Rust, got %q", file.Language)
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, file)
			}
		})
	}
}

func TestCargoTestFileMatcher_Match(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantMatch bool
	}{
		{"test file suffix", "module_test.rs", true},
		{"test file in tests dir", "tests/integration_test.rs", true},
		{"tests directory file", "tests/common.rs", true},
		{"regular rust file", "main.rs", false},
		{"lib file", "lib.rs", false},
		{"mod file", "mod.rs", false},
	}

	matcher := &CargoTestFileMatcher{}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal := framework.Signal{
				Type:  framework.SignalFileName,
				Value: tt.filename,
			}
			result := matcher.Match(ctx, signal)
			if (result.Confidence > 0) != tt.wantMatch {
				t.Errorf("Match() = %v, want match = %v", result.Confidence > 0, tt.wantMatch)
			}
		})
	}
}

func TestCargoTestContentMatcher_Match(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantMatch bool
	}{
		{"test attribute", "#[test]\nfn test_foo() {}", true},
		{"cfg test attribute", "#[cfg(test)]\nmod tests {}", true},
		{"ignore attribute", "#[ignore]\nfn ignored() {}", true},
		{"should_panic attribute", "#[should_panic]\nfn panic_test() {}", true},
		{"plain rust code", "fn main() { println!(\"hello\"); }", false},
		{"struct definition", "struct Foo { bar: i32 }", false},
		{"macro-based test rgtest", "rgtest!(my_test, |dir| {});", true},
		{"macro-based test quicktest", "quicktest!(test_case, || {});", true},
		{"non-test macro", "println!(\"test\");", false},
	}

	matcher := &CargoTestContentMatcher{}
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal := framework.Signal{
				Type:    framework.SignalFileContent,
				Value:   tt.content,
				Context: []byte(tt.content),
			}
			result := matcher.Match(ctx, signal)
			if (result.Confidence > 0) != tt.wantMatch {
				t.Errorf("Match() = %v, want match = %v", result.Confidence > 0, tt.wantMatch)
			}
		})
	}
}
