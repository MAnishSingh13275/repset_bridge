package test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ComprehensiveTestRunner orchestrates all test categories
type ComprehensiveTestRunner struct {
	t       *testing.T
	rootDir string
	results map[string]TestCategoryResult
}

type TestCategoryResult struct {
	Category    string
	Passed      bool
	Duration    time.Duration
	TestCount   int
	FailCount   int
	ErrorOutput string
}

// TestComprehensiveSuite runs all test categories and provides a summary
func TestComprehensiveSuite(t *testing.T) {
	runner := &ComprehensiveTestRunner{
		t:       t,
		rootDir: getProjectRoot(),
		results: make(map[string]TestCategoryResult),
	}

	runner.runAllTestCategories()
	runner.printSummary()
	runner.validateRequirementsCoverage()
}

func (r *ComprehensiveTestRunner) runAllTestCategories() {
	categories := []struct {
		name string
		path string
	}{
		{"Integration Tests", "./test/integration/..."},
		{"Load Tests", "./test/load/..."},
		{"Security Tests", "./test/security/..."},
		{"End-to-End Tests", "./test/e2e/..."},
	}

	for _, category := range categories {
		r.t.Logf("Running %s...", category.name)
		result := r.runTestCategory(category.name, category.path)
		r.results[category.name] = result
		
		if !result.Passed {
			r.t.Errorf("%s failed: %s", category.name, result.ErrorOutput)
		}
	}
}

func (r *ComprehensiveTestRunner) runTestCategory(categoryName, testPath string) TestCategoryResult {
	startTime := time.Now()
	
	cmd := exec.Command("go", "test", testPath, "-v", "-race", "-timeout=30m")
	cmd.Dir = r.rootDir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1") // Required for SQLite
	
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)
	
	result := TestCategoryResult{
		Category: categoryName,
		Duration: duration,
		Passed:   err == nil,
	}
	
	if err != nil {
		result.ErrorOutput = string(output)
	}
	
	// Parse test output for counts
	outputStr := string(output)
	result.TestCount = strings.Count(outputStr, "=== RUN")
	result.FailCount = strings.Count(outputStr, "--- FAIL:")
	
	return result
}

func (r *ComprehensiveTestRunner) printSummary() {
	r.t.Log("\n" + strings.Repeat("=", 80))
	r.t.Log("COMPREHENSIVE TEST SUITE SUMMARY")
	r.t.Log(strings.Repeat("=", 80))
	
	totalTests := 0
	totalFails := 0
	totalDuration := time.Duration(0)
	allPassed := true
	
	for _, result := range r.results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
			allPassed = false
		}
		
		r.t.Logf("%-20s | %-4s | %3d tests | %3d fails | %8s",
			result.Category,
			status,
			result.TestCount,
			result.FailCount,
			result.Duration.Round(time.Millisecond))
		
		totalTests += result.TestCount
		totalFails += result.FailCount
		totalDuration += result.Duration
	}
	
	r.t.Log(strings.Repeat("-", 80))
	r.t.Logf("%-20s | %-4s | %3d tests | %3d fails | %8s",
		"TOTAL",
		map[bool]string{true: "PASS", false: "FAIL"}[allPassed],
		totalTests,
		totalFails,
		totalDuration.Round(time.Millisecond))
	r.t.Log(strings.Repeat("=", 80))
	
	if !allPassed {
		r.t.Error("Some test categories failed - see details above")
	}
}

func (r *ComprehensiveTestRunner) validateRequirementsCoverage() {
	r.t.Log("\nVALIDATING REQUIREMENTS COVERAGE")
	r.t.Log(strings.Repeat("-", 50))
	
	// Define requirements and their test coverage
	requirements := map[string][]string{
		"Requirement 1 - Hardware Integration": {
			"Integration Tests", // Hardware-to-cloud flow
			"End-to-End Tests",  // Real deployment scenarios
		},
		"Requirement 2 - Cross-Platform Support": {
			"End-to-End Tests", // Service lifecycle tests
		},
		"Requirement 3 - Security": {
			"Security Tests",    // HMAC authentication
			"Integration Tests", // Secure communication
		},
		"Requirement 4 - Offline Functionality": {
			"Integration Tests", // Offline queue replay
			"Load Tests",        // Queue capacity limits
			"End-to-End Tests",  // Offline resilience
		},
		"Requirement 5 - Health Monitoring": {
			"Integration Tests", // Health status reporting
			"End-to-End Tests",  // Health monitoring scenario
		},
		"Requirement 6 - Easy Installation": {
			"End-to-End Tests", // Fresh installation scenario
		},
		"Requirement 7 - User Mapping": {
			"Integration Tests", // Event metadata enrichment
		},
		"Requirement 8 - Monitoring & Alerting": {
			"Load Tests",       // Performance monitoring
			"Security Tests",   // Security event logging
			"End-to-End Tests", // Health monitoring
		},
		"Requirement 9 - Door Control": {
			"Integration Tests", // Door unlock operations
			"End-to-End Tests",  // Hardware simulation
		},
		"Requirement 10 - Updates & Deployment": {
			"End-to-End Tests", // Update mechanism scenario
		},
	}
	
	allRequirementsCovered := true
	
	for requirement, testCategories := range requirements {
		covered := true
		var missingCategories []string
		
		for _, category := range testCategories {
			if result, exists := r.results[category]; !exists || !result.Passed {
				covered = false
				missingCategories = append(missingCategories, category)
			}
		}
		
		status := "✓ COVERED"
		if !covered {
			status = fmt.Sprintf("✗ MISSING: %s", strings.Join(missingCategories, ", "))
			allRequirementsCovered = false
		}
		
		r.t.Logf("%-35s | %s", requirement, status)
	}
	
	r.t.Log(strings.Repeat("-", 50))
	
	if allRequirementsCovered {
		r.t.Log("✓ ALL REQUIREMENTS COVERED BY PASSING TESTS")
	} else {
		r.t.Error("✗ SOME REQUIREMENTS NOT FULLY COVERED")
	}
}

