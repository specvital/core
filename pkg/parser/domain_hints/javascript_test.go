package domain_hints

import (
	"context"
	"testing"

	"github.com/specvital/core/pkg/domain"
)

func TestJavaScriptExtractor_Extract_ES6Imports(t *testing.T) {
	source := []byte(`
import { test, expect } from '@playwright/test';
import axios from 'axios';
import * as lodash from 'lodash';
import '@testing-library/jest-dom';
import type { User } from './types';

test('should work', async () => {
  const mockUser = { name: 'test' };
  authService.validateToken();
});
`)

	extractor := &JavaScriptExtractor{lang: domain.LanguageTypeScript}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	t.Run("imports", func(t *testing.T) {
		expectedImports := map[string]bool{
			"@playwright/test":          true,
			"axios":                     true,
			"lodash":                    true,
			"@testing-library/jest-dom": true,
		}

		// type-only import should be excluded
		excludedImports := []string{"./types"}

		importSet := make(map[string]bool)
		for _, imp := range hints.Imports {
			importSet[imp] = true
		}

		for imp := range expectedImports {
			if !importSet[imp] {
				t.Errorf("expected import %q to be included", imp)
			}
		}

		for _, imp := range excludedImports {
			if importSet[imp] {
				t.Errorf("expected type-only import %q to be excluded", imp)
			}
		}
	})
}

