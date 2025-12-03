package jest

import (
	"testing"

	"github.com/specvital/core/domain"
)

func TestJestStrategy_CanHandle(t *testing.T) {
	s := &Strategy{}

	tests := []struct {
		filename string
		expected bool
	}{
		{"user.test.ts", true},
		{"user.spec.ts", true},
		{"user.test.tsx", true},
		{"user.spec.tsx", true},
		{"user.test.js", true},
		{"user.spec.js", true},
		{"user.test.jsx", true},
		{"user.spec.jsx", true},
		{"__tests__/user.ts", true},
		{"src/__tests__/user.tsx", true},
		{"user.ts", false},
		{"user.go", false},
		{"testuser.ts", false},
	}

	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			result := s.CanHandle(tc.filename, nil)
			if result != tc.expected {
				t.Errorf("CanHandle(%q) = %v, want %v", tc.filename, result, tc.expected)
			}
		})
	}
}

func TestJestStrategy_Parse_SimpleDescribe(t *testing.T) {
	s := &Strategy{}

	source := []byte(`
describe('UserService', () => {
  it('should create user', () => {
    expect(true).toBe(true);
  });

  it('should delete user', () => {
    expect(true).toBe(true);
  });
});
`)

	file, err := s.Parse(source, "user.test.ts")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if file.Framework != "jest" {
		t.Errorf("Framework = %q, want %q", file.Framework, "jest")
	}

	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	suite := file.Suites[0]
	if suite.Name != "UserService" {
		t.Errorf("Suite.Name = %q, want %q", suite.Name, "UserService")
	}

	if len(suite.Tests) != 2 {
		t.Fatalf("len(Tests) = %d, want 2", len(suite.Tests))
	}

	if suite.Tests[0].Name != "should create user" {
		t.Errorf("Test[0].Name = %q, want %q", suite.Tests[0].Name, "should create user")
	}
}

func TestJestStrategy_Parse_NestedDescribe(t *testing.T) {
	s := &Strategy{}

	source := []byte(`
describe('UserService', () => {
  describe('create', () => {
    it('should work', () => {});
  });

  describe('delete', () => {
    it('should work', () => {});
  });
});
`)

	file, err := s.Parse(source, "user.test.ts")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	suite := file.Suites[0]
	if len(suite.Suites) != 2 {
		t.Fatalf("len(nested Suites) = %d, want 2", len(suite.Suites))
	}

	if suite.Suites[0].Name != "create" {
		t.Errorf("Suite[0].Name = %q, want %q", suite.Suites[0].Name, "create")
	}

	if len(suite.Suites[0].Tests) != 1 {
		t.Errorf("Suite[0].Tests = %d, want 1", len(suite.Suites[0].Tests))
	}
}

func TestJestStrategy_Parse_SkipAndOnly(t *testing.T) {
	s := &Strategy{}

	source := []byte(`
describe('Suite', () => {
  it.skip('skipped test', () => {});
  it.only('only test', () => {});
  xit('x-prefixed skip', () => {});
  fit('f-prefixed only', () => {});
  test.skip('skipped via test', () => {});
  test.todo('todo test');
});
`)

	file, err := s.Parse(source, "user.test.ts")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	suite := file.Suites[0]

	testCases := []struct {
		name   string
		status domain.TestStatus
	}{
		{"skipped test", domain.TestStatusSkipped},
		{"only test", domain.TestStatusOnly},
		{"x-prefixed skip", domain.TestStatusSkipped},
		{"f-prefixed only", domain.TestStatusOnly},
		{"skipped via test", domain.TestStatusSkipped},
		{"todo test", domain.TestStatusSkipped},
	}

	if len(suite.Tests) != len(testCases) {
		t.Fatalf("len(Tests) = %d, want %d", len(suite.Tests), len(testCases))
	}

	for i, tc := range testCases {
		if suite.Tests[i].Name != tc.name {
			t.Errorf("Test[%d].Name = %q, want %q", i, suite.Tests[i].Name, tc.name)
		}
		if suite.Tests[i].Status != tc.status {
			t.Errorf("Test[%d].Status = %q, want %q", i, suite.Tests[i].Status, tc.status)
		}
	}
}

func TestJestStrategy_Parse_DescribeSkipOnly(t *testing.T) {
	s := &Strategy{}

	source := []byte(`
describe.skip('skipped suite', () => {
  it('test in skipped', () => {});
});

describe.only('only suite', () => {
  it('test in only', () => {});
});

xdescribe('x-prefixed suite', () => {
  it('test in x-prefixed', () => {});
});

fdescribe('f-prefixed suite', () => {
  it('test in f-prefixed', () => {});
});
`)

	file, err := s.Parse(source, "user.test.ts")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(file.Suites) != 4 {
		t.Fatalf("len(Suites) = %d, want 4", len(file.Suites))
	}

	expectedSuites := []struct {
		name   string
		status domain.TestStatus
	}{
		{"skipped suite", domain.TestStatusSkipped},
		{"only suite", domain.TestStatusOnly},
		{"x-prefixed suite", domain.TestStatusSkipped},
		{"f-prefixed suite", domain.TestStatusOnly},
	}

	for i, expected := range expectedSuites {
		if file.Suites[i].Name != expected.name {
			t.Errorf("Suite[%d].Name = %q, want %q", i, file.Suites[i].Name, expected.name)
		}
		if file.Suites[i].Status != expected.status {
			t.Errorf("Suite[%d].Status = %q, want %q", i, file.Suites[i].Status, expected.status)
		}
	}
}