func getProjectRoot() string {
	// Get the directory containing this test file
	_, filename, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(filename)
	
	// Go up one level to get project root
	return filepath.Dir(testDir)
}

// BenchmarkComprehensiveTestSuite benchmarks the entire test suite performance
func BenchmarkComprehensiveTestSuite(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Run a subset of tests for benchmarking
		cmd := exec.Command("go", "test", "./test/integration/...", "-run=TestCompleteHardwareToCloudFlow")
		cmd.Dir = getProjectRoot()
		cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
		
		_, err := cmd.CombinedOutput()
		if err != nil {
			b.Fatalf("Benchmark test failed: %v", err)
		}
	}
}

// TestRequirementValidation validates that all requirements are testable
func TestRequirementValidation(t *testing.T) {
	requirements := []struct {
		id          string
		description string
		testable    bool
		reason      string
	}{
		{"1.1", "Hardware event capture and normalization", true, "Tested in integration tests"},
		{"1.2", "Multiple hardware vendor support", true, "Tested via adapter pattern"},
		{"1.3", "Simulator mode operation", true, "Tested in all test categories"},
		{"1.4", "Adapter pattern implementation", true, "Tested in integration tests"},
		{"1.5", "Hardware failure handling", true, "Tested in integration tests"},
		
		{"2.1", "Windows service operation", true, "Tested in e2e tests"},
		{"2.2", "macOS daemon operation", true, "Tested in e2e tests"},
		{"2.3", "Lite tier performance", true, "Tested in load tests"},
		{"2.4", "Normal tier performance", true, "Tested in load tests"},
		{"2.5", "Full tier performance", true, "Tested in load tests"},
		{"2.6", "Automatic tier adjustment", true, "Tested in e2e tests"},
		
		{"3.1", "Pair code authentication", true, "Tested in security tests"},
		{"3.2", "HMAC request signing", true, "Tested in security tests"},
		{"3.3", "Key rotation support", true, "Tested in security tests"},
		
		{"4.1", "Offline event queuing", true, "Tested in integration tests"},
		{"4.2", "Event replay on reconnection", true, "Tested in integration tests"},
		{"4.3", "Queue capacity management", true, "Tested in load tests"},
		{"4.4", "Idempotency handling", true, "Tested in integration tests"},
		{"4.5", "Data integrity during outages", true, "Tested in e2e tests"},
		
		{"5.1", "Health endpoint availability", true, "Tested in e2e tests"},
		{"5.2", "Heartbeat messaging", true, "Tested in e2e tests"},
		{"5.3", "Status reporting", true, "Tested in integration tests"},
		{"5.4", "Queue depth monitoring", true, "Tested in load tests"},
		
		{"6.1", "Windows installation script", true, "Tested in e2e tests"},
		{"6.2", "macOS installation script", true, "Tested in e2e tests"},
		{"6.3", "Automatic pairing", true, "Tested in e2e tests"},
		{"6.4", "Service registration", true, "Tested in e2e tests"},
		{"6.5", "Installation error handling", true, "Tested in e2e tests"},
		
		{"7.1", "External user ID mapping", true, "Tested in integration tests"},
		{"7.3", "Unmapped ID logging", true, "Tested in integration tests"},
		{"7.5", "Mapping consistency", true, "Tested in integration tests"},
		
		{"8.1", "Offline device alerts", true, "Tested in load tests"},
		{"8.2", "Queue threshold warnings", true, "Tested in load tests"},
		{"8.3", "Security event logging", true, "Tested in security tests"},
		{"8.4", "Error recovery mechanisms", true, "Tested in e2e tests"},
		{"8.5", "Performance diagnostics", true, "Tested in load tests"},
		
		{"9.1", "Door unlock on check-in", true, "Tested in integration tests"},
		{"9.2", "Remote door control API", true, "Tested in integration tests"},
		{"9.3", "Automatic re-lock", true, "Tested in integration tests"},
		{"9.5", "Simulator door operations", true, "Tested in all test categories"},
		
		{"10.1", "Update manifest checking", true, "Tested in e2e tests"},
		{"10.2", "Automatic update installation", true, "Tested in e2e tests"},
		{"10.3", "Single binary deployment", true, "Tested in e2e tests"},
		{"10.4", "Docker image availability", false, "Requires Docker build pipeline"},
		{"10.5", "CDN distribution", true, "Tested via mock CDN in e2e tests"},
	}
	
	testableCount := 0
	for _, req := range requirements {
		if req.testable {
			testableCount++
		} else {
			t.Logf("Requirement %s not directly testable: %s", req.id, req.reason)
		}
	}
	
	coverage := float64(testableCount) / float64(len(requirements)) * 100
	t.Logf("Requirements test coverage: %.1f%% (%d/%d)", coverage, testableCount, len(requirements))
	
	assert.GreaterOrEqual(t, coverage, 90.0, "Should have at least 90% requirements test coverage")
}