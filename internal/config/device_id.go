package config

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// HardwareFingerprint 硬件指纹信息
type HardwareFingerprint struct {
	MACAddresses []string `json:"mac_addresses"` // 所有网卡 MAC 地址（排序后）
	CPUInfo      string   `json:"cpu_info"`      // CPU 信息
	MotherboardID string  `json:"motherboard_id"` // 主板序列号
	DiskSerial   string   `json:"disk_serial"`   // 硬盘序列号
	Hostname     string   `json:"hostname"`      // 主机名
	OS           string   `json:"os"`            // 操作系统
}

// DeviceIDInfo 设备 ID 完整信息
type DeviceIDInfo struct {
	DeviceID            string              `json:"device_id"`             // 设备 ID (DEV-XXXXXXXXXXXX)
	HardwareFingerprint HardwareFingerprint `json:"hardware_fingerprint"`  // 硬件指纹
	PersistLocations    []string            `json:"persist_locations"`     // 持久化位置
	GeneratedAt         string              `json:"generated_at"`          // 生成时间
}

// GenerateEnhancedDeviceID 生成增强型设备 ID
// 返回：设备 ID、硬件指纹、错误
func GenerateEnhancedDeviceID() (string, *HardwareFingerprint, error) {
	fingerprint := collectHardwareFingerprint()
	deviceID := generateDeviceIDFromFingerprint(fingerprint)
	return deviceID, fingerprint, nil
}

// collectHardwareFingerprint 收集硬件指纹信息
func collectHardwareFingerprint() *HardwareFingerprint {
	fp := &HardwareFingerprint{
		OS: runtime.GOOS,
	}

	// 1. 收集所有 MAC 地址
	fp.MACAddresses = collectMACAddresses()

	// 2. 收集 CPU 信息
	fp.CPUInfo = collectCPUInfo()

	// 3. 收集主板信息
	fp.MotherboardID = collectMotherboardID()

	// 4. 收集硬盘序列号
	fp.DiskSerial = collectDiskSerial()

	// 5. 主机名
	hostname, _ := os.Hostname()
	fp.Hostname = hostname

	return fp
}

// collectMACAddresses 收集所有网卡 MAC 地址（排序）
func collectMACAddresses() []string {
	var macs []string
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range interfaces {
			// 排除回环地址和无效地址
			if iface.Flags&net.FlagLoopback == 0 && iface.HardwareAddr != nil {
				addr := iface.HardwareAddr.String()
				if addr != "" && addr != "00:00:00:00:00:00" {
					macs = append(macs, addr)
				}
			}
		}
	}
	// 排序确保稳定性
	sort.Strings(macs)
	return macs
}

// collectCPUInfo 收集 CPU 信息
func collectCPUInfo() string {
	switch runtime.GOOS {
	case "windows":
		// Windows: 使用 wmic 获取 CPU ProcessorId
		cmd := exec.Command("wmic", "cpu", "get", "ProcessorId")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) > 1 {
				return strings.TrimSpace(lines[1])
			}
		}
	case "linux":
		// Linux: 读取 /proc/cpuinfo
		data, err := os.ReadFile("/proc/cpuinfo")
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "processor") {
					return strings.TrimSpace(strings.Split(line, ":")[1])
				}
			}
		}
	case "darwin":
		// macOS: 使用 sysctl
		cmd := exec.Command("sysctl", "-n", "machdep.cpu.brand_string")
		output, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(output))
		}
	}
	return ""
}

// collectMotherboardID 收集主板序列号
func collectMotherboardID() string {
	switch runtime.GOOS {
	case "windows":
		// Windows: 使用 wmic 获取主板序列号
		cmd := exec.Command("wmic", "baseboard", "get", "SerialNumber")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) > 1 {
				serial := strings.TrimSpace(lines[1])
				// 过滤无效值
				if serial != "" && serial != "To Be Filled By O.E.M." && serial != "Default string" {
					return serial
				}
			}
		}
	case "linux":
		// Linux: 读取 DMI 信息
		data, err := os.ReadFile("/sys/class/dmi/id/board_serial")
		if err == nil {
			serial := strings.TrimSpace(string(data))
			if serial != "" && serial != "To Be Filled By O.E.M." {
				return serial
			}
		}
	case "darwin":
		// macOS: 使用 ioreg
		cmd := exec.Command("ioreg", "-l")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "IOPlatformSerialNumber") {
					parts := strings.Split(line, "=")
					if len(parts) > 1 {
						return strings.Trim(strings.TrimSpace(parts[1]), "\"")
					}
				}
			}
		}
	}
	return ""
}

