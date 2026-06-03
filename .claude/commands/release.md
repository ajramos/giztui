---
description: "Complete release management for GizTUI with automated workflow integration"
---

# Release Management: $ARGUMENTS

Manage the complete GizTUI release process with version management, quality gates, and automated workflow integration.

## Release Management Options
**Command Usage:** $ARGUMENTS

**Available Commands:**
- `prepare [version] [description]` - Prepare a new release with version bumping
- `validate` - Run comprehensive pre-release validation
- `publish [version]` - Execute the release workflow
- `hotfix [version] [issue]` - Create emergency patch release
- `status` - Check current release status and version info
- `rollback [version]` - Rollback release (emergency only)

## MANDATORY Release Process

### 1. Pre-Release Validation (CRITICAL)
- **Quality Gates Enforcement**: ALL tests, linting, formatting must pass
- **Architecture Compliance Check**: Service-first, error handling, thread safety
- **Documentation Validation**: Feature docs, changelog, version consistency
- **Git Repository Status**: Clean working directory, up-to-date with origin
- **Dependency Security**: Check for vulnerable dependencies
- **Build Verification**: Multi-platform build test

### 2. Version Management System
- **Semantic Versioning**: Automatic MAJOR.MINOR.PATCH determination
- **Version File Synchronization**: VERSION, version.go, CHANGELOG.md
- **Breaking Change Detection**: Configuration, API, command syntax changes
- **Feature Classification**: New features, bug fixes, improvements
- **Changelog Generation**: Structured release notes with user impact

### 3. Release Workflow Integration
- **Automated GitHub Workflow**: Tag-triggered release pipeline
- **Multi-platform Builds**: Linux, macOS, Windows (AMD64, ARM64)
- **Quality Assurance**: Comprehensive test suite, security scanning
- **Asset Management**: Checksums, archives, installation packages
- **Release Publishing**: GitHub release with assets and notes

### 4. Post-Release Verification
- **Installation Testing**: go install, binary downloads, version verification
- **Documentation Updates**: README, installation guides, feature docs
- **Communication**: Release announcements, package manager updates
- **Monitoring**: Release success metrics, user feedback tracking

## Command Implementations

### **Prepare Release**: `prepare [version] [description]`
```bash
# Examples:
claude release prepare 1.2.0 "AI-powered email management enhancements"
claude release prepare patch "Critical security fixes"
claude release prepare minor "Slack integration and theme improvements"
claude release prepare major "Configuration format overhaul"
```

**Automated Actions:**
1. **Version Analysis**: Analyze git history for change classification
2. **Version Bumping**: Update VERSION, version.go, CHANGELOG.md files
3. **Changelog Generation**: Auto-generate release notes from commits
4. **Pre-validation**: Run quality gates before proceeding
5. **Commit Preparation**: Stage version files with standardized message

### **Validate Release**: `validate`
```bash
# Full pre-release validation
claude release validate
```

**Validation Checklist:**
- [ ] **Code Quality**: Tests passing, linting clean, formatting correct
- [ ] **Architecture Compliance**: Service-first patterns, error handling
- [ ] **Feature Completeness**: Command parity, bulk support, theming
- [ ] **Documentation**: Features documented, shortcuts updated
- [ ] **Git Status**: Clean working directory, synchronized with origin
- [ ] **Dependencies**: Security scan, version compatibility
- [ ] **Build Test**: Multi-platform compilation verification

### **Publish Release**: `publish [version]`
```bash
# Execute automated release workflow
claude release publish 1.2.0
```

**Automated Workflow:**
1. **Final Validation**: Last-minute quality gate verification
2. **Tag Creation**: Create and push release tag to trigger workflow
3. **Workflow Monitoring**: Track GitHub Actions progress
4. **Asset Verification**: Confirm successful builds and uploads
5. **Installation Testing**: Verify go install and binary downloads
6. **Documentation Updates**: README, installation guides

### **Hotfix Release**: `hotfix [version] [issue]`
```bash
# Emergency patch release
claude release hotfix 1.2.1 "Critical deadlock in ESC handler"
```

**Hotfix Process:**
1. **Issue Assessment**: Validate criticality and impact scope
2. **Rapid Version Bump**: PATCH increment with focused changelog
3. **Minimal Testing**: Critical path validation, affected feature testing
4. **Accelerated Workflow**: Fast-track release with reduced validation
5. **Communication**: Urgent release notifications, upgrade recommendations

### **Release Status**: `status`
```bash
# Check current release information
claude release status
```

**Status Report:**
- **Current Version**: Active version from VERSION file and version.go
- **Git Status**: Working directory state, branch info, tag status
- **Last Release**: GitHub release information, workflow status
- **Pending Changes**: Unreleased commits, change classification
- **Quality Status**: Test results, linting status, build verification
- **Workflow Status**: Active/recent GitHub Actions runs

### **Rollback Release**: `rollback [version]` (Emergency Only)
```bash
# Emergency rollback (creates new patch release)
claude release rollback 1.2.0
```

**Rollback Strategy:**
1. **Impact Assessment**: Identify rollback scope and affected users
2. **Patch Release**: Create new version that reverts problematic changes
3. **Documentation**: Clear rollback communication and migration guide
4. **User Notification**: Release announcement explaining rollback reason
5. **Post-Rollback**: Schedule proper fix for next release

## Release Types & Version Bumping

### **MAJOR Release** (X.0.0)
**Breaking Changes Detected:**
- Configuration format modifications
- Command syntax changes
- API compatibility breaks
- Removed features or options

