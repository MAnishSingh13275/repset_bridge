#!/usr/bin/env pwsh
# Security Check Script for Gym Door Bridge Configuration
# This script validates configuration files for common security issues

param(
    [string]$ConfigPath = "config.yaml",
    [switch]$Verbose
)

function Write-SecurityWarning {
    param([string]$Message)
    Write-Host "⚠️  SECURITY WARNING: $Message" -ForegroundColor Yellow
}

function Write-SecurityError {
    param([string]$Message)
    Write-Host "❌ SECURITY ERROR: $Message" -ForegroundColor Red
}

function Write-SecurityOk {
    param([string]$Message)
    Write-Host "✅ $Message" -ForegroundColor Green
}

function Test-ConfigurationSecurity {
    param([string]$FilePath)
    
    if (-not (Test-Path $FilePath)) {
        Write-SecurityError "Configuration file not found: $FilePath"
        return $false
    }
    
    $content = Get-Content $FilePath -Raw
    $issues = @()
    
    # Check for default credentials
    if ($content -match 'username:\s*["\']?admin["\']?') {
        $issues += "Default username 'admin' found"
    }
    
    if ($content -match 'password:\s*["\']?admin["\']?') {
        $issues += "Default password 'admin' found"
    }
    
    # Check for empty credentials that should be filled
    if ($content -match 'device_id:\s*["\']?["\']?') {
        $issues += "Empty device_id - should be set during pairing"
    }
    
    if ($content -match 'device_key:\s*["\']?["\']?') {
        $issues += "Empty device_key - should be set during pairing"
    }
    
    # Check for placeholder values that weren't replaced
    if ($content -match 'YOUR_DEVICE_USERNAME') {
        $issues += "Placeholder 'YOUR_DEVICE_USERNAME' not replaced with actual username"
    }
    
    if ($content -match 'YOUR_DEVICE_PASSWORD') {
        $issues += "Placeholder 'YOUR_DEVICE_PASSWORD' not replaced with actual password"
    }
    
    # Check for insecure settings
    if ($content -match 'tls_enabled:\s*false' -and $content -match 'api_server:') {
        $issues += "TLS disabled for API server - consider enabling for production"
    }
    
    if ($content -match 'auth:\s*\n\s*enabled:\s*false') {
        $issues += "API authentication disabled - consider enabling for production"
    }
    
    return $issues
}

# Main execution
Write-Host "🔒 Security Check for Gym Door Bridge Configuration" -ForegroundColor Cyan
Write-Host "=================================================" -ForegroundColor Cyan
Write-Host ""

$issues = Test-ConfigurationSecurity -FilePath $ConfigPath

if ($issues.Count -eq 0) {
    Write-SecurityOk "No security issues found in $ConfigPath"
} else {
    Write-Host "Found $($issues.Count) security issue(s) in $ConfigPath:" -ForegroundColor Yellow
    Write-Host ""
    
    foreach ($issue in $issues) {
        Write-SecurityWarning $issue
    }
    
    Write-Host ""
    Write-Host "Recommendations:" -ForegroundColor Cyan
    Write-Host "- Replace all default credentials with strong, unique passwords"
    Write-Host "- Enable TLS and authentication for production deployments"
    Write-Host "- Ensure device pairing is completed (device_id and device_key set)"
    Write-Host "- Review all placeholder values and replace with actual values"
}

Write-Host ""
Write-Host "For more security guidance, see: docs/operations/security.md" -ForegroundColor Blue