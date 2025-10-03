#!/usr/bin/env pwsh
<#
.SYNOPSIS
    Validates .gitignore effectiveness for the Gym Door Bridge project
.DESCRIPTION
    This script tests that all build artifacts, runtime data files, and temporary files
    are properly excluded by the .gitignore configuration.
.EXAMPLE
    .\scripts\validate-gitignore.ps1
#>

param(
    [switch]$Verbose
)

Write-Host "=== .gitignore Effectiveness Validation ===" -ForegroundColor Cyan
Write-Host ""

# Define test patterns that should be ignored
$testPatterns = @{
    "Build Artifacts" = @(
        "gym-door-bridge.exe",
        "bridge.exe",
        "test-build.exe",
        "api.test.exe",
        "test.dll",
        "test.so",
        "test.dylib"
    )
    "Runtime Data" = @(
        "test.db",
        "test.db-wal",
        "test.db-shm", 
        "test.sqlite",
        "test.sqlite3",
        "backup_test.db"
    )
    "Log Files" = @(
        "test.log",
        "install.log",
        "debug.log"
    )
    "Temporary Files" = @(
        "test.tmp",
        "install.tmp",
        "test.cache",
        "test.pid",
        "test.lock"
    )
    "Configuration Files" = @(
        "config.yaml"
    )
    "Coverage and Profiling" = @(
        "coverage.out",
        "coverage.html",
        "test.prof",
        "cpu.prof",
        "memory.prof"
    )
}

$testDirectories = @(
    "build",
    "dist", 
    "tmp",
    "cache",
    ".cache"
)

$allTestsPassed = $true
$createdFiles = @()
$createdDirs = @()

try {
    # Test file patterns
    foreach ($category in $testPatterns.Keys) {
        Write-Host "Testing $category..." -ForegroundColor Yellow
        
        foreach ($pattern in $testPatterns[$category]) {
            # Create test file
            New-Item -ItemType File -Path $pattern -Force | Out-Null
            $createdFiles += $pattern
            
            # Check if it appears in git status as untracked (should not if properly ignored)
            $gitStatus = git status --porcelain 2>$null
            $isUntracked = $gitStatus | Where-Object { $_ -match "^\?\?" -and $_ -match [regex]::Escape($pattern) }
            
            if ($isUntracked) {
                Write-Host "  ❌ FAIL: $pattern is not being ignored (shows as untracked)" -ForegroundColor Red
                $allTestsPassed = $false
                if ($Verbose) {
                    Write-Host "    Git status shows: $isUntracked" -ForegroundColor Gray
                }
            } else {
                Write-Host "  ✅ PASS: $pattern is properly ignored" -ForegroundColor Green
            }
        }
    }
    
    # Test directory patterns
    Write-Host "Testing Directory Patterns..." -ForegroundColor Yellow
    foreach ($dir in $testDirectories) {
        # Create test directory with content
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
        New-Item -ItemType File -Path "$dir/test-content.txt" -Force | Out-Null
        $createdDirs += $dir
        
        # Check if directory contents appear in git status as untracked
        $gitStatus = git status --porcelain 2>$null
        $isUntracked = $gitStatus | Where-Object { $_ -match "^\?\?" -and $_ -match [regex]::Escape("$dir/") }
        
        if ($isUntracked) {
            Write-Host "  ❌ FAIL: $dir/ directory is not being ignored (shows as untracked)" -ForegroundColor Red
            $allTestsPassed = $false
            if ($Verbose) {
                Write-Host "    Git status shows: $isUntracked" -ForegroundColor Gray
            }
        } else {
            Write-Host "  ✅ PASS: $dir/ directory is properly ignored" -ForegroundColor Green
        }
    }
    
    # Check for any currently tracked files that should be ignored
    Write-Host "Checking for tracked files that should be ignored..." -ForegroundColor Yellow
    $trackedFiles = git ls-files 2>$null
    $problematicFiles = $trackedFiles | Where-Object { 
        $_ -match '\.(exe|dll|so|dylib|db|db-wal|db-shm|sqlite|sqlite3|log|tmp|cache|pid|lock|prof)$' -or
        $_ -match '^(build|dist|tmp|cache)/' -or
        $_ -match 'config\.yaml$'
    }
    
    if ($problematicFiles) {
        Write-Host "  ❌ WARNING: Found tracked files that should be ignored:" -ForegroundColor Red
        foreach ($file in $problematicFiles) {
            Write-Host "    - $file" -ForegroundColor Red
        }
        Write-Host "  Consider running: git rm --cached <filename>" -ForegroundColor Yellow
    } else {
        Write-Host "  ✅ PASS: No problematic tracked files found" -ForegroundColor Green
    }
    
} finally {
    # Clean up test files and directories
    Write-Host "Cleaning up test files..." -ForegroundColor Gray
    foreach ($file in $createdFiles) {
        Remove-Item $file -Force -ErrorAction SilentlyContinue
    }
    foreach ($dir in $createdDirs) {
        Remove-Item $dir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Write-Host ""
if ($allTestsPassed) {
    Write-Host "=== ✅ ALL TESTS PASSED ===" -ForegroundColor Green
    Write-Host ".gitignore is properly configured and effective" -ForegroundColor Green
    exit 0
} else {
    Write-Host "=== ❌ SOME TESTS FAILED ===" -ForegroundColor Red
    Write-Host "Please review and update .gitignore configuration" -ForegroundColor Red
    exit 1
}