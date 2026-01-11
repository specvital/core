---
allowed-tools: Bash(*), Read, Write, Grep, Glob
description: Validate DomainHints extraction quality by detecting noise patterns and statistical anomalies
---

# Domain Hints Quality Validation Command

## Purpose

Validate that DomainHints extraction produces clean, domain-relevant data by detecting:

- **Noise Patterns**: Empty strings, parser artifacts (`[.`), meaningless tokens (`fn`, `Ok`)
- **Statistical Anomalies**: Unusual NULL ratios, abnormal average counts
- **Regressions**: Unexpected changes in extraction results

This is a **data quality assurance tool** for the DomainHints extraction engine.

## User Input

```text
$ARGUMENTS
```

**Interpretation**:

| Input                  | Action                                       |
| ---------------------- | -------------------------------------------- |
| (empty)                | Run full validation on all integration repos |
| `{repo-name}`          | Validate specific repo from repos.yaml       |
| `{github-url}`         | Clone and validate external repo             |
| `noise` / `patterns`   | Focus on noise pattern detection only        |
| `stats` / `statistics` | Focus on statistical analysis only           |
| `compare {framework}`  | Compare framework stats vs baseline          |

---

## Known Noise Patterns

These patterns indicate extraction bugs or low-quality data:

### Critical (Must Filter)

| Pattern | Type            | Description                   | Severity |
| ------- | --------------- | ----------------------------- | -------- |
| `""`    | Empty           | Empty string in Calls/Imports | 🔴 Error |
| `[.`    | Parser artifact | Spread array handling bug     | 🔴 Error |
| `.`     | Single char     | Dot only                      | 🔴 Error |

### Warning (Should Filter)

| Pattern         | Type        | Description          | Severity   |
| --------------- | ----------- | -------------------- | ---------- |
| `fn`            | Meaningless | Standalone fn() call | 🟡 Warning |
| `Ok`            | Rust stdlib | Enum constructor     | 🟡 Warning |
| `Err`           | Rust stdlib | Enum constructor     | 🟡 Warning |
| `Some`          | Rust stdlib | Enum constructor     | 🟡 Warning |
| `None`          | Rust stdlib | Enum constructor     | 🟡 Warning |
| 1-2 char tokens | Too short   | Likely noise         | 🟡 Warning |

### Regex Patterns

```go
// Parser artifacts
"^[\\[\\]\\(\\)\\{\\}]+$"  // Bracket-only tokens

// Language keywords mistakenly captured
"^(fn|if|for|let|var|const)$"
```

---

## Statistical Baselines

Expected values by framework (from production data):

| Framework  | NULL Ratio | Avg Imports | Avg Calls | Notes                     |
| ---------- | ---------- | ----------- | --------- | ------------------------- |
| cypress    | ~21%       | 2-5         | 5-15      | E2E tests have no imports |
| jest       | ~5%        | 3-10        | 10-20     |                           |
| vitest     | ~7%        | 3-10        | 10-20     |                           |
| go-testing | ~2%        | 3-8         | 8-15      |                           |
| playwright | ~3%        | 2-6         | 10-25     |                           |
| cargo-test | ~5%        | 2-6         | 5-15      |                           |

**Anomaly Thresholds**:

- NULL ratio increase > 5%: 🔴 Error
- Avg imports/calls drop > 20%: 🟡 Warning

---

## Workflow

### Phase 1: Setup

**1.1 Determine validation scope**:

```bash
# Check what repos are available
cat tests/integration/repos.yaml | grep "name:"
```

**1.2 Select target(s)**:

- Empty input → all repos in repos.yaml
- Specific repo → filter to that repo
- External URL → clone to /tmp

### Phase 2: Run Parser with DomainHints

```bash
cd /workspaces/specvital-core

# For integration test repos (already cached)
just scan /path/to/cached/repo --json 2>/dev/null | jq '.'

# Or run integration test to get data
go test -tags integration ./tests/integration/... -v -run "TestScan/{repo-name}" 2>&1
```

### Phase 3: Extract DomainHints Data

**3.1 Collect all DomainHints from scan result**:

Parse the JSON output to extract:

- All `Imports` arrays
- All `Calls` arrays
- Count of NULL vs non-NULL DomainHints

**3.2 Calculate statistics**:

```
Total files: N
Files with hints: M
NULL ratio: (N-M)/N * 100%
Avg imports per file: sum(imports) / M
Avg calls per file: sum(calls) / M
```

### Phase 4: Noise Pattern Detection

**4.1 Check all Imports for noise patterns**:

```bash
# Pseudo-code
for import in all_imports:
    if import == "":
        report_error("Empty import found")
    if len(import) <= 2:
        report_warning(f"Short import: {import}")
```

**4.2 Check all Calls for noise patterns**:

```bash
# Pseudo-code
for call in all_calls:
    if call == "":
        report_error("Empty call found")
    if call in ["[.", ".", "fn", "Ok", "Err", "Some", "None"]:
        report_warning(f"Noise pattern: {call}")
    if matches_regex(call, "^[\\[\\]\\(\\)\\{\\}]+$"):
        report_error(f"Parser artifact: {call}")
```

### Phase 5: Statistical Analysis

**5.1 Compare against baselines**:

