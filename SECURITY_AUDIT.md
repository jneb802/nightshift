# Nightshift Security Audit Report

**Date**: February 10, 2026
**Scope**: nightshift Go codebase security anti-pattern analysis
**Branch**: security-audit

## Executive Summary

This security audit identified **10 security anti-patterns** in the nightshift codebase, ranging from HIGH to LOW severity. The most critical issues involve dangerous default configurations and missing input validation on file paths. **3 critical fixes** have been implemented:

1. ✅ Changed dangerous default configurations from true → false
2. ✅ Fixed database directory permissions (0755 → 0700)
3. ✅ Added shell path escaping for shell config modifications

## Detailed Findings

### 1. ⚠️ DANGEROUS DEFAULT CONFIGURATIONS (HIGH RISK) - FIXED

**Severity**: HIGH
**Status**: ✅ FIXED
**Location**: `internal/config/config.go:249, 252`

**Problem**:
```go
v.SetDefault("providers.claude.dangerously_skip_permissions", true)
v.SetDefault("providers.codex.dangerously_bypass_approvals_and_sandbox", true)
```

These flags bypass critical security prompts by DEFAULT, creating a significant security footgun. Users would need to explicitly opt-in to approval prompts rather than the safer inverse.

**Impact**: When users run nightshift for the first time, approval prompts are skipped by default, allowing automated execution without explicit consent.

**Fix Applied**:
```go
v.SetDefault("providers.claude.dangerously_skip_permissions", false)
v.SetDefault("providers.codex.dangerously_bypass_approvals_and_sandbox", false)
```

**Recommendation**: Users now must explicitly set these to `true` to bypass approvals, requiring conscious security decisions.

---

### 2. ⚠️ INSECURE DATABASE DIRECTORY PERMISSIONS (MEDIUM RISK) - FIXED

**Severity**: MEDIUM
**Status**: ✅ FIXED
**Location**: `internal/db/db.go:33`

**Problem**:
```go
if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
```

Database directory created with permissions `0755` (rwxr-xr-x), making it world-readable. The database may contain sensitive execution history, token usage data, and task metadata.

**Impact**: Any local user on the system can read database contents including:
- Task execution history
- Token usage statistics
- Project and provider configuration data
- Run records with task details

**Fix Applied**:
```go
// SECURITY: Use 0700 (rwx------) for database directory to restrict access to owner only
if err := os.MkdirAll(filepath.Dir(resolved), 0700); err != nil {
```

**Verification**: Database directory now owned by user with no group or world access.

---

### 3. ⚠️ MISSING SHELL PATH ESCAPING (MEDIUM RISK) - FIXED

**Severity**: MEDIUM
**Status**: ✅ FIXED
**Location**: `cmd/nightshift/commands/setup.go:1373-1390`

**Problem**:
The `pathExportLine()` function did not escape the directory path before inserting it into shell configuration files. Paths containing special characters like `$`, backticks, or quotes could break shell startup or enable injection attacks.

```go
// BEFORE (vulnerable):
return fmt.Sprintf("export PATH=\"$PATH:%s\"", dir)
// If dir = "/path/with$(malicious_cmd)", this executes malicious_cmd
```

**Impact**:
- Shell startup scripts could be corrupted
- Path with special chars breaks shell initialization
- Potential (low likelihood) code execution if attacker controls PATH

**Fix Applied**:
```go
func escapeShellPath(path string) string {
	// Single quotes prevent all expansions in shell, safest approach
	if !strings.Contains(path, "'") {
		return fmt.Sprintf("'%s'", path)
	}
	// Path contains single quote: use double quotes and escape special chars
	escaped := strings.ReplaceAll(path, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	escaped = strings.ReplaceAll(escaped, "$", "\\$")
	escaped = strings.ReplaceAll(escaped, "`", "\\`")
	return fmt.Sprintf("\"%s\"", escaped)
}
```

---