**Preparation Requirements:**
- Migration guide documentation
- Breaking change impact assessment
- Extensive testing across use cases
- User communication strategy

### **MINOR Release** (X.Y.0)
**New Features Added:**
- New keyboard shortcuts or commands
- Integration additions (Slack, Obsidian, AI providers)
- Enhanced functionality (themes, search, bulk operations)
- Configuration option additions

**Quality Requirements:**
- Feature completeness validation
- Command parity enforcement
- Bulk mode support verification
- Architecture compliance check

### **PATCH Release** (X.Y.Z)
**Bug Fixes & Improvements:**
- Bug fixes and stability improvements
- Performance optimizations
- Code refactoring and cleanup
- Documentation updates
- Dependency updates

**Validation Focus:**
- Regression testing
- Performance impact assessment
- Stability verification
- User impact minimization

## Advanced Release Features

### **Changelog Automation**
- **Commit Analysis**: Parse conventional commits for change classification
- **Impact Assessment**: Identify user-facing changes vs internal improvements
- **Section Generation**: Auto-organize into Features, Fixes, Technical Improvements
- **Breaking Change Detection**: Automatically flag and document breaking changes
- **Migration Guide**: Generate upgrade instructions for breaking changes

### **Quality Gate Enforcement**
```bash
# Canonical CI-equivalent local check (fmt + vet + golangci-lint + essential tests)
make pre-commit-check

# Multi-platform compilation verification
make build
```

### **Workflow Integration**
- **GitHub Actions**: Trigger automated release pipeline
- **Asset Management**: Binary builds, checksums, archives
- **Release Notes**: Auto-generated from changelog
- **Installation Testing**: Verify go install compatibility
- **Rollback Capability**: Emergency release reversal

### **Communication & Documentation**
- **README Updates**: Version references, installation instructions
- **Documentation Sync**: Feature docs, keyboard shortcuts, configuration
- **Release Announcements**: Structured communication templates
- **Package Managers**: Update distribution channels

## Troubleshooting & Recovery

### **Common Issues & Solutions**

**Workflow Failures:**
```bash
# Check and fix build issues
make pre-commit-check
gh run list --limit 1
gh run view [RUN_ID]
```

**Version Consistency:**
```bash
# Verify version synchronization
echo "VERSION file: $(cat VERSION)"
grep 'Version = ' internal/version/version.go
make version
```

**Tag Management:**
```bash
# Fix release issues with patch release
echo "X.Y.Z+1" > VERSION
# Update version.go and CHANGELOG.md
git add VERSION CHANGELOG.md internal/version/version.go
git commit -m "release: prepare vX.Y.Z+1 patch release"
```

### **Emergency Procedures**

**Critical Bug in Release:**
1. Create immediate hotfix release (patch version)
2. Revert problematic changes
3. Expedited testing and validation
4. User notification and upgrade guidance

**Workflow Infrastructure Failure:**
1. Manual release process fallback
2. Local build and asset creation
3. Manual GitHub release creation
4. Post-recovery workflow validation

## Integration with GizTUI Architecture

### **Service-First Development Verification**
- **Business Logic**: Validate all logic in `internal/services/`
- **UI Components**: Confirm presentation-only responsibilities
- **Error Handling**: Verify `ErrorHandler` usage throughout
- **Thread Safety**: Check accessor method compliance

### **Feature Completeness Validation**
- **Command Parity**: Every shortcut has equivalent `:command`
- **Bulk Mode**: All features support bulk operations
- **Theming**: `GetComponentColors()` integration
- **Documentation**: Help system updates, README synchronization

### **Testing Framework Integration**
- **Unit Tests**: Service logic with mock dependencies
- **Component Tests**: TUI behavior with test harness
- **Integration Tests**: End-to-end workflow validation
- **Visual Regression**: UI consistency verification
- **Performance Tests**: Critical operation benchmarks

## Release Management Best Practices

### **Version Planning Strategy**
1. **Release Cadence**: Regular minor releases, patch as needed
2. **Feature Batching**: Group related features in single release
3. **Breaking Change Planning**: Major releases with migration support
4. **Hotfix Readiness**: Emergency patch capability maintenance

### **Quality Assurance**
1. **Automated Testing**: Comprehensive test coverage
2. **Manual Validation**: Critical path user testing
3. **Performance Monitoring**: Resource usage and response times
4. **Compatibility Testing**: Multiple environments and configurations

### **Communication Excellence**
1. **Clear Release Notes**: User-focused change descriptions
2. **Migration Guides**: Detailed upgrade instructions for breaking changes
3. **Timeline Communication**: Release schedule and expectations
4. **Community Engagement**: Feedback collection and response

## Quick Reference Commands

```bash
# Complete release workflow
claude release prepare 1.2.0 "Feature description"
claude release validate
claude release publish 1.2.0

# Status and monitoring
claude release status
gh run watch

# Emergency procedures
claude release hotfix 1.2.1 "Critical issue description"
claude release rollback 1.2.0
```

## Reference Documentation

- **Release Procedure**: `docs/RELEASE_PROCEDURE.md` - Detailed manual process
- **Architecture Guide**: `docs/ARCHITECTURE.md` - Development patterns validation
- **Testing Guide**: `docs/TESTING.md` - Quality assurance framework
- **GitHub Workflows**: `.github/workflows/` - Automated release infrastructure
- **Version Management**: `VERSION`, `internal/version/version.go` - Version tracking

**CRITICAL**: This release management system leverages GizTUI's established patterns and automated infrastructure to ensure consistent, high-quality releases while minimizing manual effort and human error.

Execute release management with confidence using the comprehensive automation and validation provided by this command system.
