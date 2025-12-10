package phpunit

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
	"github.com/specvital/core/pkg/parser/framework"
)

func TestNewDefinition(t *testing.T) {
	def := NewDefinition()

	if def.Name != "phpunit" {
		t.Errorf("expected Name='phpunit', got '%s'", def.Name)
	}
	if def.Priority != framework.PriorityGeneric {
		t.Errorf("expected Priority=%d, got %d", framework.PriorityGeneric, def.Priority)
	}
	if len(def.Languages) != 1 || def.Languages[0] != domain.LanguagePHP {
		t.Errorf("expected Languages=[php], got %v", def.Languages)
	}
	if def.Parser == nil {
		t.Error("expected Parser to be non-nil")
	}
	if len(def.Matchers) != 4 {
		t.Errorf("expected 4 Matchers, got %d", len(def.Matchers))
	}
}

func TestPHPUnitFileMatcher_Match(t *testing.T) {
	matcher := &PHPUnitFileMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		filename           string
		expectedConfidence int
	}{
		{"Test suffix", "UserTest.php", 20},
		{"Tests suffix", "UserTests.php", 20},
		{"Test prefix", "TestUser.php", 20},
		{"Test suffix with path", "tests/Unit/UserServiceTest.php", 20},
		{"regular php file", "User.php", 0},
		{"non-php file", "UserTest.java", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal := framework.Signal{
				Type:  framework.SignalFileName,
				Value: tt.filename,
			}

			result := matcher.Match(ctx, signal)

			if result.Confidence != tt.expectedConfidence {
				t.Errorf("expected Confidence=%d, got %d", tt.expectedConfidence, result.Confidence)
			}
		})
	}
}

func TestPHPUnitContentMatcher_Match(t *testing.T) {
	matcher := &PHPUnitContentMatcher{}
	ctx := context.Background()

	tests := []struct {
		name               string
		content            string
		expectedConfidence int
	}{
		{
			name: "extends TestCase",
			content: `<?php
class UserTest extends TestCase
{
    public function testUserCreate()
    {
    }
}
`,
			expectedConfidence: 40,
		},
		{
			name: "use PHPUnit statement",
			content: `<?php
use PHPUnit\Framework\TestCase;

class UserTest extends TestCase {}
`,
			expectedConfidence: 40,
		},
		{
			name: "@test annotation",
			content: `<?php
class UserTest extends TestCase
{
    /**
     * @test
     */
    public function it_creates_user()
    {
    }
}
`,
			expectedConfidence: 40,
		},
		{
			name: "@dataProvider annotation",
			content: `<?php
class UserTest extends TestCase
{
    /**
     * @dataProvider userProvider
     */
    public function testWithData($user)
    {
    }
}
`,
			expectedConfidence: 40,
		},
		{
			name: "#[Test] attribute",
			content: `<?php
class UserTest extends TestCase
{
    #[Test]
    public function userCreation()
    {
    }
}
`,
			expectedConfidence: 40,
		},
		{
			name: "PHPUnit assertion",
			content: `<?php
class UserTest extends TestCase
{
    public function testExample()
    {
        $this->assertEquals(1, 1);
    }
}
`,
			expectedConfidence: 40,
		},
		{
			name: "no PHPUnit patterns",
			content: `<?php
class User
{
    public function getName()
    {
        return $this->name;
    }
}
`,
			expectedConfidence: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signal := framework.Signal{
				Type:    framework.SignalFileContent,
				Value:   tt.content,
				Context: []byte(tt.content),
			}

			result := matcher.Match(ctx, signal)

			if result.Confidence != tt.expectedConfidence {
				t.Errorf("expected Confidence=%d, got %d", tt.expectedConfidence, result.Confidence)
			}
		})
	}
}