func TestJestStrategy_Parse_DescribeEach(t *testing.T) {
	s := &Strategy{}

	source := []byte(`
describe.each([['a'], ['b']])('case %s', (v) => {
  it('works', () => {});
});
`)

	file, err := s.Parse(source, "user.test.ts")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(file.Suites) != 2 {
		t.Fatalf("len(Suites) = %d, want 2", len(file.Suites))
	}

	if file.Suites[0].Name != "case a" {
		t.Errorf("Suite[0].Name = %q, want %q", file.Suites[0].Name, "case a")
	}

	if file.Suites[1].Name != "case b" {
		t.Errorf("Suite[1].Name = %q, want %q", file.Suites[1].Name, "case b")
	}

	// Each suite should have the nested test
	for i, suite := range file.Suites {
		if len(suite.Tests) != 1 {
			t.Errorf("Suite[%d].Tests = %d, want 1", i, len(suite.Tests))
		}
	}
}

func TestJestStrategy_Parse_ItEach(t *testing.T) {
	s := &Strategy{}

	source := []byte(`
describe('Suite', () => {
  it.each([[1], [2], [3]])('test %s', (n) => {});
});
`)

	file, err := s.Parse(source, "user.test.ts")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	suite := file.Suites[0]
	if len(suite.Tests) != 3 {
		t.Fatalf("len(Tests) = %d, want 3", len(suite.Tests))
	}

	expectedNames := []string{"test 1", "test 2", "test 3"}
	for i, name := range expectedNames {
		if suite.Tests[i].Name != name {
			t.Errorf("Test[%d].Name = %q, want %q", i, suite.Tests[i].Name, name)
		}
	}
}

func TestJestStrategy_Parse_TopLevelTests(t *testing.T) {
	s := &Strategy{}

	source := []byte(`
it('top level test', () => {});
test('another top level', () => {});
`)

	file, err := s.Parse(source, "user.test.ts")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(file.Tests) != 2 {
		t.Fatalf("len(Tests) = %d, want 2", len(file.Tests))
	}

	if file.Tests[0].Name != "top level test" {
		t.Errorf("Test[0].Name = %q, want %q", file.Tests[0].Name, "top level test")
	}
}

func TestJestStrategy_Parse_Location(t *testing.T) {
	s := &Strategy{}

	source := []byte(`describe('Suite', () => {
  it('test', () => {});
});`)

	file, err := s.Parse(source, "user.test.ts")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	suite := file.Suites[0]

	// Suite starts at line 1
	if suite.Location.StartLine != 1 {
		t.Errorf("Suite.Location.StartLine = %d, want 1", suite.Location.StartLine)
	}

	// Test starts at line 2
	if suite.Tests[0].Location.StartLine != 2 {
		t.Errorf("Test.Location.StartLine = %d, want 2", suite.Tests[0].Location.StartLine)
	}

	if suite.Location.File != "user.test.ts" {
		t.Errorf("Suite.Location.File = %q, want %q", suite.Location.File, "user.test.ts")
	}
}

func TestJestStrategy_Parse_VerificationExample(t *testing.T) {
	// Verification test from plan.md
	s := &Strategy{}

	source := []byte(`
describe('UserService', () => {
  it.skip('should skip', () => {})
  describe.each([['a'], ['b']])('case %s', (v) => {
    it('works', () => {})
  })
})
`)

	file, err := s.Parse(source, "user.test.ts")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Expect: 1 top-level suite 'UserService'
	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	userService := file.Suites[0]
	if userService.Name != "UserService" {
		t.Errorf("Suite.Name = %q, want %q", userService.Name, "UserService")
	}

	// Expect: 1 skipped test + 2 nested suites
	if len(userService.Tests) != 1 {
		t.Fatalf("len(Tests) = %d, want 1", len(userService.Tests))
	}

	skipTest := userService.Tests[0]
	if skipTest.Name != "should skip" {
		t.Errorf("Test.Name = %q, want %q", skipTest.Name, "should skip")
	}
	if skipTest.Status != domain.TestStatusSkipped {
		t.Errorf("Test.Status = %q, want %q", skipTest.Status, domain.TestStatusSkipped)
	}

	// 2 cases from describe.each
	if len(userService.Suites) != 2 {
		t.Fatalf("len(nested Suites) = %d, want 2", len(userService.Suites))
	}

	if userService.Suites[0].Name != "case a" {
		t.Errorf("Suite[0].Name = %q, want %q", userService.Suites[0].Name, "case a")
	}

	if userService.Suites[1].Name != "case b" {
		t.Errorf("Suite[1].Name = %q, want %q", userService.Suites[1].Name, "case b")
	}

	// Each nested suite has 1 test
	for i, suite := range userService.Suites {
		if len(suite.Tests) != 1 {
			t.Errorf("Suite[%d].Tests = %d, want 1", i, len(suite.Tests))
		}
		if suite.Tests[0].Name != "works" {
			t.Errorf("Suite[%d].Tests[0].Name = %q, want %q", i, suite.Tests[0].Name, "works")
		}
	}

	// Total tests: 1 (skipped) + 2 (from each case)
	totalTests := file.CountTests()
	if totalTests != 3 {
		t.Errorf("CountTests() = %d, want 3", totalTests)
	}
}

func TestJestStrategy_Parse_JavaScript(t *testing.T) {
	s := &Strategy{}

	source := []byte(`
describe('JSTest', () => {
  it('works in JS', () => {});
});
`)

	file, err := s.Parse(source, "user.test.js")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if file.Language != domain.LanguageJavaScript {
		t.Errorf("Language = %q, want %q", file.Language, domain.LanguageJavaScript)
	}

	if len(file.Suites) != 1 {
		t.Fatalf("len(Suites) = %d, want 1", len(file.Suites))
	}

	if file.Suites[0].Name != "JSTest" {
		t.Errorf("Suite.Name = %q, want %q", file.Suites[0].Name, "JSTest")
	}
}
