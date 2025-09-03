//go:build windows

package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// WindowsCredentialManager uses Windows DPAPI for secure credential storage
type WindowsCredentialManager struct {
	serviceName string
	credPath    string
}

// NewWindowsCredentialManager creates a new Windows credential manager
func NewWindowsCredentialManager() (*WindowsCredentialManager, error) {
	// Get user's AppData directory
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return nil, fmt.Errorf("APPDATA environment variable not set")
	}

	credPath := filepath.Join(appData, "GymDoorBridge", "credentials.dat")

	// Ensure directory exists
	dir := filepath.Dir(credPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create credentials directory: %w", err)
	}

	return &WindowsCredentialManager{
		serviceName: "GymDoorBridge",
		credPath:    credPath,
	}, nil
}

// Windows DPAPI structures and functions
var (
	crypt32                = syscall.NewLazyDLL("crypt32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

// StoreCredentials stores device credentials using Windows DPAPI
func (w *WindowsCredentialManager) StoreCredentials(deviceID, deviceKey string) error {
	creds := DeviceCredentials{
		DeviceID:  deviceID,
		DeviceKey: deviceKey,
	}

	// Marshal credentials to JSON
	data, err := json.Marshal(creds)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Encrypt using DPAPI
	encryptedData, err := w.encryptData(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	// Store in Windows registry or credential store
	// For simplicity, we'll use a file-based approach with DPAPI encryption
	return w.writeEncryptedFile(encryptedData)
}

// GetCredentials retrieves device credentials using Windows DPAPI
func (w *WindowsCredentialManager) GetCredentials() (string, string, error) {
	// Read encrypted file
	encryptedData, err := w.readEncryptedFile()
	if err != nil {
		return "", "", fmt.Errorf("failed to read encrypted credentials: %w", err)
	}

	// Decrypt using DPAPI
	data, err := w.decryptData(encryptedData)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	// Unmarshal credentials
	var creds DeviceCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return creds.DeviceID, creds.DeviceKey, nil
}

// DeleteCredentials removes stored credentials
func (w *WindowsCredentialManager) DeleteCredentials() error {
	return w.deleteEncryptedFile()
}

// HasCredentials checks if credentials are stored
func (w *WindowsCredentialManager) HasCredentials() bool {
	return w.encryptedFileExists()
}

// encryptData encrypts data using Windows DPAPI
func (w *WindowsCredentialManager) encryptData(data []byte) ([]byte, error) {
	var inBlob dataBlob
	inBlob.pbData = &data[0]
	inBlob.cbData = uint32(len(data))

	var outBlob dataBlob

	ret, _, err := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0, // description
		0, // optional entropy
		0, // reserved
		0, // prompt struct
		0, // flags
		uintptr(unsafe.Pointer(&outBlob)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("CryptProtectData failed: %v", err)
	}

	// Copy the encrypted data
	encryptedData := make([]byte, outBlob.cbData)
	copy(encryptedData, (*[1 << 30]byte)(unsafe.Pointer(outBlob.pbData))[:outBlob.cbData:outBlob.cbData])

	// Free the allocated memory
	syscall.LocalFree(syscall.Handle(unsafe.Pointer(outBlob.pbData)))

	return encryptedData, nil
}

// decryptData decrypts data using Windows DPAPI
func (w *WindowsCredentialManager) decryptData(encryptedData []byte) ([]byte, error) {
	var inBlob dataBlob
	inBlob.pbData = &encryptedData[0]
	inBlob.cbData = uint32(len(encryptedData))

	var outBlob dataBlob

	ret, _, err := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0, // description
		0, // optional entropy
		0, // reserved
		0, // prompt struct
		0, // flags
		uintptr(unsafe.Pointer(&outBlob)),
	)

	if ret == 0 {
		return nil, fmt.Errorf("CryptUnprotectData failed: %v", err)
	}

	// Copy the decrypted data
	data := make([]byte, outBlob.cbData)
	copy(data, (*[1 << 30]byte)(unsafe.Pointer(outBlob.pbData))[:outBlob.cbData:outBlob.cbData])

	// Free the allocated memory
	syscall.LocalFree(syscall.Handle(unsafe.Pointer(outBlob.pbData)))

	return data, nil
}

// File operations for credential storage
func (w *WindowsCredentialManager) writeEncryptedFile(data []byte) error {
	file, err := os.Create(w.credPath)
	if err != nil {
		return fmt.Errorf("failed to create credentials file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

func (w *WindowsCredentialManager) readEncryptedFile() ([]byte, error) {
	if !w.encryptedFileExists() {
		return nil, fmt.Errorf("no credentials stored")
	}

	data, err := os.ReadFile(w.credPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	return data, nil
}

func (w *WindowsCredentialManager) deleteEncryptedFile() error {
	if w.encryptedFileExists() {
		err := os.Remove(w.credPath)
		if err != nil {
			return fmt.Errorf("failed to delete credentials file: %w", err)
		}
	}
	return nil
}

func (w *WindowsCredentialManager) encryptedFileExists() bool {
	_, err := os.Stat(w.credPath)
	return err == nil
}

// newPlatformCredentialManager creates a Windows credential manager
func newPlatformCredentialManager() (CredentialManager, error) {
	return NewWindowsCredentialManager()
}
