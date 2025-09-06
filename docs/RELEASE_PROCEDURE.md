# 🚀 Release Procedure Guide

This document provides a comprehensive, step-by-step procedure for creating GizTUI releases. Follow these steps exactly to ensure consistent, high-quality releases.

## 📋 Table of Contents

- [Pre-Release Checklist](#-pre-release-checklist)
- [Semantic Versioning Guidelines](#-semantic-versioning-guidelines)
- [Step-by-Step Release Process](#-step-by-step-release-process)
- [Post-Release Tasks](#-post-release-tasks)
- [Rollback Procedures](#-rollback-procedures)
- [Troubleshooting](#-troubleshooting)

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
- [ ] All new features follow service-first architecture (CLAUDE.md)
- [ ] Command parity implemented (every keyboard shortcut has `:command`)
- [ ] Proper error handling using `ErrorHandler` pattern
- [ ] Thread-safe accessor methods used throughout
- [ ] Theming implemented with `GetComponentColors()` pattern

### **Documentation Requirements**
- [ ] All new features documented in appropriate docs
- [ ] Breaking changes clearly identified
- [ ] Configuration changes documented
- [ ] Keyboard shortcuts updated if needed

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

## 🎯 Step-by-Step Release Process

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

### **Phase 2: Pre-Release Validation**

3. **Run Complete Test Suite**
   ```bash
   # Clean build and test
   make clean
   make test
   make lint
   make vet
   
   # Verify no issues
   echo "Exit code: $?"
   ```

4. **Version Verification**
   ```bash
   # Ensure we're on main with clean state
   git checkout main
   git pull origin main
   git status  # Should be clean
   ```

### **Phase 3: Version Update**

5. **Update VERSION File**
   ```bash
   # Replace X.Y.Z with your target version
   echo "X.Y.Z" > VERSION
   
   # Verify the change
   make version
   ```

6. **Update CHANGELOG.md**
   
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

### **Phase 4: Pre-Release Testing**

7. **Build and Test Release**
   ```bash
   # Test the release build process
   make release-build
   
   # Verify all binaries were created
   ls -la build/
   
   # Test the primary binary
   ./build/giztui-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m) --version
   ```

### **Phase 5: Commit and Tag**

8. **Commit Version Changes**
   ```bash
   # Stage version changes
   git add VERSION CHANGELOG.md
   
   # Commit with standardized message
   git commit -m "release: bump version to vX.Y.Z for [feature name]

   - Update VERSION from A.B.C to X.Y.Z
   - Add CHANGELOG.md entry documenting changes
   - [Brief description of main changes]

   🤖 Generated with [Claude Code](https://claude.ai/code)

   Co-Authored-By: Claude <noreply@anthropic.com>"
   ```

9. **Push and Verify CI**
   ```bash
   # Push to remote
   git push origin main
   
   # Wait for CI to pass - check GitHub Actions
   # Visit: https://github.com/ajramos/giztui/actions
   ```

### **Phase 6: Release Creation**

10. **Create Final Release**
    ```bash
    # Build final release artifacts
    make release
    
    # Verify all files are present
    ls -la build/
    cat build/checksums.txt
    cat build/archive-checksums.txt
    ```

11. **Create GitHub Release**
    ```bash
    # Create release with binaries
    gh release create vX.Y.Z build/*.tar.gz build/*.zip \
      --title "GizTUI vX.Y.Z - [Feature Name]" \
      --notes "$(cat <<'EOF'
    # 🎉 GizTUI vX.Y.Z - [Feature Name]

    Brief description of the main changes in this release.

    ## ✨ **New Features**

    - **Feature 1**: Description
    - **Feature 2**: Description

    ## 🛠️ **Technical Improvements**

    - **Area**: Description of improvements
    - **Quality**: Code quality enhancements

    ## 📦 **Installation**

    Download the appropriate binary for your platform from the assets below, or install via Go:

    \`\`\`bash
    go install github.com/ajramos/giztui/cmd/giztui@vX.Y.Z
    \`\`\`

    **Full Changelog**: https://github.com/ajramos/giztui/compare/v[PREV]...vX.Y.Z

    🤖 Generated with [Claude Code](https://claude.ai/code)
    EOF
    )"
    ```

## 🎯 Post-Release Tasks

### **Verification**

12. **Verify Release**
    ```bash
    # Check GitHub release page
    gh release view vX.Y.Z
    
    # Test installation
    go install github.com/ajramos/giztui/cmd/giztui@vX.Y.Z
    giztui --version
    ```

13. **Update Documentation** (if needed)
    ```bash
    # Update installation instructions in README.md
    # Update any version-specific documentation
    ```

### **Communication**

14. **Announce Release** (optional)
    - Update project README with new version
    - Announce in relevant channels/communities
    - Update package manager entries if applicable

## 🔄 Rollback Procedures

If you need to rollback a release:

### **For Draft Releases (before publishing)**
```bash
# Delete draft release
gh release delete vX.Y.Z

# Reset VERSION file
git checkout HEAD~1 -- VERSION CHANGELOG.md
git commit -m "rollback: revert version bump to vX.Y.Z"
git push origin main
```

### **For Published Releases**
```bash
# Mark release as pre-release (don't delete published releases)
gh release edit vX.Y.Z --prerelease

# Create hotfix release with proper version
echo "X.Y.Z-hotfix.1" > VERSION
# Follow normal release process
```

## 🐛 Troubleshooting

### **Common Issues**

**Problem**: `make release` fails with test errors
```bash
# Solution: Fix failing tests first
make test
# Address any failing tests, then retry release
```

**Problem**: Git is not clean
```bash
# Solution: Clean working directory
git status
git stash  # if you want to keep changes
git checkout .  # if you want to discard changes
```

**Problem**: `gh` command not found
```bash
# Solution: Install GitHub CLI
# macOS: brew install gh
# Linux: https://cli.github.com/
# Windows: winget install GitHub.cli
gh auth login
```

**Problem**: Version injection not working
```bash
# Solution: Ensure VERSION file exists and has content
cat VERSION
make clean
make build
./build/giztui --version
```

**Problem**: Pre-commit hooks failing
```bash
# Solution: Fix formatting and linting issues
make fmt
make lint
# Address any issues, then retry commit
```

### **Emergency Procedures**

**Critical Bug in Release**:
1. Immediately mark release as pre-release: `gh release edit vX.Y.Z --prerelease`
2. Create hotfix branch: `git checkout -b hotfix/vX.Y.Z+1`
3. Fix critical issue
4. Follow normal release process for patch version
5. Update original release notes with deprecation notice

**Build System Issues**:
1. Verify Go version: `go version` (should be 1.23+)
2. Clean and retry: `make clean && make release`
3. Check disk space: `df -h`
4. Verify network access for dependency downloads

## 📚 Related Documentation

- [Architecture Guide](ARCHITECTURE.md) - Service-first development patterns
- [Testing Guide](TESTING.md) - Quality assurance framework  
- [Development Setup](DEVELOPMENT_SETUP.md) - Environment setup
- [CHANGELOG.md](../CHANGELOG.md) - Release history
- [CLAUDE.md](../CLAUDE.md) - Development guidelines

## 🎯 Quick Reference Commands

```bash
# Pre-release checks
make test && make lint && make vet

# Version update
echo "X.Y.Z" > VERSION

# Release creation
git add VERSION CHANGELOG.md
git commit -m "release: bump version to vX.Y.Z for [feature]"
git push origin main
make release
gh release create vX.Y.Z build/*.tar.gz build/*.zip --title "GizTUI vX.Y.Z"

# Verification
gh release view vX.Y.Z
go install github.com/ajramos/giztui/cmd/giztui@vX.Y.Z
```

---

**Remember**: This process ensures consistent, high-quality releases. Don't skip steps - each one serves a specific purpose in maintaining software quality and user experience.