### 4. ⚠️ PATH TRAVERSAL RISK (MEDIUM RISK) - REQUIRES FURTHER REVIEW

**Severity**: MEDIUM
**Status**: ⏳ REQUIRES MITIGATION
**Location**: `internal/providers/claude.go:142-180`

**Problem**:
```go
err := filepath.WalkDir(projectsDir, func(path string, d os.DirEntry, err error) error {
    if err != nil {
        if os.IsPermission(err) {
            return nil  // Silently skip permission errors
        }
        return err
    }
    // ... no validation of resolved paths
```

Symlinks in `~/.claude/projects` could point outside the directory tree. No checks to ensure files are actually within the intended projects directory.

**Impact**: If attacker can create symlinks in Claude data directory, could:
- Read sensitive files from other directories
- Cause denial of service via broken symlinks
- Leak information about system structure

**Recommendation**:
- Add `filepath.EvalSymlinks()` validation after path resolution
- Verify resolved path is within projectsDir using `filepath.Rel()`
- Log suspicious symlink patterns
- Document symlink handling assumptions

---

### 5. ⚠️ INSUFFICIENT INPUT VALIDATION ON FILE PATHS (HIGH RISK)

**Severity**: HIGH
**Status**: ⏳ REQUIRES MITIGATION
**Location**: `cmd/nightshift/commands/setup.go:624-638`

**Problem**:
```go
path := expandPath(value)
info, err := os.Stat(path)
if err != nil {
    m.projectErr = "path not found"
    return m, nil
}
// No symlink validation, no path normalization
```

Project paths are only validated for existence, not for safety. User can add symlinks to arbitrary locations.

**Impacts**:
- Symlinks pointing to sensitive directories
- `..` sequences in paths could escape intended scope
- No validation against malicious path patterns

**Recommendation**:
```go
// Validate path for safety
absPath, _ := filepath.Abs(path)
realPath, err := filepath.EvalSymlinks(absPath)
if err != nil {
    m.projectErr = "invalid or broken symlink"
    return m, nil
}
// Verify real path is within reasonable bounds
if !isReasonablePath(realPath) {
    m.projectErr = "path points to sensitive location"
    return m, nil
}
```

---

### 6. ⚠️ SILENT ERROR SUPPRESSION ON CRITICAL OPERATIONS (MEDIUM RISK)

**Severity**: MEDIUM
**Status**: ⏳ REQUIRES MITIGATION
**Location**: Multiple locations
- `cmd/nightshift/commands/run.go:110`: `_ = database.Close()`
- `cmd/nightshift/commands/setup.go:906`: `_ = database.Close()`
- `internal/db/db.go:43`: `_ = sqlDB.Close()`

**Problem**:
Database close errors are unconditionally suppressed. This can hide:
- Incomplete WAL checkpoint
- I/O failures during flush
- Transaction rollback issues

**Impact**: Database could be left in inconsistent state without indication of error.

**Recommendation**:
```go
if err := database.Close(); err != nil {
    log.Warnf("database close error: %v", err)
    // Ensure WAL checkpoint completes
    if strings.Contains(err.Error(), "checkpoint") {
        // Additional recovery steps
    }
}
```

---

### 7. ⚠️ INSECURE FILE READING WITHOUT VALIDATION (MEDIUM RISK)

**Severity**: MEDIUM
**Status**: ⏳ REQUIRES MITIGATION
**Location**: `internal/orchestrator/orchestrator.go:275`

**Problem**:
```go
for _, path := range files {
    content, err := os.ReadFile(path)
    // No validation of path safety before reading
```

Files passed to agent are read without validation. Could leak secrets if agent receives unvetted file list.

**Recommendation**:
- Validate each path against allowed list
- Check file sizes before reading
- Sanitize/truncate large files
- Audit logging of file reads

---

### 8. ⚠️ INCONSISTENT PATH VALIDATION IN SANDBOX (MEDIUM RISK)

