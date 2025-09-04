package discovery

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// DeviceInfo represents a discovered biometric device
type DeviceInfo struct {
	Type         string            `json:"type"`          // essl, zkteco, realtime
	IP           string            `json:"ip"`            // Device IP address
	Port         int               `json:"port"`          // Device port
	Model        string            `json:"model"`         // Device model
	SerialNumber string            `json:"serial_number"` // Device serial number
	Version      string            `json:"version"`       // Firmware version
	Status       string            `json:"status"`        // online, offline
	Config       map[string]string `json:"config"`        // Auto-generated config
}

// DeviceDiscovery handles automatic discovery of biometric devices
type DeviceDiscovery struct {
	logger    *logrus.Logger
	devices   map[string]*DeviceInfo
	mutex     sync.RWMutex
	stopChan  chan struct{}
	isRunning bool
}

// NewDeviceDiscovery creates a new device discovery instance
func NewDeviceDiscovery(logger *logrus.Logger) *DeviceDiscovery {
	return &DeviceDiscovery{
		logger:   logger,
		devices:  make(map[string]*DeviceInfo),
		stopChan: make(chan struct{}),
	}
}

// Start begins device discovery
func (d *DeviceDiscovery) Start(ctx context.Context) error {
	d.isRunning = true
	d.logger.Info("Starting biometric device discovery")

	// Initial discovery scan
	go d.discoveryLoop(ctx)

	return nil
}

// Stop stops device discovery
func (d *DeviceDiscovery) Stop() error {
	if !d.isRunning {
		return nil
	}

	d.isRunning = false
	close(d.stopChan)
	d.logger.Info("Device discovery stopped")
	return nil
}

// GetDiscoveredDevices returns all discovered devices
func (d *DeviceDiscovery) GetDiscoveredDevices() map[string]*DeviceInfo {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	// Return a copy to avoid race conditions
	devices := make(map[string]*DeviceInfo)
	for k, v := range d.devices {
		devices[k] = v
	}
	return devices
}

// discoveryLoop runs continuous device discovery
func (d *DeviceDiscovery) discoveryLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Scan every 30 seconds
	defer ticker.Stop()

	// Initial scan
	d.scanForDevices()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopChan:
			return
		case <-ticker.C:
			d.scanForDevices()
		}
	}
}

// scanForDevices scans the local network for biometric devices
func (d *DeviceDiscovery) scanForDevices() {
	d.logger.Debug("Scanning for biometric devices")

	// Get local network ranges
	networks, err := d.getLocalNetworks()
	if err != nil {
		d.logger.WithError(err).Error("Failed to get local networks")
		return
	}

	var wg sync.WaitGroup
	deviceChan := make(chan *DeviceInfo, 100)

	// Scan each network
	for _, network := range networks {
		wg.Add(1)
		go func(net *net.IPNet) {
			defer wg.Done()
			d.scanNetwork(net, deviceChan)
		}(network)
	}

	// Close channel when all scans complete
	go func() {
		wg.Wait()
		close(deviceChan)
	}()

	// Collect discovered devices
	for device := range deviceChan {
		d.mutex.Lock()
		key := fmt.Sprintf("%s:%d", device.IP, device.Port)
		d.devices[key] = device
		d.mutex.Unlock()

		d.logger.WithFields(logrus.Fields{
			"type":   device.Type,
			"ip":     device.IP,
			"port":   device.Port,
			"model":  device.Model,
		}).Info("Discovered biometric device")
	}
}

// getLocalNetworks returns local network ranges to scan
func (d *DeviceDiscovery) getLocalNetworks() ([]*net.IPNet, error) {
	var networks []*net.IPNet

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil { // IPv4 only
					networks = append(networks, ipnet)
				}
			}
		}
	}

	return networks, nil
}

// scanNetwork scans a network range for biometric devices
func (d *DeviceDiscovery) scanNetwork(network *net.IPNet, deviceChan chan<- *DeviceInfo) {
	// Common biometric device ports
	ports := []int{4370, 80, 8080, 8000, 9999, 5005}

	// Generate IP range
	ips := d.generateIPRange(network)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 50) // Limit concurrent connections

	for _, ip := range ips {
		for _, port := range ports {
			wg.Add(1)
			go func(ip string, port int) {
				defer wg.Done()
				semaphore <- struct{}{} // Acquire
				defer func() { <-semaphore }() // Release

				if device := d.probeDevice(ip, port); device != nil {
					deviceChan <- device
				}
			}(ip, port)
		}
	}

	wg.Wait()
}

// generateIPRange generates all IPs in a network range
func (d *DeviceDiscovery) generateIPRange(network *net.IPNet) []string {
	var ips []string
	
	ip := network.IP.Mask(network.Mask)
	for ip := ip.Mask(network.Mask); network.Contains(ip); d.incrementIP(ip) {
		ips = append(ips, ip.String())
	}
	
	return ips
}

// incrementIP increments an IP address
func (d *DeviceDiscovery) incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// probeDevice attempts to identify a biometric device at the given IP:port
func (d *DeviceDiscovery) probeDevice(ip string, port int) *DeviceInfo {
	address := fmt.Sprintf("%s:%d", ip, port)
	
	// Try to connect with timeout
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return nil // No device or not responding
	}
	defer conn.Close()

	// Try to identify device type
	deviceType := d.identifyDeviceType(conn, ip, port)
	if deviceType == "" {
		return nil // Unknown device type
	}

	// Get device info
	model, serial, version := d.getDeviceInfo(conn, deviceType)

	return &DeviceInfo{
		Type:         deviceType,
		IP:           ip,
		Port:         port,
		Model:        model,
		SerialNumber: serial,
		Version:      version,
		Status:       "online",
		Config:       d.generateDeviceConfig(deviceType, ip, port),
	}
}

