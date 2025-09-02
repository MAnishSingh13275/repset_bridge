package windows

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSigningConfig(t *testing.T) {
	config := DefaultSigningConfig()
	
	assert.NotNil(t, config)
	assert.Equal(t, "http://timestamp.digicert.com", config.TimestampURL)
	assert.Equal(t, "Gym Door Access Bridge", config.Description)
	assert.Empty(t, config.CertificatePath)
	assert.Empty(t, config.CertPassword)
}

func TestSignBinaryValidation(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Signing tests only run on Windows")
	}
	
	// Create a temporary file to simulate a binary
	tempDir, err := os.MkdirTemp("", "signing_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	binaryPath := filepath.Join(tempDir, "test.exe")
	err = os.WriteFile(binaryPath, []byte("fake binary content"), 0644)
	require.NoError(t, err)
	
	tests := []struct {
		name      string
		config    *SigningConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "missing certificate path",
			config: &SigningConfig{
				CertificatePath: "",
				CertPassword:    "password",
				TimestampURL:    "http://timestamp.digicert.com",
				Description:     "Test",
			},
			expectErr: true,
			errMsg:    "certificate path is required",
		},
		{
			name: "missing certificate password",
			config: &SigningConfig{
				CertificatePath: "cert.pfx",
				CertPassword:    "",
				TimestampURL:    "http://timestamp.digicert.com",
				Description:     "Test",
			},
			expectErr: true,
			errMsg:    "certificate password is required",
		},
		{
			name: "non-existent binary",
			config: &SigningConfig{
				CertificatePath: "cert.pfx",
				CertPassword:    "password",
				TimestampURL:    "http://timestamp.digicert.com",
				Description:     "Test",
			},
			expectErr: true,
			errMsg:    "", // Don't check specific error message as it depends on signtool availability
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testBinaryPath string
			if tt.name == "non-existent binary" {
				testBinaryPath = filepath.Join(tempDir, "nonexistent.exe")
			} else {
				testBinaryPath = binaryPath
			}
			
			err := SignBinary(testBinaryPath, tt.config)
			
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				// Note: This will likely fail in test environment due to missing signtool
				// or invalid certificate, but we're testing the validation logic
				if err != nil {
					t.Logf("Expected signing to work but got error (likely missing signtool): %v", err)
				}
			}
		})
	}
}

func TestVerifySignatureValidation(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Signing tests only run on Windows")
	}
	
	// Test with non-existent file
	err := VerifySignature("nonexistent.exe")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "binary file does not exist")
}

func TestFindSignTool(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("SignTool tests only run on Windows")
	}
	
	// This test will try to find signtool.exe
	// It might fail in environments without Windows SDK
	signtoolPath, err := findSignTool()
	
	if err != nil {
		t.Logf("SignTool not found (expected in many test environments): %v", err)
		// This is not necessarily a failure - many test environments don't have Windows SDK
	} else {
		t.Logf("Found SignTool at: %s", signtoolPath)
		
		// Verify the file exists
		_, err := os.Stat(signtoolPath)
		assert.NoError(t, err, "SignTool path should exist")
	}
}

func TestFindWindowsSDK(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows SDK tests only run on Windows")
	}
	
	sdkPath := findWindowsSDK()
	
	if sdkPath == "" {
		t.Log("Windows SDK not found (expected in many test environments)")
	} else {
		t.Logf("Found Windows SDK at: %s", sdkPath)
		
		// Verify the path exists
		_, err := os.Stat(sdkPath)
		assert.NoError(t, err, "Windows SDK path should exist")
	}
}

func TestCreateSigningScript(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "signing_script_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	config := &SigningConfig{
		CertificatePath: "C:\\certs\\mycert.pfx",
		CertPassword:    "secretpassword",
		TimestampURL:    "http://timestamp.digicert.com",
		Description:     "Test Application",
	}
	
	scriptPath := filepath.Join(tempDir, "sign.ps1")
	err = CreateSigningScript(scriptPath, config)
	require.NoError(t, err)
	
	// Verify script was created
	assert.FileExists(t, scriptPath)
	
	// Read script content
	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err)
	
	scriptContent := string(content)
	
	// Verify script contains expected elements
	assert.Contains(t, scriptContent, "param(")
	assert.Contains(t, scriptContent, "[Parameter(Mandatory=$true)]")
	assert.Contains(t, scriptContent, "$BinaryPath")
	assert.Contains(t, scriptContent, config.TimestampURL)
	assert.Contains(t, scriptContent, config.Description)
	
	// Verify password is not in plain text (should be replaced with secure prompt)
	assert.NotContains(t, scriptContent, config.CertPassword)
	assert.Contains(t, scriptContent, "Read-Host -Prompt 'Certificate Password' -AsSecureString")
}

func TestSigningConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config *SigningConfig
		valid  bool
	}{
		{
			name: "valid config",
			config: &SigningConfig{
				CertificatePath: "cert.pfx",
				CertPassword:    "password",
				TimestampURL:    "http://timestamp.digicert.com",
				Description:     "Test App",
			},
			valid: true,
		},
		{
			name: "empty certificate path",
			config: &SigningConfig{
				CertificatePath: "",
				CertPassword:    "password",
				TimestampURL:    "http://timestamp.digicert.com",
				Description:     "Test App",
			},
			valid: false,
		},
		{
			name: "empty certificate password",
			config: &SigningConfig{
				CertificatePath: "cert.pfx",
				CertPassword:    "",
				TimestampURL:    "http://timestamp.digicert.com",
				Description:     "Test App",
			},
			valid: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We don't have a direct validation function, but we can test
			// the validation logic through SignBinary
			tempDir, err := os.MkdirTemp("", "validation_test")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)
			
			binaryPath := filepath.Join(tempDir, "test.exe")
			err = os.WriteFile(binaryPath, []byte("fake binary"), 0644)
			require.NoError(t, err)
			
			err = SignBinary(binaryPath, tt.config)
			
			if tt.valid {
				// Even valid configs might fail due to missing signtool or invalid cert
				// but they shouldn't fail validation
				if err != nil && (err.Error() == "certificate path is required for signing" || 
					err.Error() == "certificate password is required for signing") {
					t.Errorf("Valid config failed validation: %v", err)
				}
			} else {
				// Invalid configs should fail validation
				assert.Error(t, err)
			}
		})
	}
}

// Benchmark signing operations
func BenchmarkDefaultSigningConfig(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DefaultSigningConfig()
	}
}

func BenchmarkFindSignTool(b *testing.B) {
	if runtime.GOOS != "windows" {
		b.Skip("SignTool benchmarks only run on Windows")
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = findSignTool()
	}
}