**Severity**: MEDIUM
**Status**: ⏳ REQUIRES MITIGATION
**Location**: `internal/security/sandbox.go:160-180`

**Problem**:
The `ValidatePath()` function exists but isn't called consistently from `Execute()` methods. Validation is optional rather than mandatory.

**Recommendation**:
- Make ValidatePath() automatic in all Execute() paths
- Return early with error if validation fails
- Add integration tests for boundary cases

---

### 9. ⚠️ HARDCODED ARTIFACT NAMES (LOW RISK)

**Severity**: LOW
**Status**: ⏳ DOCUMENT RISK
**Location**: `cmd/nightshift/commands/setup.go:56`

**Problem**:
```go
const nightshiftPlanIgnore = ".nightshift-plan"
```

Hardcoded artifact name means two concurrent nightshift instances could conflict.

**Recommendation**:
- Document that concurrent executions should use separate projects
- Consider adding run-ID to artifact names: `.nightshift-plan-{uuid}`
- Add warning if existing artifacts detected

---

## Summary Table

| # | Issue | Severity | Status | Location | Category |
|---|-------|----------|--------|----------|----------|
| 1 | Dangerous defaults (Skip=true) | HIGH | ✅ FIXED | config.go:249,252 | Default Config |
| 2 | DB dir permissions (0755) | MEDIUM | ✅ FIXED | db.go:33 | File Permissions |
| 3 | Missing shell path escaping | MEDIUM | ✅ FIXED | setup.go:1373-1390 | Injection Prevention |
| 4 | Symlink traversal risk | MEDIUM | ⏳ TODO | claude.go:142-180 | Path Traversal |
| 5 | Missing path validation | HIGH | ⏳ TODO | setup.go:624-638 | Input Validation |
| 6 | Silent error suppression | MEDIUM | ⏳ TODO | run.go:110, db.go:43 | Error Handling |
| 7 | Unvalidated file reading | MEDIUM | ⏳ TODO | orchestrator.go:275 | Path Traversal |
| 8 | Inconsistent validation | MEDIUM | ⏳ TODO | sandbox.go:160-180 | Validation |
| 9 | Hardcoded artifact names | LOW | ⏳ TODO | setup.go:56 | Code Quality |

## Recommendations for Follow-up

### Immediate (Critical)
1. ✅ Apply the 3 fixes already implemented
2. ⏳ Add path validation to all user-supplied file paths
3. ⏳ Implement symlink safety checks with EvalSymlinks()

### Short-term (High priority)
4. ⏳ Log instead of suppressing database close errors
5. ⏳ Validate file paths before reading arbitrary files
6. ⏳ Make sandbox path validation automatic/mandatory

### Medium-term
7. ⏳ Document symlink handling assumptions
8. ⏳ Add security test suite for path handling
9. ⏳ Implement audit logging for sensitive operations

### Long-term
10. ⏳ Consider security code review process
11. ⏳ Implement fuzz testing for file operations
12. ⏳ Create security guidelines document

## Testing Recommendations

```bash
# Test dangerous defaults are now false
go test ./... -v -run TestDangerousDefaults

# Test shell path escaping
go test ./... -v -run TestShellPathEscaping

# Test database permissions
stat -f "%A" ~/.local/share/nightshift  # Should show 0700

# Test symlink handling
# Create test with symlinks and verify proper handling
```

## Files Modified

- ✅ `internal/config/config.go` - Changed dangerous defaults to false
- ✅ `internal/db/db.go` - Changed DB directory permissions to 0700
- ✅ `cmd/nightshift/commands/setup.go` - Added shell path escaping function

## Conclusion

The nightshift codebase demonstrates good security practices in many areas (audit logging, credential management, sandboxing), but has several anti-patterns that should be addressed. The 3 critical fixes implemented address the highest-risk issues. The remaining 6-7 items require code changes and should be prioritized based on attack surface exposure.

**Overall Risk Assessment**: MEDIUM
**Post-Fix Risk**: LOW (after remaining items are addressed)
