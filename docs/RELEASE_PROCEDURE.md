# 🚀 Release Procedure Guide

This document provides the correct procedure for creating GizTUI releases using the **automated GitHub workflow system**.

## 📋 Table of Contents

- [Overview](#-overview)
- [Pre-Release Checklist](#-pre-release-checklist)
- [Semantic Versioning Guidelines](#-semantic-versioning-guidelines)
- [Release Process](#-release-process)
- [Post-Release Tasks](#-post-release-tasks)
- [Troubleshooting](#-troubleshooting)

## 🎯 Overview

GizTUI uses **automated GitHub workflows** for releases. The process is:

1. **Prepare**: Update version files and changelog
2. **Trigger**: Tag and push to trigger automated workflow  
3. **Verify**: Confirm workflow success and test installation

**Important**: The workflow handles building, packaging, and publishing. Never bypass it with manual processes.

## ✅ Pre-Release Checklist

Before starting any release, ensure all these conditions are met:

### **Code Quality Gates**
- [ ] All features are complete and tested
- [ ] All tests pass: `make test`
- [ ] Linting passes: `make lint` 
- [ ] Code formatting is correct: `make fmt`
- [ ] No critical issues in `make vet`
- [ ] All pre-commit hooks pass

### **Architecture Compliance** 
- [ ] All new features follow service-first architecture
- [ ] Command parity implemented (every keyboard shortcut has `:command`)
- [ ] Proper error handling using `ErrorHandler` pattern
- [ ] Thread-safe accessor methods used throughout
- [ ] Theming implemented with `GetComponentColors()` pattern

### **Documentation Requirements**
- [ ] All new features documented in appropriate docs
- [ ] Breaking changes clearly identified
- [ ] Configuration changes documented
- [ ] **Shortcut/command doc-sync (three sources must agree)** — for every new key or command shipped this cycle, confirm it lives in **all three**:
  1. `docs/KEYBOARD_SHORTCUTS.md` (the reference table — this is the one that silently drifts)
  2. the in-app `?` help (built in `internal/tui/app.go`)
  3. the command completion registry (`internal/tui/command_completion.go`)
  - Quick audit of registered commands vs the reference doc:
    ```bash
    for n in $(grep -oE '\{name: "[a-z:-]+"' internal/tui/command_completion.go | sed -E 's/.*"([a-z:-]+)".*/\1/' | sort -u); do
      grep -qE ":$n\b|\`$n\`" docs/KEYBOARD_SHORTCUTS.md || echo "check docs for :$n"
    done
    ```
    (Aliases whose canonical form is documented are fine; use this to catch genuinely-missing entries.)
- [ ] Version consistency is enforced by `go test ./internal/version/` — `VERSION`, `internal/version/version.go`, `CHANGELOG.md`, and `packaging/homebrew/giztui-desktop.rb` must all agree (the test fails the build if any drifts)

### **Git Repository Status**
- [ ] Working directory is clean: `git status`
- [ ] All intended changes are committed
- [ ] Currently on `main` branch
- [ ] Local `main` is up to date with `origin/main`

## 📊 Semantic Versioning Guidelines

GizTUI follows [Semantic Versioning](https://semver.org/): `MAJOR.MINOR.PATCH`

### **MAJOR Version (X.0.0)**
Increment when making **breaking changes**:
- Configuration format changes
- Command syntax changes
- Removed features or options
- API compatibility breaks

### **MINOR Version (X.Y.0)**
Increment when adding **new features** (backward compatible):
- New keyboard shortcuts or commands
- New integrations (Slack, Obsidian, etc.)
- New AI features or providers
- New configuration options
- Enhanced existing functionality

### **PATCH Version (X.Y.Z)**
Increment for **bug fixes and improvements** (backward compatible):
- Bug fixes
- Performance improvements
- Code refactoring
- Documentation updates
- Dependency updates

## 🎯 Release Process

### **Phase 1: Version Planning**

1. **Determine Version Type**
   ```bash
   # Review changes since last release
   git log $(git describe --tags --abbrev=0)..HEAD --oneline
   ```

2. **Choose New Version Number**
   ```bash
   # Check current version
   make version
   cat VERSION
   
   # Decide: MAJOR.MINOR.PATCH
   # Examples:
   # Bug fixes: 1.1.0 → 1.1.1
   # New feature: 1.1.1 → 1.2.0  
   # Breaking change: 1.2.0 → 2.0.0
   ```

### **Phase 2: Version File Updates**

3. **Update VERSION File**
   ```bash
   # Replace X.Y.Z with your target version
   echo "X.Y.Z" > VERSION
   
   # Verify the change
   make version
   ```

4. **Update version.go File (CRITICAL)**
   ```bash
   # CRITICAL: Update hardcoded version for go install consistency
   # Edit internal/version/version.go line 13:
   # Change: Version = "OLD.VERSION" 
   # To:     Version = "X.Y.Z"
   
   # Verify both files are in sync:
   echo "VERSION file: $(cat VERSION)"
   grep 'Version = ' internal/version/version.go
   ```

5. **Update CHANGELOG.md**
   
   Add a new section at the top of CHANGELOG.md:
   ```markdown
   ## [X.Y.Z] - YYYY-MM-DD

   ### ✨ New Features
   - **Feature Name**: Brief description of what was added
   - **Enhancement**: Description of improvements

   ### 🛠️ Technical Improvements
   - **Area**: Description of technical changes
   - **Quality**: Code quality or architecture improvements

   ### 🐛 Bug Fixes (if any)
   - **Issue**: Description of what was fixed

   ---
   ```

   **Keep it concise** - Focus on user-visible changes and major technical improvements.

### **Phase 3: Pre-Release Validation**

6. **Run Local Tests**
   ```bash
   # Comprehensive local validation
   make test
   make lint
   make vet
   
   # Verify no issues
   echo "Exit code: $?"
   ```

7. **Build Test**
   ```bash
   # Optional: Test local build to ensure everything compiles
   make build
   ./build/giztui-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) --version
   # Should show: GizTUI X.Y.Z
   ```

### **Phase 4: Release Trigger**

8. **Commit Version Changes**
   ```bash
   # Stage version files (all three are required)
   git add VERSION CHANGELOG.md internal/version/version.go
   
   # Commit with standardized message
   git commit -m "release: prepare vX.Y.Z for [feature name]

   - Update VERSION from A.B.C to X.Y.Z
   - Update version.go hardcoded version for go install consistency  
   - Add CHANGELOG.md entry documenting changes"
   ```

9. **Create and Push Tag**
   ```bash
   # Create release tag
   git tag vX.Y.Z
   
   # Push everything to trigger the automated workflow
   git push origin main
   git push origin vX.Y.Z
   
   # The GitHub workflow will now automatically:
   # - Build binaries for all platforms with injected version info
   # - Run comprehensive tests and quality checks
   # - Generate checksums and archives
   # - Create GitHub release with assets and release notes
   ```

### **Phase 5: Automated Workflow**

10. **Monitor Workflow Execution**
    ```bash
    # Check workflow status
    gh run list --limit 1
    
    # Watch workflow progress (optional)
    gh run watch
    
    # Workflow URL: https://github.com/ajramos/giztui/actions
    ```

    The automated workflow performs:
    - ✅ Multi-platform binary builds with version injection
    - ✅ Comprehensive test suite execution
    - ✅ Security scanning and quality checks
    - ✅ Checksum generation for all assets
    - ✅ Archive creation (.tar.gz, .zip)
    - ✅ GitHub release creation with release notes
    - ✅ Asset upload and publishing

## 🎯 Post-Release Tasks

### **Phase 6: Verification**

11. **Verify Release Completion**
    ```bash
    # Check GitHub release was created
    gh release view vX.Y.Z
    
    # Verify workflow succeeded
    gh run list --limit 1 --json conclusion
    ```

12. **Test Installation Methods**
    ```bash
    # Test go install (most common user method)
    go install github.com/ajramos/giztui/cmd/giztui@vX.Y.Z
    giztui --version
    # Should show: GizTUI X.Y.Z
    
    # Test binary download (optional)
    gh release download vX.Y.Z
    ```

13. **Verify the Homebrew tap cask bumped** (desktop)
    ```bash
    # The release-desktop.yml `homebrew` job pushes the new version + universal-DMG
    # sha256 to ajramos/homebrew-giztui. It SILENTLY SKIPS if HOMEBREW_TAP_TOKEN is
    # unset — its job still shows green, so verify the tap actually moved:
    curl -fsSL https://raw.githubusercontent.com/ajramos/homebrew-giztui/main/Casks/giztui-desktop.rb | grep -E 'version|sha256'
    # version must equal this release; sha256 must equal the universal .dmg entry in
    # the release's desktop-checksums.txt. If it did NOT bump, either set the
    # HOMEBREW_TAP_TOKEN secret and re-run the workflow, or bump the tap cask manually.
    ```

14. **Update Documentation** (if needed)
    ```bash
    # Update installation instructions in README.md
    # Update any version-specific documentation
    ```

### **Communication**

14. **Announce Release** (optional)
    - Update project README with new version
    - Announce in relevant channels/communities
    - Update package manager entries if applicable

## 🐛 Troubleshooting

### **Workflow Failures**

**Problem**: GitHub workflow fails during build
```bash
# Solution: Check workflow logs and fix issues locally first
gh run list --limit 1
gh run view [RUN_ID]

# Common fixes:
make test        # Fix failing tests
make lint        # Fix linting issues  
make vet         # Fix static analysis issues
```

**Problem**: Version injection not working in workflow
```bash
# Check that VERSION file format is correct
cat VERSION      # Should contain just: X.Y.Z (no 'v' prefix)

# Verify tag format is correct
git tag --list | tail -1   # Should be: vX.Y.Z
```

**Problem**: Go install still shows old version
```bash
# Solution: Ensure version.go was updated AND committed before tagging
grep 'Version = ' internal/version/version.go
# Should show: Version = "X.Y.Z"

# Clear Go module cache and retry
go clean -modcache
go install github.com/ajramos/giztui/cmd/giztui@vX.Y.Z
```

### **Tag Management Issues**

**Problem**: Need to fix release after tagging
```bash
# DON'T move published tags - create patch release instead
# Example: v1.2.0 has issues → create v1.2.1

echo "X.Y.Z+1" > VERSION
# Update version.go and CHANGELOG.md
git add VERSION CHANGELOG.md internal/version/version.go
git commit -m "release: prepare vX.Y.Z+1 patch release"
git tag vX.Y.Z+1
git push origin main && git push origin vX.Y.Z+1
```

**Problem**: Workflow didn't trigger on tag push
```bash
# Check if tag was pushed correctly
git ls-remote --tags origin | grep vX.Y.Z

# Manual workflow trigger (if needed)
gh workflow run release.yml -f version=vX.Y.Z
```

### **Version Consistency Issues**

**Problem**: Different versions in different build methods
- **Make builds**: Use VERSION file + git info → Always correct for releases
- **Go install builds**: Use hardcoded version.go → Must be manually synced
- **Workflow builds**: Use ldflags injection → Always correct for releases

**Solution**: Always update both VERSION and version.go files before releasing.

## 📚 Related Documentation

- [GitHub Release Workflow](../.github/workflows/release.yml) - Automated release process
- [CI/CD Pipeline](../.github/workflows/ci-comprehensive.yml) - Quality assurance
- [Architecture Guide](ARCHITECTURE.md) - Service-first development patterns
- [Testing Guide](TESTING.md) - Quality assurance framework
- [CHANGELOG.md](../CHANGELOG.md) - Release history

## 🎯 Quick Reference Commands

```bash
# Complete release process
echo "X.Y.Z" > VERSION
# Edit internal/version/version.go to set Version = "X.Y.Z"
# Edit CHANGELOG.md with release notes

git add VERSION CHANGELOG.md internal/version/version.go
git commit -m "release: prepare vX.Y.Z for [feature]"
git tag vX.Y.Z
git push origin main && git push origin vX.Y.Z

# Verify workflow success
gh run watch
gh release view vX.Y.Z

# Test installation
go install github.com/ajramos/giztui/cmd/giztui@vX.Y.Z
giztui --version
```

---

**Remember**: This process leverages GitHub's automated infrastructure for consistent, high-quality releases. The workflow handles the complex parts - focus on proper version management and testing.