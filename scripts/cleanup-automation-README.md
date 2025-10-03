# File Cleanup Automation Scripts

This directory contains automated file cleanup scripts designed to identify and remove duplicate files, build artifacts, and other unnecessary files from the project.

## Scripts Overview

### 1. `cleanup-automation.go`
The main Go-based cleanup engine that provides:
- **Duplicate file detection** using SHA256 hash comparison
- **Backup mechanism** to safely store files before removal
- **Pattern-based file identification** for build artifacts and temporary files
- **Dry-run mode** to preview changes before execution

### 2. `cleanup-automation.ps1`
PowerShell wrapper script for Windows users that provides:
- Easy command-line interface
- Parameter validation
- Go installation verification
- Cross-platform compatibility

### 3. `cleanup-automation.sh`
Shell script wrapper for Unix-like systems (Linux, macOS) that provides:
- Bash-based interface
- POSIX-compliant execution
- Automatic permission handling

## Usage

### Windows (PowerShell)
```powershell
# Dry run to see what would be cleaned
.\scripts\cleanup-automation.ps1 -DryRun -Verbose

# Perform actual cleanup with custom backup directory
.\scripts\cleanup-automation.ps1 -BackupDir "my-backup-folder"

# Show help
.\scripts\cleanup-automation.ps1 -Help
```

### Linux/macOS (Bash)
```bash
# Make script executable (first time only)
chmod +x scripts/cleanup-automation.sh

# Dry run to see what would be cleaned
./scripts/cleanup-automation.sh --dry-run --verbose

# Perform actual cleanup with custom backup directory
./scripts/cleanup-automation.sh --backup-dir my-backup-folder

# Show help
./scripts/cleanup-automation.sh --help
```

### Direct Go Execution
```bash
# From project root directory
go run scripts/cleanup-automation.go --dry-run --verbose
```

## Features

### Duplicate File Detection
- Uses SHA256 hashing to identify identical files
- Preserves the first occurrence (usually in the most appropriate location)
- Marks subsequent duplicates for removal

### Backup Mechanism
- Creates timestamped backup directories
- Maintains original directory structure in backups
- Generates backup manifest with removal reasons
- Allows safe rollback if needed

### Pattern-Based Cleanup
The script identifies and removes files matching these patterns:
- **Build artifacts**: `*.exe`, `build/**`, `dist/**`
- **Runtime files**: `*.db`, `*.db-shm`, `*.db-wal`, `config.yaml`
- **Temporary files**: `*.log`, `*.tmp`
- **Duplicate documentation**: `releases/**/README.md`

### Specific File Removals
Based on project analysis, these specific files are targeted:
- `gym-door-bridge.exe` (root directory executable)
- `simple-repset-installer.ps1` (duplicate installer)
- `repset-bridge-installer-FIXED.ps1` (duplicate installer)
- `setup-heartbeat-task.ps1` (duplicate script)
- `bridge-heartbeat-service.ps1` (duplicate script)

## Safety Features

### Dry Run Mode
Always test with `--dry-run` first to see what would be changed:
```powershell
.\scripts\cleanup-automation.ps1 -DryRun -Verbose
```

### Automatic Backups
All files are backed up before removal unless in dry-run mode:
- Backup directory: `cleanup-backup-YYYYMMDD-HHMMSS`
- Maintains original directory structure
- Includes manifest file with removal reasons

### Skip Patterns
The following directories are automatically skipped:
- `.git` (version control)
- `.kiro` (Kiro configuration)
- `node_modules` (Node.js dependencies)
- `vendor` (Go vendor directory)

## Requirements

- **Go 1.16+** must be installed and available in PATH
- **Write permissions** in the project directory
- **Sufficient disk space** for backup creation

## Output

The script provides detailed output including:
- File scanning progress
- Duplicate detection results
- Backup creation status
- File removal confirmation
- Summary statistics

### Example Output
```
Starting file cleanup automation...
Scanning files and calculating hashes...
Scanned: README.md (hash: a1b2c3d4)
Scanned: docs/README.md (hash: a1b2c3d4)
Found duplicate: docs/README.md (original: README.md)
Found 15 duplicate files
Found 8 files to remove based on cleanup rules
Creating backup in directory: cleanup-backup-20241004-143022
Backed up: docs/README.md -> cleanup-backup-20241004-143022/docs/README.md
Backup completed. 23 files backed up.
Removing 23 files...
Removed: docs/README.md (Duplicate of README.md)
Successfully removed 23 files.
Cleaning up empty directories...
Cleanup completed successfully!
```

## Troubleshooting

### Go Not Found
If you get "Go not found" errors:
1. Install Go from https://golang.org/dl/
2. Ensure Go is in your system PATH
3. Restart your terminal/PowerShell session

### Permission Errors
If you encounter permission errors:
- Ensure you have write access to the project directory
- On Unix systems, make the shell script executable: `chmod +x scripts/cleanup-automation.sh`
- Run with appropriate privileges if needed

### Backup Recovery
To restore files from backup:
1. Locate the backup directory (e.g., `cleanup-backup-20241004-143022`)
2. Review the `backup-manifest.txt` file
3. Copy files back to their original locations as needed

## Integration

This script can be integrated into:
- **CI/CD pipelines** for automated cleanup
- **Pre-commit hooks** to prevent artifact commits
- **Scheduled maintenance** tasks
- **Development workflows** for regular cleanup

## Customization

To customize the cleanup behavior:
1. Modify the `removePatterns` in `identifyFilesToRemove()`
2. Add specific files to the `specificFiles` list
3. Adjust `skipPatterns` to exclude additional directories
4. Modify hash calculation for different file types if needed