```
Framework: {framework}
Expected NULL ratio: {baseline}%
Actual NULL ratio: {actual}%
Delta: {delta}%
Status: {PASS|WARN|FAIL}
```

**5.2 Detect anomalies**:

- NULL ratio significantly higher than baseline
- Average imports/calls significantly lower
- Unusual distribution patterns

### Phase 6: Generate Report

**Report Location**: `/workspaces/specvital-core/domain-hints-quality-report.md`

**Language**: 🇰🇷 Korean (한국어) - MANDATORY

---

## Report Template

```markdown
# DomainHints 품질 검증 보고서

**일시**: {timestamp}
**대상**: {repo-name or "전체 통합 테스트 저장소"}
**프레임워크**: {framework(s)}

---

## 📊 요약

| 항목             | 결과                          |
| ---------------- | ----------------------------- |
| 검사 파일 수     | {n}                           |
| DomainHints 존재 | {n} ({percentage}%)           |
| 노이즈 패턴 발견 | {n}건                         |
| 통계 이상치      | {n}건                         |
| **최종 상태**    | {✅ PASS / ⚠️ WARN / ❌ FAIL} |

---

## 🔍 노이즈 패턴 검사

### 발견된 노이즈

| 패턴        | 타입   | 발견 횟수 | 샘플 파일     |
| ----------- | ------ | --------- | ------------- |
| `{pattern}` | {type} | {count}   | `{file_path}` |

### 패턴별 상세

#### `{pattern}` ({count}건)

**출처 분석**:

- {repo/framework}: {count}건

**원인 추정**:
{description of likely cause}

---

## 📈 통계 분석

### 프레임워크별 현황

| 프레임워크  | 파일 수 | NULL 비율 | 기준선      | 상태     |
| ----------- | ------- | --------- | ----------- | -------- |
| {framework} | {n}     | {actual}% | {baseline}% | {status} |

### Imports 분포

| 지표            | 값  |
| --------------- | --- |
| 총 고유 imports | {n} |
| 파일당 평균     | {n} |
| 최대            | {n} |

**Top 10 Imports**:

| Import     | 출현 횟수 |
| ---------- | --------- |
| `{import}` | {count}   |

### Calls 분포

| 지표          | 값  |
| ------------- | --- |
| 총 고유 calls | {n} |
| 파일당 평균   | {n} |
| 최대          | {n} |

**Top 10 Calls**:

| Call     | 출현 횟수 |
| -------- | --------- |
| `{call}` | {count}   |

---

## 🐛 발견된 문제

### 🔴 Critical (즉시 수정 필요)

{IF critical issues exist}

1. **{issue}**: {description}
   - 영향 범위: {scope}
   - 권장 조치: {action}
     {ELSE}
     없음
     {ENDIF}

### 🟡 Warning (검토 필요)

{IF warnings exist}

1. **{issue}**: {description}
   - 영향 범위: {scope}
   - 권장 조치: {action}
     {ELSE}
     없음
     {ENDIF}

---

## 📋 결론

{Based on findings}

**If PASS**:

> DomainHints 추출 품질이 양호합니다.
>
> - 노이즈 패턴: 없음
> - 통계 이상치: 없음

**If WARN**:

> 경미한 품질 이슈가 발견되었습니다.
>
> - {list of warnings}
>   권장: 다음 릴리스에서 개선 검토

**If FAIL**:

> 심각한 품질 이슈가 발견되었습니다.
>
> - {list of critical issues}
>   **즉시 수정 필요**

---

## 📝 권장 조치

- [ ] {action item 1}
- [ ] {action item 2}
```

---

## Validation Criteria

### ✅ PASS

- No critical noise patterns found
- NULL ratio within baseline ± 5%
- No statistical anomalies

### ⚠️ WARN

- Warning-level noise patterns found (fn, Ok, etc.)
- NULL ratio slightly above baseline (5-10%)
- Minor statistical deviations

### ❌ FAIL

- Critical noise patterns found (empty string, [., etc.)
- NULL ratio significantly above baseline (>10%)
- Parser artifacts detected

---

## Key Rules

### ✅ Must Do

- **Write report in Korean (한국어로 리포트 작성)** ← CRITICAL
- Check ALL Imports and Calls for noise patterns
- Compare statistics against framework baselines
- Provide actionable recommendations
- Include sample file paths for each issue

### ❌ Must Not Do

- **Write report in English** ← Use Korean only
- Ignore any noise pattern (even 1 = potential bug)
- Skip statistical analysis
- Miss framework-specific baselines (e.g., Cypress 21% NULL is normal)

### 🎯 Principles

1. **Quality First**: Even minor noise degrades AI domain classification
2. **Quantitative**: Measure exact counts and percentages
3. **Comparative**: Always compare against baselines
4. **Actionable**: Every issue needs a clear fix recommendation

---

## Quick Commands

```bash
# Full validation on all repos
/validate-domain-hints

# Single repo
/validate-domain-hints grafana

# External repo
/validate-domain-hints https://github.com/vercel/next.js

# Focus on noise only
/validate-domain-hints noise

# Compare Cypress stats
/validate-domain-hints compare cypress
```

---

## Execution

Now execute the DomainHints quality validation according to the guidelines above.
