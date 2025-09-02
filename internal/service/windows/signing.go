package windows

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SigningConfig represents code signing configuration
type SigningConfig struct {
	CertificatePath string // Path to .pfx certificate file
	CertPassword    string // Certificate password
	TimestampURL    string // Timestamp server URL
	Description     string // Binary description
}

// DefaultSigningConfig returns default signing configuration
func DefaultSigningConfig() *SigningConfig {
	return &SigningConfig{
		TimestampURL: "http://timestamp.digicert.com",
		Description:  "Gym Door Access Bridge",
	}
}

// SignBinary signs a Windows binary with Authenticode
func SignBinary(binaryPath string, config *SigningConfig) error {
	if config.CertificatePath == "" {
		return fmt.Errorf("certificate path is required for signing")
	}
	
	if config.CertPassword == "" {
		return fmt.Errorf("certificate password is required for signing")
	}
	
	// Check if signtool.exe is available
	signtoolPath, err := findSignTool()
	if err != nil {
		return fmt.Errorf("signtool.exe not found: %w", err)
	}
	
	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary file does not exist: %s", binaryPath)
	}
	
	// Check if certificate exists
	if _, err := os.Stat(config.CertificatePath); os.IsNotExist(err) {
		return fmt.Errorf("certificate file does not exist: %s", config.CertificatePath)
	}
	
	// Build signtool command
	args := []string{
		"sign",
		"/f", config.CertificatePath,
		"/p", config.CertPassword,
		"/fd", "SHA256", // Use SHA256 digest algorithm
		"/tr", config.TimestampURL,
		"/td", "SHA256", // Use SHA256 for timestamp
		"/d", config.Description,
	}
	
	args = append(args, binaryPath)
	
	// Execute signtool
	cmd := exec.Command(signtoolPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to sign binary: %w\nOutput: %s", err, string(output))
	}
	
	fmt.Printf("Binary signed successfully: %s\n", binaryPath)
	return nil
}

// VerifySignature verifies the Authenticode signature of a binary
func VerifySignature(binaryPath string) error {
	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary file does not exist: %s", binaryPath)
	}
	
	// Find signtool
	signtoolPath, err := findSignTool()
	if err != nil {
		return fmt.Errorf("signtool.exe not found: %w", err)
	}
	
	// Verify signature
	cmd := exec.Command(signtoolPath, "verify", "/pa", binaryPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("signature verification failed: %w\nOutput: %s", err, string(output))
	}
	
	fmt.Printf("Signature verified successfully: %s\n", binaryPath)
	return nil
}

// findSignTool locates signtool.exe on the system
func findSignTool() (string, error) {
	// Common locations for signtool.exe
	commonPaths := []string{
		// Windows SDK paths
		`C:\Program Files (x86)\Windows Kits\10\bin\x64\signtool.exe`,
		`C:\Program Files (x86)\Windows Kits\10\bin\x86\signtool.exe`,
		`C:\Program Files\Microsoft SDKs\Windows\v7.1\Bin\signtool.exe`,
		`C:\Program Files (x86)\Microsoft SDKs\Windows\v7.1A\Bin\signtool.exe`,
		
		// Visual Studio paths
		`C:\Program Files (x86)\Microsoft Visual Studio\2019\Enterprise\SDK\ScopeCppSDK\vc15\VC\bin\signtool.exe`,
		`C:\Program Files (x86)\Microsoft Visual Studio\2019\Professional\SDK\ScopeCppSDK\vc15\VC\bin\signtool.exe`,
		`C:\Program Files (x86)\Microsoft Visual Studio\2017\Enterprise\SDK\ScopeCppSDK\vc15\VC\bin\signtool.exe`,
		`C:\Program Files (x86)\Microsoft Visual Studio\2017\Professional\SDK\ScopeCppSDK\vc15\VC\bin\signtool.exe`,
	}
	
	// Check if signtool is in PATH
	if path, err := exec.LookPath("signtool.exe"); err == nil {
		return path, nil
	}
	
	// Check common installation paths
	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	
	// Try to find Windows SDK installation
	if sdkPath := findWindowsSDK(); sdkPath != "" {
		signtoolPath := filepath.Join(sdkPath, "bin", "x64", "signtool.exe")
		if _, err := os.Stat(signtoolPath); err == nil {
			return signtoolPath, nil
		}
		
		signtoolPath = filepath.Join(sdkPath, "bin", "x86", "signtool.exe")
		if _, err := os.Stat(signtoolPath); err == nil {
			return signtoolPath, nil
		}
	}
	
	return "", fmt.Errorf("signtool.exe not found. Please install Windows SDK or Visual Studio")
}

// findWindowsSDK attempts to locate Windows SDK installation
func findWindowsSDK() string {
	// Check registry for Windows SDK installation
	// This is a simplified approach - in production, you might want to use
	// Windows registry APIs to find the exact SDK version
	
	sdkPaths := []string{
		`C:\Program Files (x86)\Windows Kits\10`,
		`C:\Program Files\Windows Kits\10`,
		`C:\Program Files (x86)\Microsoft SDKs\Windows\v7.1`,
		`C:\Program Files\Microsoft SDKs\Windows\v7.1`,
	}
	
	for _, path := range sdkPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	return ""
}

// CreateSigningScript creates a PowerShell script for automated signing
func CreateSigningScript(outputPath string, config *SigningConfig) error {
	script := fmt.Sprintf(`# Automated binary signing script
# Generated by Gym Door Bridge

param(
    [Parameter(Mandatory=$true)]
    [string]$BinaryPath,
    
    [string]$CertPath = "%s",
    [string]$CertPassword = "%s",
    [string]$TimestampURL = "%s",
    [string]$Description = "%s"
)

# Find signtool.exe
$signtool = $null
$commonPaths = @(
    "${env:ProgramFiles(x86)}\Windows Kits\10\bin\x64\signtool.exe",
    "${env:ProgramFiles(x86)}\Windows Kits\10\bin\x86\signtool.exe",
    "${env:ProgramFiles}\Microsoft SDKs\Windows\v7.1\Bin\signtool.exe"
)

foreach ($path in $commonPaths) {
    if (Test-Path $path) {
        $signtool = $path
        break
    }
}

if (-not $signtool) {
    # Try to find in PATH
    $signtool = Get-Command signtool.exe -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source
}

if (-not $signtool) {
    Write-Error "signtool.exe not found. Please install Windows SDK."
    exit 1
}

# Sign the binary
Write-Host "Signing binary: $BinaryPath"
& $signtool sign /f $CertPath /p $CertPassword /fd SHA256 /tr $TimestampURL /td SHA256 /d $Description $BinaryPath

if ($LASTEXITCODE -eq 0) {
    Write-Host "Binary signed successfully!"
} else {
    Write-Error "Failed to sign binary"
    exit $LASTEXITCODE
}
`, config.CertificatePath, config.CertPassword, config.TimestampURL, config.Description)
	
	// Remove password from script for security
	script = strings.ReplaceAll(script, config.CertPassword, "$(Read-Host -Prompt 'Certificate Password' -AsSecureString | ConvertFrom-SecureString)")
	
	return os.WriteFile(outputPath, []byte(script), 0644)
}