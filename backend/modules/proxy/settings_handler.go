package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// SettingsHandler 代理设置处理器
type SettingsHandler struct {
	dataDir      string
	settings     *ProxySettings
	mu           sync.RWMutex
	proxyService *Service // 代理服务引用，用于同步配置
}

// NewSettingsHandler 创建设置处理器
func NewSettingsHandler(dataDir string) *SettingsHandler {
	h := &SettingsHandler{
		dataDir:  dataDir,
		settings: GetDefaultProxySettings(),
	}
	// 加载已保存的设置
	h.loadSettings()
	return h
}

// SetProxyService 设置代理服务引用
func (h *SettingsHandler) SetProxyService(s *Service) {
	h.proxyService = s
	if err := h.syncProxyServiceSettings(h.GetCurrentSettings()); err != nil {
		fmt.Printf("同步代理设置失败: %v\n", err)
	}
}

func (h *SettingsHandler) syncProxyServiceSettings(settings *ProxySettings) error {
	if h.proxyService == nil || settings == nil {
		return nil
	}

	transparentMode := "off"
	if settings.TUN.Enable {
		transparentMode = "tun"
	}

	return h.proxyService.PatchConfig(map[string]interface{}{
		"mixedPort":       float64(settings.MixedPort),
		"socksPort":       float64(settings.SocksPort),
		"redirPort":       float64(settings.RedirPort),
		"tproxyPort":      float64(settings.TProxyPort),
		"allowLan":        settings.AllowLan,
		"ipv6":            settings.IPv6,
		"mode":            settings.Mode,
		"logLevel":        settings.LogLevel,
		"tunEnabled":      settings.TUN.Enable,
		"tunStack":        settings.TUN.Stack,
		"transparentMode": transparentMode,
		"autoStart":       settings.AutoStart,
		"autoStartDelay":  float64(settings.AutoStartDelay),
	})
}

// RegisterRoutes 注册路由
func (h *SettingsHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/settings", h.GetSettings)
	r.PUT("/settings", h.UpdateSettings)
	r.POST("/settings/reset", h.ResetSettings)
	r.GET("/settings/presets", h.GetPresets)
	r.POST("/settings/apply-preset", h.ApplyPreset)
}

// settingsFilePath 获取设置文件路径
func (h *SettingsHandler) settingsFilePath() string {
	return filepath.Join(h.dataDir, "proxy_settings.yaml")
}

// loadSettings 加载设置
func (h *SettingsHandler) loadSettings() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, err := os.ReadFile(h.settingsFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，使用默认设置
			return nil
		}
		return err
	}

	var settings ProxySettings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return err
	}

	// 设置默认值（旧配置文件可能缺少新字段）
	if settings.AutoStartDelay == 0 {
		settings.AutoStartDelay = 15 // 默认延迟 15 秒
	}
	normalizeProxySettings(&settings)

	h.settings = &settings
	return nil
}

func normalizeProxySettings(settings *ProxySettings) {
	if settings == nil {
		return
	}
	defaults := GetDefaultProxySettings()
	if settings.DNS.Listen == "" {
		settings.DNS = defaults.DNS
	}
	if settings.TUN.Device == "" {
		settings.TUN = defaults.TUN
	} else {
		if settings.TUN.Stack == "" {
			settings.TUN.Stack = defaults.TUN.Stack
		}
		if settings.TUN.MTU == 0 {
			settings.TUN.MTU = defaults.TUN.MTU
		}
		if settings.TUN.UDPTimeout == 0 {
			settings.TUN.UDPTimeout = defaults.TUN.UDPTimeout
		}
		if len(settings.TUN.DNSHijack) == 0 {
			settings.TUN.DNSHijack = defaults.TUN.DNSHijack
		}
		if len(settings.TUN.Inet4Address) == 0 {
			settings.TUN.Inet4Address = defaults.TUN.Inet4Address
		}
		if len(settings.TUN.RouteExcludeAddress) == 0 {
			settings.TUN.RouteExcludeAddress = defaults.TUN.RouteExcludeAddress
		}
	}
	settings.TUN.RouteExcludeAddress = filterDockerSubnets(settings.TUN.RouteExcludeAddress)
	if settings.TUN.Enable && settings.DNS.Listen == "0.0.0.0:53" {
		settings.DNS.Listen = defaults.DNS.Listen
	}
}

func filterDockerSubnets(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if value == "172.16.0.0/12" {
			continue
		}
		filtered = append(filtered, value)
	}
	return filtered
}

// saveSettings 保存设置
func (h *SettingsHandler) saveSettings() error {
	data, err := yaml.Marshal(h.settings)
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(h.settingsFilePath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(h.settingsFilePath(), data, 0644)
}

// GetSettings 获取当前设置
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": h.settings,
	})
}