// identifyDeviceType attempts to identify the device type
func (d *DeviceDiscovery) identifyDeviceType(conn net.Conn, ip string, port int) string {
	// Set read timeout
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))

	// Try different device identification methods
	if d.isZKTecoDevice(conn, port) {
		return "zkteco"
	}
	
	if d.isESSLDevice(conn, port) {
		return "essl"
	}
	
	if d.isRealtimeDevice(conn, port) {
		return "realtime"
	}

	return ""
}

// isZKTecoDevice checks if device is ZKTeco
func (d *DeviceDiscovery) isZKTecoDevice(conn net.Conn, port int) bool {
	// ZKTeco devices typically use port 4370
	if port != 4370 {
		return false
	}

	// Try ZKTeco connection command
	zkCommand := []byte{0x50, 0x50, 0x82, 0x7D, 0x13, 0x00, 0x00, 0x00}
	_, err := conn.Write(zkCommand)
	if err != nil {
		return false
	}

	// Read response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil || n < 8 {
		return false
	}

	// Check for ZKTeco response pattern
	return buffer[0] == 0x50 && buffer[1] == 0x50
}

// isESSLDevice checks if device is ESSL
func (d *DeviceDiscovery) isESSLDevice(conn net.Conn, port int) bool {
	// ESSL devices often use HTTP on port 80 or 8080
	if port != 80 && port != 8080 {
		return false
	}

	// Try HTTP request to identify ESSL
	httpRequest := "GET / HTTP/1.1\r\nHost: " + conn.RemoteAddr().String() + "\r\n\r\n"
	_, err := conn.Write([]byte(httpRequest))
	if err != nil {
		return false
	}

	// Read response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil || n == 0 {
		return false
	}

	response := string(buffer[:n])
	// Look for ESSL-specific headers or content
	return strings.Contains(strings.ToLower(response), "essl") ||
		   strings.Contains(strings.ToLower(response), "x990") ||
		   strings.Contains(strings.ToLower(response), "biomax")
}

// isRealtimeDevice checks if device is Realtime
func (d *DeviceDiscovery) isRealtimeDevice(conn net.Conn, port int) bool {
	// Realtime devices often use port 5005 or 9999
	if port != 5005 && port != 9999 {
		return false
	}

	// Try Realtime-specific command
	rtCommand := []byte{0xAA, 0x55, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	_, err := conn.Write(rtCommand)
	if err != nil {
		return false
	}

	// Read response
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil || n < 4 {
		return false
	}

	// Check for Realtime response pattern
	return buffer[0] == 0xAA && buffer[1] == 0x55
}

// getDeviceInfo retrieves device information
func (d *DeviceDiscovery) getDeviceInfo(conn net.Conn, deviceType string) (model, serial, version string) {
	// Device-specific info retrieval
	switch deviceType {
	case "zkteco":
		return d.getZKTecoInfo(conn)
	case "essl":
		return d.getESSLInfo(conn)
	case "realtime":
		return d.getRealtimeInfo(conn)
	}
	return "Unknown", "Unknown", "Unknown"
}

// getZKTecoInfo gets ZKTeco device info
func (d *DeviceDiscovery) getZKTecoInfo(conn net.Conn) (model, serial, version string) {
	// Implement ZKTeco info retrieval
	return "ZKTeco Device", "ZK" + strconv.FormatInt(time.Now().Unix(), 10), "1.0"
}

// getESSLInfo gets ESSL device info
func (d *DeviceDiscovery) getESSLInfo(conn net.Conn) (model, serial, version string) {
	// Implement ESSL info retrieval
	return "ESSL Device", "ES" + strconv.FormatInt(time.Now().Unix(), 10), "1.0"
}

// getRealtimeInfo gets Realtime device info
func (d *DeviceDiscovery) getRealtimeInfo(conn net.Conn) (model, serial, version string) {
	// Implement Realtime info retrieval
	return "Realtime Device", "RT" + strconv.FormatInt(time.Now().Unix(), 10), "1.0"
}

// generateDeviceConfig generates configuration for discovered device
func (d *DeviceDiscovery) generateDeviceConfig(deviceType, ip string, port int) map[string]string {
	config := map[string]string{
		"ip":         ip,
		"port":       strconv.Itoa(port),
		"connection": "tcp",
		"timeout":    "30",
	}

	// Add device-specific config
	switch deviceType {
	case "zkteco":
		config["comm_password"] = "0"
		config["encoding"] = "utf-8"
	case "essl":
		config["username"] = "admin"
		config["password"] = "admin"
	case "realtime":
		config["device_id"] = "1"
		config["comm_key"] = "0"
	}

	return config
}

// GenerateAdapterConfig generates adapter configuration from discovered devices
func (d *DeviceDiscovery) GenerateAdapterConfig() map[string]interface{} {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	adapters := make(map[string]interface{})
	enabledAdapters := []string{}

	for key, device := range d.devices {
		if device.Status != "online" {
			continue
		}

		adapterName := fmt.Sprintf("%s_%s_%d", device.Type, strings.ReplaceAll(device.IP, ".", "_"), device.Port)
		
		adapters[adapterName] = map[string]interface{}{
			"device_type":   device.Type,
			"connection":    "tcp",
			"device_config": device.Config,
			"sync_interval": 10, // Poll every 10 seconds
		}

		enabledAdapters = append(enabledAdapters, adapterName)
	}

	return map[string]interface{}{
		"enabled_adapters": enabledAdapters,
		"adapter_configs":  adapters,
	}
}