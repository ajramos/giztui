package version

import (
	"strings"
	"testing"
)

func TestGetInfo(t *testing.T) {
	info := GetInfo()

	// Version should always be set
	if info.Version == "" {
		t.Error("Version should not be empty")
	}

	// Should have a build method
	if info.BuildMethod == "" {
		t.Error("BuildMethod should not be empty")
	}

	// Platform should be set
	if !strings.Contains(info.Platform, "/") {
		t.Error("Platform should contain OS/ARCH format")
	}

	// Go version should be set
	if !strings.HasPrefix(info.GoVersion, "go") {
		t.Error("GoVersion should start with 'go'")
	}
}

func TestGetVersionString(t *testing.T) {
	versionStr := GetVersionString()

	if !strings.Contains(versionStr, "GizTUI") {
		t.Error("Version string should contain 'GizTUI'")
	}

	if !strings.Contains(versionStr, Version) {
		t.Error("Version string should contain the version number")
	}
}

func TestGetDetailedVersionString(t *testing.T) {
	detailed := GetDetailedVersionString()

	expectedFields := []string{
		"GizTUI",
		"Git commit:",
		"Build method:",
		"Go version:",
		"Platform:",
	}

	for _, field := range expectedFields {
		if !strings.Contains(detailed, field) {
			t.Errorf("Detailed version string should contain '%s'", field)
		}
	}
}

func TestBuildMethodDetection(t *testing.T) {
	method := getBuildMethod()

	// Should be one of the known methods
	validMethods := []string{"make", "go-install", "unknown"}
	found := false
	for _, valid := range validMethods {
		if method == valid {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Build method should be one of %v, got: %s", validMethods, method)
	}
}

func TestIsRelease(t *testing.T) {
	// This test depends on current build state, so we just verify it doesn't panic
	isRel := IsRelease()
	isDev := IsDevelopment()

	// They should be opposites
	if isRel == isDev {
		t.Error("IsRelease() and IsDevelopment() should return opposite values")
	}
}