// collectDiskSerial 收集硬盘序列号
func collectDiskSerial() string {
	switch runtime.GOOS {
	case "windows":
		// Windows: 使用 wmic 获取物理磁盘序列号
		cmd := exec.Command("wmic", "diskdrive", "get", "SerialNumber")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) > 1 {
				return strings.TrimSpace(lines[1])
			}
		}
	case "linux":
		// Linux: 读取 /sys/block/sda/device/serial
		data, err := os.ReadFile("/sys/block/sda/device/serial")
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	case "darwin":
		// macOS: 使用 diskutil
		cmd := exec.Command("diskutil", "info", "disk0")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "Volume UUID") {
					parts := strings.Split(line, ":")
					if len(parts) > 1 {
						return strings.TrimSpace(parts[1])
					}
				}
			}
		}
	}
	return ""
}

// generateDeviceIDFromFingerprint 从硬件指纹生成设备 ID
func generateDeviceIDFromFingerprint(fp *HardwareFingerprint) string {
	// 组合所有硬件信息
	components := []string{
		strings.Join(fp.MACAddresses, ","),
		fp.CPUInfo,
		fp.MotherboardID,
		fp.DiskSerial,
		fp.Hostname,
		fp.OS,
	}

	// 过滤空值
	var validComponents []string
	for _, comp := range components {
		if comp != "" {
			validComponents = append(validComponents, comp)
		}
	}

	// 生成 SHA256 哈希
	combined := strings.Join(validComponents, "|")
	hash := sha256.Sum256([]byte(combined))

	// 使用 Base64 编码（URL 安全）并截取前 16 个字符
	encoded := base64.RawURLEncoding.EncodeToString(hash[:])
	if len(encoded) > 16 {
		encoded = encoded[:16]
	}

	return fmt.Sprintf("DEV-%s", encoded)
}

// LoadOrGenerateDeviceID 加载或生成设备 ID
// 优先级：配置文件 > 系统目录 > 生成新 ID
func LoadOrGenerateDeviceID() (string, *HardwareFingerprint, error) {
	// 1. 尝试从配置文件加载
	configDeviceID := loadDeviceIDFromConfig()
	if configDeviceID != "" {
		// 验证设备 ID 格式
		if isValidDeviceID(configDeviceID) {
			// 加载硬件指纹（如果存在）
			fp := loadHardwareFingerprintFromConfig()
			return configDeviceID, fp, nil
		}
	}

	// 2. 尝试从系统目录加载
	systemDeviceID, systemFP := loadDeviceIDFromSystem()
	if systemDeviceID != "" && isValidDeviceID(systemDeviceID) {
		// 同步到配置文件
		saveDeviceIDToConfig(systemDeviceID, systemFP)
		return systemDeviceID, systemFP, nil
	}

	// 3. 生成新的设备 ID
	deviceID, fp, err := GenerateEnhancedDeviceID()
	if err != nil {
		return "", nil, err
	}

	// 4. 持久化到配置文件和系统目录
	saveDeviceIDToConfig(deviceID, fp)
	saveDeviceIDToSystem(deviceID, fp)

	return deviceID, fp, nil
}

// isValidDeviceID 验证设备 ID 格式
func isValidDeviceID(deviceID string) bool {
	// 新格式：DEV-XXXXXXXXXXXX (16字符)
	if strings.HasPrefix(deviceID, "DEV-") && len(deviceID) == 20 {
		return true
	}
	// 兼容旧格式：MAC-xxxxxxxx 或 HOST-xxxxxxxx
	if (strings.HasPrefix(deviceID, "MAC-") || strings.HasPrefix(deviceID, "HOST-")) && len(deviceID) >= 12 {
		return true
	}
	return false
}

// loadDeviceIDFromConfig 从配置文件加载设备 ID
func loadDeviceIDFromConfig() string {
	// 这个函数会在 config.go 中调用，此时 viper 已经初始化
	// 直接返回空，让 config.go 处理
	return ""
}

// loadHardwareFingerprintFromConfig 从配置文件加载硬件指纹
func loadHardwareFingerprintFromConfig() *HardwareFingerprint {
	// 尝试从配置目录读取硬件指纹文件
	configDir := getConfigDir()
	fpFile := filepath.Join(configDir, "hardware_fingerprint.json")

	data, err := os.ReadFile(fpFile)
	if err != nil {
		return nil
	}

	var fp HardwareFingerprint
	if err := json.Unmarshal(data, &fp); err != nil {
		return nil
	}

	return &fp
}