// UpdateSettings 更新设置
func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	var settings ProxySettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "Invalid settings: " + err.Error(),
		})
		return
	}
	normalizeProxySettings(&settings)

	h.mu.Lock()
	h.settings = &settings
	err := h.saveSettings()
	h.mu.Unlock()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "Failed to save settings: " + err.Error(),
		})
		return
	}

	// 同步到 proxy 服务
	if h.proxyService != nil {
		if err := h.syncProxyServiceSettings(&settings); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    1,
				"message": "Failed to sync runtime settings: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Settings updated successfully",
	})
}

// ResetSettings 重置为默认设置
func (h *SettingsHandler) ResetSettings(c *gin.Context) {
	h.mu.Lock()
	h.settings = GetDefaultProxySettings()
	err := h.saveSettings()
	h.mu.Unlock()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "Failed to save settings: " + err.Error(),
		})
		return
	}

	if h.proxyService != nil {
		defaults := h.settings
		if err := h.syncProxyServiceSettings(defaults); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    1,
				"message": "Failed to sync runtime settings: " + err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Settings reset to defaults",
		"data":    h.settings,
	})
}

// SettingsPreset 设置预设
type SettingsPreset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// GetPresets 获取预设列表
func (h *SettingsHandler) GetPresets(c *gin.Context) {
	presets := []SettingsPreset{
		{
			ID:          "gateway",
			Name:        "Linux 网关",
			Description: "软路由/旁路由模式，TUN+DNS 最佳性能",
			Icon:        "server",
		},
		{
			ID:          "desktop",
			Name:        "桌面客户端",
			Description: "Windows/macOS 系统代理模式",
			Icon:        "monitor",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": presets,
	})
}

// ApplyPreset 应用预设
func (h *SettingsHandler) ApplyPreset(c *gin.Context) {
	var req struct {
		PresetID string `json:"presetId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "Invalid request",
		})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	switch req.PresetID {
	case "gateway":
		h.settings = getGatewayPreset()
	case "desktop":
		h.settings = getDesktopPreset()
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "Unknown preset: " + req.PresetID,
		})
		return
	}

	if err := h.saveSettings(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "Failed to save settings",
		})
		return
	}
	if err := h.syncProxyServiceSettings(h.settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": "Failed to sync runtime settings: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Preset applied successfully",
		"data":    h.settings,
	})
}

// GetCurrentSettings 获取当前设置（供其他模块调用）
func (h *SettingsHandler) GetCurrentSettings() *ProxySettings {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 返回副本
	data, _ := json.Marshal(h.settings)
	var copy ProxySettings
	json.Unmarshal(data, &copy)
	return &copy
}

// === 预设配置 ===

// getGatewayPreset Linux 网关预设
func getGatewayPreset() *ProxySettings {
	s := GetDefaultProxySettings()
	// 网关模式优化
	s.AllowLan = true
	s.BindAddress = "*"
	s.FindProcessMode = "off" // 关闭进程匹配
	s.TCPConcurrent = true
	s.UnifiedDelay = true
	s.GeodataMode = true
	s.GeodataLoader = "standard"

	// TUN 模式
	s.TUN.Enable = true
	s.TUN.Stack = "mixed"
	s.TUN.MTU = 9000
	s.TUN.GSO = true
	s.TUN.AutoRoute = false
	s.TUN.AutoRedirect = false
	s.TUN.StrictRoute = false
	s.TUN.EndpointIndependentNat = true
	s.TUN.Inet4Address = []string{"198.18.0.1/16"}
	s.TUN.RouteExcludeAddress = []string{
		"192.168.0.0/16",
		"10.0.0.0/8",
		"127.0.0.0/8",
		"fc00::/7",
		"fe80::/10",
	}

	// DNS
	s.DNS.Enable = true
	s.DNS.Listen = "0.0.0.0:1053"
	s.DNS.EnhancedMode = "fake-ip"
	s.DNS.RespectRules = true

	return s
}

// getDesktopPreset 桌面客户端预设
func getDesktopPreset() *ProxySettings {
	s := GetDefaultProxySettings()
	s.AllowLan = false
	s.BindAddress = "127.0.0.1"
	s.FindProcessMode = "strict"
	s.TCPConcurrent = true
	s.UnifiedDelay = true

	// TUN 关闭（使用系统代理）
	s.TUN.Enable = false

	// DNS
	s.DNS.Enable = true
	s.DNS.Listen = "127.0.0.1:1053"
	s.DNS.EnhancedMode = "fake-ip"

	return s
}
