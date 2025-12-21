---
allowed-tools: Bash(*), Read, Write, Grep, Glob
description: Validate SpecVital parser accuracy by comparing with actual test framework CLI results
---

# Real-World Parser Validation Command

## Purpose

Validate that SpecVital Core parser accurately detects tests by comparing:

- **Ground Truth**: Actual test count from framework CLI (e.g., `vitest list` → 379 tests)
- **Parser Result**: Our parser's detected test count (e.g., 350 tests)
- **Delta**: If mismatch exists → parser bug found

This is a **quality assurance tool** for the Core parser engine.

## User Input

```text
$ARGUMENTS
```

**Interpretation**:

- **Empty**: Auto-select a popular open-source repo (NOT in repos.yaml)
- **Provided**: Natural language - parse intent and find appropriate repo

**Examples**:

| User Input                         | Interpretation                      |
| ---------------------------------- | ----------------------------------- |
| (empty)                            | Pick popular repo not in repos.yaml |
| `axios`                            | Test axios repository               |
| `https://github.com/lodash/lodash` | Test lodash                         |
| `Python project with pytest`       | Find Python repo using pytest       |
| `Something with Vitest`            | Find repo using Vitest              |

---

## Workflow

### Phase 1: Repository Selection

**1.1 Read repos.yaml to get exclusion list**:

```bash
grep "url:" tests/integration/repos.yaml
```

**1.2 Select repository**:

- If empty input: Pick well-known repo NOT in repos.yaml
- If input provided: Parse intent and resolve to GitHub URL

**1.3 Detect expected framework**:

- Check package.json for JS/TS repos
- Check pyproject.toml/setup.py for Python
- Check go.mod for Go
- Infer test framework from dependencies

### Phase 2: Clone Repository

```bash
git clone --depth 1 {url} /tmp/specvital-validate-{repo-name}
cd /tmp/specvital-validate-{repo-name}
```

### Phase 3: Get Ground Truth (Actual Test Count)

**Two strategies** (try in order):

#### Strategy A: CLI Execution (Primary)

Run the actual test framework CLI to get real test count.

**Step 1: Identify framework and version**

Check the repository's dependency file:

- `package.json` → devDependencies (jest, vitest, playwright, mocha, etc.)
- `pyproject.toml` / `setup.py` → pytest version
- `go.mod` → Go version

**Step 2: Query Context7 MCP for current CLI usage**

Use Context7 MCP to get up-to-date CLI documentation for the specific framework version:

```
1. mcp__context7__resolve-library-id → Get library ID (e.g., "/vitest/vitest")
2. mcp__context7__get-library-docs → Query "list tests CLI" or "collect tests"
```

This ensures:

- Correct CLI flags for the installed version
- Awareness of deprecated options
- Framework-specific quirks

**Step 3: Install dependencies and execute**

```bash
# Install project dependencies first
npm install / pip install -e . / go mod download

# Run the CLI command from Context7 docs
{framework-specific-command}
```

**Note**: Always verify CLI output format before counting

#### Strategy B: AI Manual Analysis (Fallback)

**When to use**: CLI fails due to:

- Missing environment variables (DATABASE_URL, API_KEY, etc.)
- Database/external service requirements
- Complex build setup failures
- Unsupported runtime in devcontainer

**How to execute**:

1. **Glob test files**: Find all test files using framework patterns

   ```bash
   # Examples
   **/*.test.{js,ts,jsx,tsx}  # Jest/Vitest
   **/*_test.py               # pytest
   **/*_test.go               # Go
   ```

2. **Read each file**: Analyze test structure manually

3. **Count tests**: Identify test functions/blocks by pattern
   - JavaScript: `it()`, `test()`, `it.each()`, `test.each()`
   - Python: `def test_*`, `@pytest.mark.parametrize`
   - Go: `func Test*`, `func Benchmark*`

4. **Handle edge cases**:
   - Nested describes → count leaf tests only
   - Parameterized tests → count as 1 (matches parser behavior)
   - Skipped tests → still count them

**Important**: Document which strategy was used in the report

### Phase 4: Run SpecVital Parser

```bash
cd /workspaces/specvital-core
just scan /tmp/specvital-validate-{repo-name}
```

### Phase 5: Compare Results

**Calculate accuracy**:

```
Ground Truth: 379 tests (from vitest list)
Parser Result: 350 tests
Delta: -29 tests (7.7% under-detection)
```

**Interpret delta**:

| Delta | Status  | Meaning               |
| ----- | ------- | --------------------- |
| 0     | ✅ PASS | Parser is accurate    |
| ≠ 0   | ❌ FAIL | Parser bug, needs fix |

**Important**: Even 1 test difference means a bug exists. No tolerance.

### Phase 6: Investigate Discrepancies

If delta ≠ 0:

1. **Sample ground truth files**: Pick 5 test files from CLI output
2. **Check parser output**: Are they detected? Correct test count?
3. **Identify patterns**: What's being missed?
   - Dynamic test names?
   - Unusual file patterns?
   - Nested describes?
   - Test.each/parameterized tests?
4. **Document the bug**: Create actionable issue for parser fix

### Phase 7: Generate Report

Create comprehensive validation report:

- **Language**: Korean
- **Location**: `/workspaces/specvital-core/realworld-test-report.md` (single file, overwrite)
- **Format**: Use the report template below

### Phase 8: Cleanup

```bash
rm -rf /tmp/specvital-validate-{repo-name}
```

---

## Validation Criteria

### ✅ PASS

- Delta = 0 (exact match)
- Framework correctly detected
- No parse errors

### ❌ FAIL

- Delta ≠ 0 (any difference = bug)
- Wrong framework detected
- Parser crash or timeout
- Missing test patterns

---

## Report Template

```markdown
# Parser Validation Report: {repo-name}

**Date**: {timestamp}
**Repository**: [{owner}/{repo}]({url})
**Framework**: {framework}

---

## 📊 Comparison Results

| Source                           | Test Count           |
| -------------------------------- | -------------------- |
| **Ground Truth** ({cli-command}) | {n}                  |
| **SpecVital Parser**             | {n}                  |
| **Delta**                        | {±n} ({percentage}%) |

**Status**: {PASS|FAIL}

---

## 🔍 Ground Truth Details

**Method**: {CLI Execution | AI Manual Analysis}

**Command/Approach used**:

\`\`\`bash
{command or "Manual file analysis via AI"}
\`\`\`

**Output sample**:

\`\`\`
{first 10 lines of output or file analysis summary}
\`\`\`

**Note** (if AI analysis):

> CLI failed because: {reason}
> Analyzed {n} test files manually

---

## 📈 Parser Results

| Metric         | Value  |
| -------------- | ------ |
| Files Scanned  | {n}    |
| Files Matched  | {n}    |
| Tests Detected | {n}    |
| Duration       | {time} |

### Framework Distribution

| Framework | Files | Tests |
| --------- | ----- | ----- |
| {name}    | {n}   | {n}   |

---

## 🐛 Discrepancy Analysis

{IF delta ≠ 0}

### Missing Tests Investigation

**Sample comparison**:

| File     | Ground Truth | Parser | Delta |
| -------- | ------------ | ------ | ----- |
| `{path}` | {n}          | {n}    | {±n}  |

**Patterns identified**:

- {pattern 1}: {description}
- {pattern 2}: {description}

**Root cause hypothesis**:
{explanation of why tests are missing}

{ELSE}
No significant discrepancies found.
{ENDIF}

---

## 📋 Conclusion

{Based on status}

**If PASS**:

> Parser accurately detects tests in this repository (exact match).
> Ready to add to `tests/integration/repos.yaml`:
>
> \`\`\`yaml
>
> - name: {repo-name}
>   url: {url}
>   ref: {ref}
>   framework: {framework}
>   \`\`\`

**If FAIL**:

> Parser has accuracy issues. Do not add to repos.yaml until fixed.
>
> **Issues to fix**:
>
> - {issue 1}
> - {issue 2}

---

## 📝 Next Steps

- [ ] {action based on findings}
```

---

## Key Rules

### ✅ Must Do

- Always get ground truth from actual test CLI
- Install dependencies (`npm install`, `pip install`, etc.) for CLI to work
- Compare at both file-level and test-level
- Investigate any delta ≠ 0
- Document root cause of discrepancies

### ❌ Must Not Do

- Skip ground truth collection
- Assume parser is correct without verification
- Ignore any discrepancy (even 1 = bug)
- Test repos already in repos.yaml (waste of time)

### 🎯 Principles

1. **Ground Truth First**: Real CLI results are the source of truth
2. **Quantitative**: Measure exact delta, not "looks right"
3. **Diagnostic**: Explain WHY discrepancies exist
4. **Actionable**: Clear next steps for fixes

---

## Execution

Now execute the parser validation according to the guidelines above.

```

```