// loadDeviceIDFromSystem 从系统目录加载设备 ID
func loadDeviceIDFromSystem() (string, *HardwareFingerprint) {
	systemDir := getSystemDeviceDir()
	if systemDir == "" {
		return "", nil
	}

	// 读取设备 ID 文件
	deviceIDFile := filepath.Join(systemDir, "device_id")
	data, err := os.ReadFile(deviceIDFile)
	if err != nil {
		return "", nil
	}

	deviceID := strings.TrimSpace(string(data))

	// 读取硬件指纹文件
	fpFile := filepath.Join(systemDir, "hardware_fingerprint.json")
	fpData, err := os.ReadFile(fpFile)
	if err != nil {
		return deviceID, nil
	}

	var fp HardwareFingerprint
	if err := json.Unmarshal(fpData, &fp); err != nil {
		return deviceID, nil
	}

	return deviceID, &fp
}

// saveDeviceIDToConfig 保存设备 ID 到配置文件
func saveDeviceIDToConfig(deviceID string, fp *HardwareFingerprint) error {
	// 保存硬件指纹到配置目录
	configDir := getConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	fpFile := filepath.Join(configDir, "hardware_fingerprint.json")
	fpData, err := json.MarshalIndent(fp, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fpFile, fpData, 0644)
}

// saveDeviceIDToSystem 保存设备 ID 到系统目录
func saveDeviceIDToSystem(deviceID string, fp *HardwareFingerprint) error {
	systemDir := getSystemDeviceDir()
	if systemDir == "" {
		return nil // 系统目录不可用，跳过
	}

	// 创建目录
	if err := os.MkdirAll(systemDir, 0755); err != nil {
		return err
	}

	// 保存设备 ID
	deviceIDFile := filepath.Join(systemDir, "device_id")
	if err := os.WriteFile(deviceIDFile, []byte(deviceID), 0644); err != nil {
		return err
	}

	// 保存硬件指纹
	fpFile := filepath.Join(systemDir, "hardware_fingerprint.json")
	fpData, err := json.MarshalIndent(fp, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(fpFile, fpData, 0644)
}

// getConfigDir 获取配置目录
func getConfigDir() string {
	// 优先使用当前目录
	return "."
}

// getSystemDeviceDir 获取系统设备目录
func getSystemDeviceDir() string {
	switch runtime.GOOS {
	case "windows":
		// Windows: %ProgramData%\wx_channel
		programData := os.Getenv("ProgramData")
		if programData != "" {
			return filepath.Join(programData, "wx_channel")
		}
	case "linux", "darwin":
		// Linux/macOS: /var/lib/wx_channel
		return "/var/lib/wx_channel"
	}
	return ""
}

// CalculateFingerprintSimilarity 计算两个硬件指纹的相似度（0-100）
func CalculateFingerprintSimilarity(fp1, fp2 *HardwareFingerprint) int {
	if fp1 == nil || fp2 == nil {
		return 0
	}

	score := 0
	maxScore := 0

	// MAC 地址匹配（权重：30）
	maxScore += 30
	if len(fp1.MACAddresses) > 0 && len(fp2.MACAddresses) > 0 {
		matchCount := 0
		for _, mac1 := range fp1.MACAddresses {
			for _, mac2 := range fp2.MACAddresses {
				if mac1 == mac2 {
					matchCount++
					break
				}
			}
		}
		score += (matchCount * 30) / len(fp1.MACAddresses)
	}

	// CPU 信息匹配（权重：25）
	maxScore += 25
	if fp1.CPUInfo != "" && fp2.CPUInfo != "" && fp1.CPUInfo == fp2.CPUInfo {
		score += 25
	}

	// 主板序列号匹配（权重：25）
	maxScore += 25
	if fp1.MotherboardID != "" && fp2.MotherboardID != "" && fp1.MotherboardID == fp2.MotherboardID {
		score += 25
	}

	// 硬盘序列号匹配（权重：20）
	maxScore += 20
	if fp1.DiskSerial != "" && fp2.DiskSerial != "" && fp1.DiskSerial == fp2.DiskSerial {
		score += 20
	}

	// 计算百分比
	if maxScore == 0 {
		return 0
	}
	return (score * 100) / maxScore
}