func TestPHPUnitParser_Parse(t *testing.T) {
	p := &PHPUnitParser{}
	ctx := context.Background()

	t.Run("test* method naming convention", func(t *testing.T) {
		source := `<?php
use PHPUnit\Framework\TestCase;

class UserTest extends TestCase
{
    public function testUserCanBeCreated()
    {
        $this->assertTrue(true);
    }

    public function testUserCanLogin()
    {
        $this->assertTrue(true);
    }

    public function helperMethod()
    {
        // not a test
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "UserTest.php")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if testFile.Path != "UserTest.php" {
			t.Errorf("expected Path='UserTest.php', got '%s'", testFile.Path)
		}
		if testFile.Framework != "phpunit" {
			t.Errorf("expected Framework='phpunit', got '%s'", testFile.Framework)
		}
		if testFile.Language != domain.LanguagePHP {
			t.Errorf("expected Language=php, got '%s'", testFile.Language)
		}
		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if suite.Name != "UserTest" {
			t.Errorf("expected Suite.Name='UserTest', got '%s'", suite.Name)
		}
		if len(suite.Tests) != 2 {
			t.Fatalf("expected 2 Tests in suite, got %d", len(suite.Tests))
		}
		if suite.Tests[0].Name != "testUserCanBeCreated" {
			t.Errorf("expected Tests[0].Name='testUserCanBeCreated', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[1].Name != "testUserCanLogin" {
			t.Errorf("expected Tests[1].Name='testUserCanLogin', got '%s'", suite.Tests[1].Name)
		}
	})

	t.Run("@test annotation in docblock", func(t *testing.T) {
		source := `<?php
use PHPUnit\Framework\TestCase;

class AnnotationTest extends TestCase
{
    /**
     * @test
     */
    public function it_creates_a_user()
    {
    }

    /**
     * @test
     */
    public function it_deletes_a_user()
    {
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "AnnotationTest.php")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if len(suite.Tests) != 2 {
			t.Fatalf("expected 2 Tests, got %d", len(suite.Tests))
		}

		if suite.Tests[0].Name != "it_creates_a_user" {
			t.Errorf("expected Tests[0].Name='it_creates_a_user', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[1].Name != "it_deletes_a_user" {
			t.Errorf("expected Tests[1].Name='it_deletes_a_user', got '%s'", suite.Tests[1].Name)
		}
	})

	t.Run("#[Test] attribute (PHP 8+)", func(t *testing.T) {
		source := `<?php
use PHPUnit\Framework\TestCase;

class AttributeTest extends TestCase
{
    #[Test]
    public function userCreation()
    {
    }

    #[Test]
    public function userDeletion()
    {
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "AttributeTest.php")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if len(suite.Tests) != 2 {
			t.Fatalf("expected 2 Tests, got %d", len(suite.Tests))
		}

		if suite.Tests[0].Name != "userCreation" {
			t.Errorf("expected Tests[0].Name='userCreation', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[1].Name != "userDeletion" {
			t.Errorf("expected Tests[1].Name='userDeletion', got '%s'", suite.Tests[1].Name)
		}
	})

	t.Run("mixed test detection methods", func(t *testing.T) {
		source := `<?php
use PHPUnit\Framework\TestCase;

class MixedTest extends TestCase
{
    public function testByConvention()
    {
    }

    /**
     * @test
     */
    public function by_annotation()
    {
    }

    #[Test]
    public function byAttribute()
    {
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "MixedTest.php")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if len(suite.Tests) != 3 {
			t.Fatalf("expected 3 Tests, got %d", len(suite.Tests))
		}
	})

	t.Run("class not extending TestCase is ignored", func(t *testing.T) {
		source := `<?php
class NotATest
{
    public function testSomething()
    {
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "NotATest.php")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 0 {
			t.Errorf("expected 0 Suites for non-TestCase class, got %d", len(testFile.Suites))
		}
	})

	t.Run("multiple classes in file", func(t *testing.T) {
		source := `<?php
use PHPUnit\Framework\TestCase;

class FirstTest extends TestCase
{
    public function testFirst()
    {
    }
}

class SecondTest extends TestCase
{
    public function testSecond()
    {
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "MultipleTests.php")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 2 {
			t.Fatalf("expected 2 Suites, got %d", len(testFile.Suites))
		}

		if testFile.Suites[0].Name != "FirstTest" {
			t.Errorf("expected Suites[0].Name='FirstTest', got '%s'", testFile.Suites[0].Name)
		}
		if testFile.Suites[1].Name != "SecondTest" {
			t.Errorf("expected Suites[1].Name='SecondTest', got '%s'", testFile.Suites[1].Name)
		}
	})

	t.Run("@dataProvider annotation", func(t *testing.T) {
		source := `<?php
use PHPUnit\Framework\TestCase;

class DataProviderTest extends TestCase
{
    /**
     * @dataProvider userProvider
     */
    public function testWithDataProvider($name)
    {
    }

    public function userProvider()
    {
        return [
            ['Alice'],
            ['Bob'],
        ];
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "DataProviderTest.php")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		// testWithDataProvider should be detected (has test prefix)
		// userProvider should NOT be detected (not a test)
		if len(suite.Tests) != 1 {
			t.Fatalf("expected 1 Test, got %d", len(suite.Tests))
		}

		if suite.Tests[0].Name != "testWithDataProvider" {
			t.Errorf("expected Tests[0].Name='testWithDataProvider', got '%s'", suite.Tests[0].Name)
		}
	})

	t.Run("empty test class", func(t *testing.T) {
		source := `<?php
use PHPUnit\Framework\TestCase;

class EmptyTest extends TestCase
{
}
`
		testFile, err := p.Parse(ctx, []byte(source), "EmptyTest.php")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 0 {
			t.Errorf("expected 0 Suites for empty test class, got %d", len(testFile.Suites))
		}
	})

	t.Run("#[Skip] attribute marks test as skipped", func(t *testing.T) {
		source := `<?php
use PHPUnit\Framework\TestCase;

class SkipTest extends TestCase
{
    #[Skip]
    public function testSkipped()
    {
    }

    public function testActive()
    {
    }
}
`
		testFile, err := p.Parse(ctx, []byte(source), "SkipTest.php")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(testFile.Suites) != 1 {
			t.Fatalf("expected 1 Suite, got %d", len(testFile.Suites))
		}

		suite := testFile.Suites[0]
		if len(suite.Tests) != 2 {
			t.Fatalf("expected 2 Tests, got %d", len(suite.Tests))
		}

		// First test should be skipped
		if suite.Tests[0].Name != "testSkipped" {
			t.Errorf("expected Tests[0].Name='testSkipped', got '%s'", suite.Tests[0].Name)
		}
		if suite.Tests[0].Status != domain.TestStatusSkipped {
			t.Errorf("expected Tests[0].Status=skipped, got '%s'", suite.Tests[0].Status)
		}
		if suite.Tests[0].Modifier != "#[Skip]" {
			t.Errorf("expected Tests[0].Modifier='#[Skip]', got '%s'", suite.Tests[0].Modifier)
		}

		// Second test should be active
		if suite.Tests[1].Name != "testActive" {
			t.Errorf("expected Tests[1].Name='testActive', got '%s'", suite.Tests[1].Name)
		}
		if suite.Tests[1].Status != domain.TestStatusActive {
			t.Errorf("expected Tests[1].Status=active, got '%s'", suite.Tests[1].Status)
		}
	})
}