func TestJavaScriptExtractor_Extract_CommonJS(t *testing.T) {
	source := []byte(`
const lodash = require('lodash');
const { get } = require('axios');
const path = require('path');

test('should work', async () => {
  const mockData = getData();
});
`)

	extractor := &JavaScriptExtractor{lang: domain.LanguageJavaScript}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedImports := map[string]bool{
		"lodash": true,
		"axios":  true,
		"path":   true,
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

func TestJavaScriptExtractor_Extract_Calls(t *testing.T) {
	source := []byte(`
import { test, expect } from '@playwright/test';

test('should work', async () => {
  authService.validateToken('token');
  userRepo.findById(1);
  const result = orderService.create(order);
  doSomething();
});
`)

	extractor := &JavaScriptExtractor{lang: domain.LanguageTypeScript}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	expectedCalls := map[string]bool{
		"authService.validateToken": true,
		"userRepo.findById":         true,
		"orderService.create":       true,
		"doSomething":               true,
	}

	// Test framework calls should be excluded
	excludedCalls := []string{"test", "expect"}

	callSet := make(map[string]bool)
	for _, call := range hints.Calls {
		callSet[call] = true
	}

	for call := range expectedCalls {
		if !callSet[call] {
			t.Errorf("expected call %q to be included, got %v", call, hints.Calls)
		}
	}

	for _, call := range excludedCalls {
		if callSet[call] {
			t.Errorf("expected test framework call %q to be excluded", call)
		}
	}
}

func TestJavaScriptExtractor_Extract_Variables(t *testing.T) {
	source := []byte(`
import { test } from '@playwright/test';

test('should work', async () => {
  const mockUser = { name: 'test' };
  let fakeClient = new Client();
  var stubRepo = createStubRepo();
  const testData = getData();
  const expectedResult = 'ok';
  let wantValue = 42;
  var gotResponse = await fetch();
  const fixtureOrder = Order.create();
  const regularVar = 'ignored';
  const count = 10;
});
`)

	extractor := &JavaScriptExtractor{lang: domain.LanguageTypeScript}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	shouldMatch := []string{
		"mockUser", "fakeClient", "stubRepo", "testData",
		"expectedResult", "wantValue", "gotResponse", "fixtureOrder",
	}
	shouldNotMatch := []string{"regularVar", "count"}

	varSet := make(map[string]bool)
	for _, v := range hints.Variables {
		varSet[v] = true
	}

	for _, v := range shouldMatch {
		if !varSet[v] {
			t.Errorf("expected %q to be included in variables, got %v", v, hints.Variables)
		}
	}

	for _, v := range shouldNotMatch {
		if varSet[v] {
			t.Errorf("expected %q to be excluded from variables", v)
		}
	}
}

func TestJavaScriptExtractor_Extract_EmptyFile(t *testing.T) {
	source := []byte(`// empty file`)

	extractor := &JavaScriptExtractor{lang: domain.LanguageJavaScript}
	hints := extractor.Extract(context.Background(), source)

	if hints != nil {
		t.Errorf("expected nil for empty file, got %+v", hints)
	}
}

func TestJavaScriptExtractor_Extract_MixedImports(t *testing.T) {
	source := []byte(`
import { test } from '@playwright/test';
const axios = require('axios');
import type { Response } from 'express';

test('mixed imports', async () => {
  const mockResponse = {};
});
`)

	extractor := &JavaScriptExtractor{lang: domain.LanguageTypeScript}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	importSet := make(map[string]bool)
	for _, imp := range hints.Imports {
		importSet[imp] = true
	}

	// Both ES6 and CommonJS should be captured
	if !importSet["@playwright/test"] {
		t.Error("expected @playwright/test import")
	}
	if !importSet["axios"] {
		t.Error("expected axios import (CommonJS)")
	}

	// Type-only imports should be excluded
	if importSet["express"] {
		t.Error("expected type-only express import to be excluded")
	}
}

func TestJavaScriptExtractor_Extract_PlaywrightFile(t *testing.T) {
	source := []byte(`
import { test, expect } from '@playwright/test';
import { LoginPage } from './pages/login';

test.describe('authentication flow', () => {
  const mockCredentials = { email: 'test@example.com', password: 'secret' };

  test('should login successfully', async ({ page }) => {
    const loginPage = new LoginPage(page);
    await loginPage.goto();
    await authService.login(mockCredentials);
    await expect(page).toHaveURL('/dashboard');
  });
});
`)

	extractor := &JavaScriptExtractor{lang: domain.LanguageTypeScript}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Verify imports include both library and local imports
	importSet := make(map[string]bool)
	for _, imp := range hints.Imports {
		importSet[imp] = true
	}

	if !importSet["@playwright/test"] {
		t.Error("expected @playwright/test import")
	}
	if !importSet["./pages/login"] {
		t.Error("expected ./pages/login import")
	}

	// Verify domain-relevant variables
	varSet := make(map[string]bool)
	for _, v := range hints.Variables {
		varSet[v] = true
	}

	if !varSet["mockCredentials"] {
		t.Error("expected mockCredentials variable")
	}

	// Verify calls (excluding test framework)
	callSet := make(map[string]bool)
	for _, c := range hints.Calls {
		callSet[c] = true
	}

	if !callSet["authService.login"] {
		t.Errorf("expected authService.login call, got %v", hints.Calls)
	}
}

func TestJavaScriptExtractor_Extract_TSX(t *testing.T) {
	source := []byte(`
import React from 'react';
import { render, screen } from '@testing-library/react';
import { UserProfile } from './UserProfile';

test('should render user profile', () => {
  const mockUser = { id: 1, name: 'John' };
  render(<UserProfile user={mockUser} />);
  userService.getProfile(mockUser.id);
  expect(screen.getByText('John')).toBeInTheDocument();
});
`)

	extractor := &JavaScriptExtractor{lang: domain.LanguageTSX}
	hints := extractor.Extract(context.Background(), source)

	if hints == nil {
		t.Fatal("expected hints, got nil")
	}

	// Verify imports
	importSet := make(map[string]bool)
	for _, imp := range hints.Imports {
		importSet[imp] = true
	}

	if !importSet["react"] {
		t.Error("expected react import")
	}
	if !importSet["@testing-library/react"] {
		t.Error("expected @testing-library/react import")
	}
	if !importSet["./UserProfile"] {
		t.Error("expected ./UserProfile import")
	}

	// Verify variables
	varSet := make(map[string]bool)
	for _, v := range hints.Variables {
		varSet[v] = true
	}

	if !varSet["mockUser"] {
		t.Errorf("expected mockUser variable, got %v", hints.Variables)
	}

	// Verify calls
	callSet := make(map[string]bool)
	for _, c := range hints.Calls {
		callSet[c] = true
	}

	if !callSet["userService.getProfile"] {
		t.Errorf("expected userService.getProfile call, got %v", hints.Calls)
	}
}
