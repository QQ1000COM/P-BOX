package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// MihomoConfig Mihomo/Clash 配置结构
type MihomoConfig struct {
	// 基础配置
	MixedPort          int    `yaml:"mixed-port,omitempty"`
	Port               int    `yaml:"port,omitempty"`
	SocksPort          int    `yaml:"socks-port,omitempty"`
	RedirPort          int    `yaml:"redir-port,omitempty"`
	TProxyPort         int    `yaml:"tproxy-port,omitempty"`
	AllowLan           bool   `yaml:"allow-lan"`
	BindAddress        string `yaml:"bind-address,omitempty"`
	Mode               string `yaml:"mode"`
	LogLevel           string `yaml:"log-level"`
	IPv6               bool   `yaml:"ipv6"`
	ExternalController string `yaml:"external-controller"`
	Secret             string `yaml:"secret,omitempty"`

	// 高级配置
	UnifiedDelay       bool     `yaml:"unified-delay,omitempty"`
	TCPConcurrent      bool     `yaml:"tcp-concurrent,omitempty"`
	FindProcessMode    string   `yaml:"find-process-mode,omitempty"`
	GlobalClientFinger string   `yaml:"global-client-fingerprint,omitempty"`
	GeodataMode        bool     `yaml:"geodata-mode,omitempty"`
	GeodataLoader      string   `yaml:"geodata-loader,omitempty"`
	GeositeMatcher     string   `yaml:"geosite-matcher,omitempty"` // succinct: 高效匹配器
	GeoAutoUpdate      bool     `yaml:"geo-auto-update,omitempty"`
	GeoUpdateInterval  int      `yaml:"geo-update-interval,omitempty"`
	GeoxURL            *GeoxURL `yaml:"geox-url,omitempty"`
	GlobalUA           string   `yaml:"global-ua,omitempty"`    // 下载外部资源的 UA
	ETagSupport        bool     `yaml:"etag-support,omitempty"` // ETag 缓存支持

	// TCP Keep-Alive 配置 (降低移动设备功耗)
	KeepAliveInterval int  `yaml:"keep-alive-interval,omitempty"`
	KeepAliveIdle     int  `yaml:"keep-alive-idle,omitempty"`
	DisableKeepAlive  bool `yaml:"disable-keep-alive,omitempty"` // 完全禁用 (省电模式)

	// 模块配置
	Profile *ProfileConfig `yaml:"profile,omitempty"`
	DNS     *DNSConfig     `yaml:"dns,omitempty"`
	TUN     *TUNConfig     `yaml:"tun,omitempty"`
	Sniffer *SnifferConfig `yaml:"sniffer,omitempty"`

	// 代理配置
	Proxies       []map[string]interface{} `yaml:"proxies"`
	ProxyGroups   []ProxyGroup             `yaml:"proxy-groups"`
	RuleProviders map[string]RuleProvider  `yaml:"rule-providers,omitempty"`
	Rules         []string                 `yaml:"rules"`
}

// GeoxURL GEO 数据源
type GeoxURL struct {
	GeoIP   string `yaml:"geoip,omitempty"`
	GeoSite string `yaml:"geosite,omitempty"`
	MMDB    string `yaml:"mmdb,omitempty"`
	ASN     string `yaml:"asn,omitempty"`
}

// ProfileConfig 缓存配置
type ProfileConfig struct {
	StoreSelected bool `yaml:"store-selected,omitempty"`
	StoreFakeIP   bool `yaml:"store-fake-ip,omitempty"`
}

// SnifferConfig 嗅探配置
type SnifferConfig struct {
	Enable          bool                     `yaml:"enable"`
	ForceDNSMapping bool                     `yaml:"force-dns-mapping,omitempty"` // 对 redir-host 强制嗅探
	ParsePureIP     bool                     `yaml:"parse-pure-ip,omitempty"`
	OverrideDest    bool                     `yaml:"override-destination,omitempty"`
	Sniff           map[string]SniffProtocol `yaml:"sniff,omitempty"`
	SkipDomain      []string                 `yaml:"skip-domain,omitempty"` // 跳过嗅探的域名
}

type SniffProtocol struct {
	Ports []interface{} `yaml:"ports,omitempty"`
}

// RuleProvider 规则提供者
type RuleProvider struct {
	Type     string `yaml:"type"`
	Behavior string `yaml:"behavior"`
	URL      string `yaml:"url"`
	Path     string `yaml:"path"`
	Interval int    `yaml:"interval,omitempty"`
	Format   string `yaml:"format,omitempty"`
}

type DNSConfig struct {
	Enable                bool                `yaml:"enable"`
	PreferH3              bool                `yaml:"prefer-h3,omitempty"`
	CacheAlgorithm        string              `yaml:"cache-algorithm,omitempty"` // lru 或 arc
	Listen                string              `yaml:"listen,omitempty"`
	IPv6                  bool                `yaml:"ipv6"`
	UseHosts              bool                `yaml:"use-hosts,omitempty"`
	UseSystemHosts        bool                `yaml:"use-system-hosts,omitempty"`
	EnhancedMode          string              `yaml:"enhanced-mode,omitempty"`
	FakeIPRange           string              `yaml:"fake-ip-range,omitempty"`
	FakeIPFilter          []string            `yaml:"fake-ip-filter,omitempty"`
	RespectRules          bool                `yaml:"respect-rules,omitempty"`
	DefaultNameserver     []string            `yaml:"default-nameserver,omitempty"`
	ProxyServerNameserver []string            `yaml:"proxy-server-nameserver,omitempty"`
	DirectNameserver      []string            `yaml:"direct-nameserver,omitempty"` // 直连出口 DNS
	Nameserver            []string            `yaml:"nameserver,omitempty"`
	Fallback              []string            `yaml:"fallback,omitempty"`
	FallbackFilter        *FallbackFilter     `yaml:"fallback-filter,omitempty"`
	NameserverPolicy      map[string][]string `yaml:"nameserver-policy,omitempty"`
}

type FallbackFilter struct {
	GeoIP     bool     `yaml:"geoip"`
	GeoIPCode string   `yaml:"geoip-code,omitempty"`
	IPCidr    []string `yaml:"ipcidr,omitempty"`
	Domain    []string `yaml:"domain,omitempty"`
}

type TUNConfig struct {
	Enable              bool     `yaml:"enable"`
	Device              string   `yaml:"device,omitempty"`
	Stack               string   `yaml:"stack,omitempty"`
	DNSHijack           []string `yaml:"dns-hijack,omitempty"`
	Inet4Address        []string `yaml:"inet4-address,omitempty"`
	AutoRoute           bool     `yaml:"auto-route"`
	AutoRedirect        bool     `yaml:"auto-redirect,omitempty"`
	AutoDetectInterface bool     `yaml:"auto-detect-interface"`
	StrictRoute         bool     `yaml:"strict-route,omitempty"`
	MTU                 int      `yaml:"mtu,omitempty"`
	UDPTimeout          int      `yaml:"udp-timeout,omitempty"`

	// GSO 通用分段卸载 (仅 Linux，提升吞吐量)
	GSO        bool `yaml:"gso,omitempty"`
	GSOMaxSize int  `yaml:"gso-max-size,omitempty"`

	// 路由地址 (不配置则使用默认路由 0.0.0.0/0)
	RouteAddress        []string `yaml:"route-address,omitempty"`
	RouteExcludeAddress []string `yaml:"route-exclude-address,omitempty"` // 排除的地址，如局域网

	// Linux 专用
	Iproute2TableIndex     int  `yaml:"iproute2-table-index,omitempty"`
	Iproute2RuleIndex      int  `yaml:"iproute2-rule-index,omitempty"`
	EndpointIndependentNat bool `yaml:"endpoint-independent-nat,omitempty"`
}

type ProxyGroup struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Proxies  []string `yaml:"proxies"`
	URL      string   `yaml:"url,omitempty"`
	Interval int      `yaml:"interval,omitempty"`
}

// ProxyNode 代理节点
type ProxyNode struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Server     string `json:"server"`
	Port       int    `json:"port"`
	ServerPort int    `json:"serverPort"` // 兼容 node 模块的字段名
	Config     string `json:"config"`     // JSON 格式的完整配置
	IsManual   bool   `json:"isManual"`   // 是否手动添加的节点
}

// GetPort 获取端口（兼容两种字段名）
func (n *ProxyNode) GetPort() int {
	if n.Port > 0 {
		return n.Port
	}
	return n.ServerPort
}

// ConfigGeneratorOptions 配置生成选项
type ConfigGeneratorOptions struct {
	// 基础设置
	MixedPort int    `json:"mixedPort"`
	AllowLan  bool   `json:"allowLan"`
	Mode      string `json:"mode"` // rule, global, direct
	LogLevel  string `json:"logLevel"`
	IPv6      bool   `json:"ipv6"`

	// 透明代理
	EnableTProxy bool `json:"enableTProxy"`
	TProxyPort   int  `json:"tproxyPort"`

	// TUN 模式
	EnableTUN bool `json:"enableTun"`

	// DNS 设置
	EnableDNS    bool     `json:"enableDns"`
	DNSListen    string   `json:"dnsListen"`
	EnhancedMode string   `json:"enhancedMode"` // fake-ip, redir-host
	Nameservers  []string `json:"nameservers"`
	Fallback     []string `json:"fallback"`

	// API
	ExternalController string `json:"externalController"`
	Secret             string `json:"secret"`

	// 性能优化设置（从 ProxySettings 读取）
	UnifiedDelay            bool   `json:"unifiedDelay"`
	TCPConcurrent           bool   `json:"tcpConcurrent"`
	FindProcessMode         string `json:"findProcessMode"`
	GlobalClientFingerprint string `json:"globalClientFingerprint"`
	KeepAliveInterval       int    `json:"keepAliveInterval"`
	KeepAliveIdle           int    `json:"keepAliveIdle"`
	DisableKeepAlive        bool   `json:"disableKeepAlive"`

	// GEO 数据设置
	GeodataMode       bool   `json:"geodataMode"`
	GeodataLoader     string `json:"geodataLoader"`
	GeositeMatcher    string `json:"geositeMatcher"`
	GeoAutoUpdate     bool   `json:"geoAutoUpdate"`
	GeoUpdateInterval int    `json:"geoUpdateInterval"`
	GlobalUA          string `json:"globalUa"`
	ETagSupport       bool   `json:"etagSupport"`

	// TUN 设置
	TUNSettings *TUNSettings `json:"tunSettings"`

	// 配置模板（可选，为 nil 时使用默认生成）
	Template *ConfigTemplate `json:"-"`
}

// ConfigGenerator 配置生成器
type ConfigGenerator struct {
	dataDir string
}

func NewConfigGenerator(dataDir string) *ConfigGenerator {
	return &ConfigGenerator{dataDir: dataDir}
}

// helper 函数：布尔值默认值
func getOrDefault(val bool, def bool) bool {
	// bool 零值是 false，无法区分是否设置
	// 这里直接返回传入的值，默认值在 options 初始化时设置
	return val || def
}

// helper 函数：字符串默认值
func getOrDefaultStr(val string, def string) string {
	if val == "" {
		return def
	}
	return val
}

// helper 函数：整数默认值
func getOrDefaultInt(val int, def int) int {
	if val == 0 {
		return def
	}
	return val
}

// GenerateConfig 生成 Mihomo 配置
func (g *ConfigGenerator) GenerateConfig(nodes []ProxyNode, options ConfigGeneratorOptions) (*MihomoConfig, error) {
	if options.MixedPort == 0 {
		options.MixedPort = 7890
	}
	if options.Mode == "" {
		options.Mode = "rule"
	}
	if options.LogLevel == "" {
		options.LogLevel = "info"
	}
	if options.ExternalController == "" {
		options.ExternalController = "127.0.0.1:9090"
	}

	geoxURL, hasLocalGeox := g.getGeoxURL()

	config := &MihomoConfig{
		// 基础配置
		MixedPort:          options.MixedPort,
		AllowLan:           options.AllowLan,
		Mode:               options.Mode,
		LogLevel:           options.LogLevel,
		IPv6:               options.IPv6,
		ExternalController: options.ExternalController,
		Secret:             options.Secret,

		// 高级配置 (从代理设置读取)
		UnifiedDelay:      getOrDefault(options.UnifiedDelay, true),
		TCPConcurrent:     getOrDefault(options.TCPConcurrent, true),
		FindProcessMode:   getOrDefaultStr(options.FindProcessMode, "off"),
		GeodataMode:       getOrDefault(options.GeodataMode, true),
		GeodataLoader:     getOrDefaultStr(options.GeodataLoader, "standard"),
		GeositeMatcher:    getOrDefaultStr(options.GeositeMatcher, "succinct"),
		GeoAutoUpdate:     getOrDefault(options.GeoAutoUpdate, true) && !hasLocalGeox,
		GeoUpdateInterval: getOrDefaultInt(options.GeoUpdateInterval, 24),
		GlobalUA:          getOrDefaultStr(options.GlobalUA, "clash.meta"),
		ETagSupport:       getOrDefault(options.ETagSupport, true),

		// TCP Keep-Alive (从代理设置读取)
		KeepAliveInterval: getOrDefaultInt(options.KeepAliveInterval, 15),
		KeepAliveIdle:     getOrDefaultInt(options.KeepAliveIdle, 30),
		DisableKeepAlive:  options.DisableKeepAlive,

		// GEO 数据源
		GeoxURL: geoxURL,

		// 缓存配置
		Profile: &ProfileConfig{
			StoreSelected: true,
			StoreFakeIP:   true,
		},
	}

	// 透明代理端口 - 只有启用时才设置
	if options.EnableTProxy {
		if options.TProxyPort > 0 {
			config.TProxyPort = options.TProxyPort
		}
		// Redir 端口 (用于 iptables REDIRECT)
		config.RedirPort = 7892
	}
	// 系统代理模式不设置 redir-port 和 tproxy-port

	// DNS 配置
	config.DNS = g.generateDNSConfig(options)

	// TUN 配置 (从代理设置读取)
	if options.EnableTUN {
		tunSettings := options.TUNSettings
		// 默认 TUN 设置
		device := "p-box"
		stack := "mixed"
		mtu := 9000
		udpTimeout := 300
		gso := true
		gsoMaxSize := 65536
		strictRoute := false
		autoRoute := false
		autoRedirect := false
		autoDetectInterface := true
		endpointIndependentNat := true
		dnsHijack := []string{"any:53", "tcp://any:53"}
		inet4Address := []string{"198.18.0.1/16"}
		routeExcludeAddress := []string{
			"192.168.0.0/16", "10.0.0.0/8",
			"127.0.0.0/8", "fc00::/7", "fe80::/10",
		}

		// 从设置覆盖
		if tunSettings != nil {
			if tunSettings.Device != "" {
				device = tunSettings.Device
			}
			if tunSettings.Stack != "" {
				stack = tunSettings.Stack
			}
			if tunSettings.MTU > 0 {
				mtu = tunSettings.MTU
			}
			if tunSettings.UDPTimeout > 0 {
				udpTimeout = tunSettings.UDPTimeout
			}
			gso = tunSettings.GSO
			if tunSettings.GSOMaxSize > 0 {
				gsoMaxSize = tunSettings.GSOMaxSize
			}
			strictRoute = tunSettings.StrictRoute
			autoRoute = tunSettings.AutoRoute
			autoRedirect = tunSettings.AutoRedirect
			autoDetectInterface = tunSettings.AutoDetectInterface
			endpointIndependentNat = tunSettings.EndpointIndependentNat
			if len(tunSettings.DNSHijack) > 0 {
				dnsHijack = tunSettings.DNSHijack
			}
			if len(tunSettings.Inet4Address) > 0 {
				inet4Address = tunSettings.Inet4Address
			}
			if len(tunSettings.RouteExcludeAddress) > 0 {
				routeExcludeAddress = tunSettings.RouteExcludeAddress
			}
		}
		routeExcludeAddress = filterDockerSubnets(routeExcludeAddress)

		config.TUN = &TUNConfig{
			Enable:                 true,
			Device:                 device,
			Stack:                  stack,
			DNSHijack:              dnsHijack,
			Inet4Address:           inet4Address,
			AutoRoute:              autoRoute,
			AutoRedirect:           autoRedirect,
			AutoDetectInterface:    autoDetectInterface,
			StrictRoute:            strictRoute,
			MTU:                    mtu,
			UDPTimeout:             udpTimeout,
			GSO:                    gso,
			GSOMaxSize:             gsoMaxSize,
			EndpointIndependentNat: endpointIndependentNat,
			RouteExcludeAddress:    routeExcludeAddress,
		}
		if config.RedirPort == 0 {
			config.RedirPort = 7892
		}
		// TUN 模式下调整 DNS 配置
		if config.DNS != nil {
			if config.DNS.Listen == "" || config.DNS.Listen == "0.0.0.0:53" {
				config.DNS.Listen = "0.0.0.0:1053"
			}
			config.DNS.EnhancedMode = "fake-ip" // fake-ip 模式响应更快
		}
	}

	// 嗅探配置
	config.Sniffer = &SnifferConfig{
		Enable:          true,
		ForceDNSMapping: true, // 对 redir-host 强制嗅探
		ParsePureIP:     true,
		OverrideDest:    true,
		Sniff: map[string]SniffProtocol{
			"HTTP": {Ports: []interface{}{80, "8080-8880"}},
			"TLS":  {Ports: []interface{}{443, 8443}},
			"QUIC": {Ports: []interface{}{443, 8443}},
		},
		SkipDomain: []string{
			"+.push.apple.com", // 跳过苹果推送
		},
	}

	// 转换代理节点
	config.Proxies = g.convertProxies(nodes)

	// 生成代理组（始终使用模板，确保名称一致）
	template := options.Template
	if template == nil {
		template = GetDefaultConfigTemplate()
	}
	config.ProxyGroups = g.generateProxyGroupsFromTemplate(nodes, template.ProxyGroups)

	// 生成规则提供者
	config.RuleProviders = g.generateRuleProviders()

	// 生成规则（使用模板中的规则）
	config.Rules = g.generateRulesFromTemplate(template.Rules)

	return config, nil
}

// generateDNSConfig 生成 DNS 配置 (防止 DNS 泄漏 + 性能优化)
func (g *ConfigGenerator) generateDNSConfig(options ConfigGeneratorOptions) *DNSConfig {
	dns := &DNSConfig{
		Enable:         true,
		PreferH3:       true,  // 优先 HTTP/3，更快
		CacheAlgorithm: "arc", // ARC 缓存算法，命中率更高
		IPv6:           options.IPv6,
		UseHosts:       true,
		UseSystemHosts: false, // 不使用系统 hosts，防止泄漏
		EnhancedMode:   options.EnhancedMode,
		RespectRules:   true, // DNS 查询遵循代理规则，防止泄漏
	}

	if dns.EnhancedMode == "" {
		dns.EnhancedMode = "fake-ip"
	}

	if options.DNSListen != "" {
		dns.Listen = options.DNSListen
	} else {
		dns.Listen = "0.0.0.0:1053"
	}

	if dns.EnhancedMode == "fake-ip" {
		dns.FakeIPRange = "198.18.0.1/16"
		dns.FakeIPFilter = []string{
			// === 直连域名使用真实 IP (不使用 fake-ip) ===
			"geosite:cn",      // 国内域名直接返回真实 IP
			"geosite:private", // 私有域名

			// === 本地域名 ===
			"*.lan",
			"*.local",
			"*.localhost",
			"*.localdomain",
			"*.home.arpa",

			// === 网络检测 ===
			"+.msftconnecttest.com",
			"+.msftncsi.com",
			"connectivitycheck.gstatic.com",
			"captive.apple.com",
			"wifi.vivo.com.cn",
			"connect.rom.miui.com",

			// === NTP 时间同步 ===
			"time.*.com",
			"time.*.gov",
			"time.*.apple.com",
			"time.*.edu.cn",
			"ntp.*.com",
			"pool.ntp.org",

			// === STUN/NAT 穿透 ===
			"stun.*.*",
			"stun.*.*.*",
			"+.stun.playstation.net",
			"+.stun.xbox.com",
			"+.stun.l.google.com",

			// === 本地服务发现 ===
			"+._tcp.*",
			"+._udp.*",

			// === 国内常用服务 ===
			"localhost.ptlogin2.qq.com",
			"+.market.xiaomi.com",
			"+.qq.com",
			"+.tencent.com",
			"+.weixin.qq.com",
			"+.alipay.com",
			"+.taobao.com",
			"+.tmall.com",
			"+.jd.com",
			"+.baidu.com",
			"+.bilibili.com",
			"+.163.com",
			"+.126.com",
		}
	}

	// 默认 DNS (用于解析 DOH 域名) - 必须是 IP
	dns.DefaultNameserver = []string{
		"223.5.5.5",
		"119.29.29.29",
	}

	// 代理节点域名解析 - 使用国内 DNS (因为代理节点通常是国内购买的)
	dns.ProxyServerNameserver = []string{
		"223.5.5.5",    // 阿里 DNS (IP 直连，更快)
		"119.29.29.29", // 腾讯 DNS
		"https://doh.pub/dns-query",
	}

	// 直连出口 DNS - 用于直连流量的域名解析 (国内 DNS，更快)
	dns.DirectNameserver = []string{
		"223.5.5.5",
		"119.29.29.29",
		"https://doh.pub/dns-query",
	}

	// 主 DNS 服务器 - 默认使用海外 DNS（未匹配域名走代理查询）
	if len(options.Nameservers) > 0 {
		dns.Nameserver = options.Nameservers
	} else {
		dns.Nameserver = []string{
			"https://dns.google/dns-query",
			"https://cloudflare-dns.com/dns-query",
			"1.1.1.1", // Cloudflare DNS (IP，备用)
		}
	}

	// 后备 DNS - 海外 DNS，用于解析被污染的域名
	if len(options.Fallback) > 0 {
		dns.Fallback = options.Fallback
	} else {
		dns.Fallback = []string{
			"https://dns.google/dns-query",
			"https://cloudflare-dns.com/dns-query",
			"https://dns.quad9.net/dns-query",
		}
	}

	// 后备过滤 - 配置何时使用 fallback
	dns.FallbackFilter = &FallbackFilter{
		GeoIP:     true,
		GeoIPCode: "CN",
		IPCidr: []string{
			"240.0.0.0/4",  // 保留地址
			"0.0.0.0/32",   // 无效 IP
			"127.0.0.1/32", // 本地回环 (可能是污染)
		},
		Domain: []string{
			"+.google.com",
			"+.facebook.com",
			"+.youtube.com",
			"+.twitter.com",
			"+.googleapis.com",
			"+.gstatic.com",
			"+.github.com",
			"+.githubusercontent.com",
		},
	}

	// 域名策略 - 国内域名用国内 DNS，其他域名用默认的海外 DNS
	dns.NameserverPolicy = map[string][]string{
		// 国内域名使用国内 DNS（直连查询，不走代理）
		"geosite:cn": {
			"https://doh.pub/dns-query",
			"https://dns.alidns.com/dns-query",
		},
		// 私有域名使用国内 DNS
		"geosite:private": {
			"https://doh.pub/dns-query",
			"https://dns.alidns.com/dns-query",
		},
	}
	// 未匹配的域名会使用 nameserver（海外 DNS），配合 respect-rules 走代理查询

	return dns
}

// convertProxies 转换代理节点为 Clash/Mihomo 格式
func (g *ConfigGenerator) convertProxies(nodes []ProxyNode) []map[string]interface{} {
	var proxies []map[string]interface{}

	for _, node := range nodes {
		proxy := make(map[string]interface{})
		isCompleteConfig := false

		// 如果有完整配置（从 Clash YAML 解析来的），直接使用
		if node.Config != "" {
			if err := json.Unmarshal([]byte(node.Config), &proxy); err == nil {
				// 检查是否是完整的 Clash 配置（包含必要字段）
				if _, hasType := proxy["type"]; hasType {
					if _, hasServer := proxy["server"]; hasServer {
						// 确保端口是整数
						if port, ok := proxy["port"].(float64); ok {
							proxy["port"] = int(port)
						}
						isCompleteConfig = true
						// 不再 continue，继续执行类型特定的修复
					}
				}
			}
			// 如果解析失败或不完整，清空重新构建
			if !isCompleteConfig {
				proxy = make(map[string]interface{})
			}
		}

		// 构建基础配置（仅当不是完整配置时）
		if !isCompleteConfig {
			proxy["name"] = node.Name
			proxy["type"] = node.Type
			proxy["server"] = node.Server
			proxy["port"] = node.GetPort()
		}

		// 如果有额外配置，合并进去（仅当不是完整配置时）
		if !isCompleteConfig && node.Config != "" {
			var extraConfig map[string]interface{}
			if err := json.Unmarshal([]byte(node.Config), &extraConfig); err == nil {
				// 合并额外配置
				for k, v := range extraConfig {
					// 跳过基础字段，避免覆盖
					if k == "name" || k == "type" || k == "server" || k == "port" {
						continue
					}
					proxy[k] = v
				}
			}
		}

		// 确定实际的代理类型（优先使用 proxy 中的类型）
		proxyType := node.Type
		if pt, ok := proxy["type"].(string); ok && pt != "" {
			proxyType = pt
		}

		// 根据协议类型进行字段转换
		switch proxyType {
		case "hysteria2", "hy2":
			proxy["type"] = "hysteria2"
			// 处理 TLS 配置
			if tls, ok := proxy["tls"].(map[string]interface{}); ok {
				if sni, ok := tls["server_name"].(string); ok && sni != "" {
					proxy["sni"] = sni
				}
				if insecure, ok := tls["insecure"].(bool); ok {
					proxy["skip-cert-verify"] = insecure
				}
				if alpn, ok := tls["alpn"].([]interface{}); ok && len(alpn) > 0 {
					proxy["alpn"] = alpn
				}
				delete(proxy, "tls")
			}
			if sni, ok := proxy["server_name"].(string); ok {
				proxy["sni"] = sni
				delete(proxy, "server_name")
			}
			// 处理 obfs 配置 (sing-box 格式 -> Mihomo 格式)
			if obfs, ok := proxy["obfs"].(map[string]interface{}); ok {
				if obfsType, ok := obfs["type"].(string); ok && obfsType != "" {
					proxy["obfs"] = obfsType
				}
				if obfsPwd, ok := obfs["password"].(string); ok && obfsPwd != "" {
					proxy["obfs-password"] = obfsPwd
				}
			}
			// 处理带宽限制 (up_mbps/down_mbps -> up/down)
			if upMbps, ok := proxy["up_mbps"].(float64); ok && upMbps > 0 {
				proxy["up"] = fmt.Sprintf("%d Mbps", int(upMbps))
				delete(proxy, "up_mbps")
			}
			if downMbps, ok := proxy["down_mbps"].(float64); ok && downMbps > 0 {
				proxy["down"] = fmt.Sprintf("%d Mbps", int(downMbps))
				delete(proxy, "down_mbps")
			}
			// Hysteria2 默认启用 UDP
			if _, ok := proxy["udp"]; !ok {
				proxy["udp"] = true
			}

		case "hysteria", "hy":
			proxy["type"] = "hysteria"
			// Hysteria 默认启用 UDP 和 fast-open
			if _, ok := proxy["udp"]; !ok {
				proxy["udp"] = true
			}
			if _, ok := proxy["fast-open"]; !ok {
				proxy["fast-open"] = true
			}

		case "vless":
			// VLESS 需要 uuid
			if _, ok := proxy["uuid"]; !ok {
				if password, ok := proxy["password"].(string); ok {
					proxy["uuid"] = password
					delete(proxy, "password")
				}
			}
			// 默认启用 UDP
			if _, ok := proxy["udp"]; !ok {
				proxy["udp"] = true
			}
			// 客户端指纹
			if _, ok := proxy["client-fingerprint"]; !ok {
				proxy["client-fingerprint"] = "chrome"
			}
			// 删除 Mihomo 不需要的字段
			delete(proxy, "packet_encoding")
			delete(proxy, "encryption")

		case "vmess":
			// VMess 需要 uuid, alterId, cipher
			if _, ok := proxy["uuid"]; !ok {
				if password, ok := proxy["password"].(string); ok {
					proxy["uuid"] = password
					delete(proxy, "password")
				}
			}
			// alter_id -> alterId (订阅解析器使用下划线，Mihomo 使用驼峰)
			if alterId, ok := proxy["alter_id"]; ok {
				proxy["alterId"] = alterId
				delete(proxy, "alter_id")
			}
			if _, ok := proxy["alterId"]; !ok {
				proxy["alterId"] = 0
			}
			// security -> cipher (订阅解析器使用 security，Mihomo 使用 cipher)
			if security, ok := proxy["security"].(string); ok && security != "" {
				if _, exists := proxy["cipher"]; !exists {
					proxy["cipher"] = security
				}
				delete(proxy, "security")
			}
			if _, ok := proxy["cipher"]; !ok {
				proxy["cipher"] = "auto"
			}
			// 默认启用 UDP
			if _, ok := proxy["udp"]; !ok {
				proxy["udp"] = true
			}
			// 客户端指纹
			if _, ok := proxy["client-fingerprint"]; !ok {
				proxy["client-fingerprint"] = "chrome"
			}
			// 删除 Mihomo 不需要的字段
			delete(proxy, "global_padding")
			delete(proxy, "authenticated_length")
			delete(proxy, "packet_encoding")

		case "ss", "shadowsocks":
			proxy["type"] = "ss"
			// SS 必须有 cipher 字段，否则 Mihomo 会报错
			if _, ok := proxy["cipher"]; !ok {
				// 尝试从 method 字段获取（某些订阅使用 method 而不是 cipher）
				if method, ok := proxy["method"].(string); ok && method != "" {
					proxy["cipher"] = method
					delete(proxy, "method")
				} else {
					// 默认使用 aes-256-gcm（最常用的加密方式）
					proxy["cipher"] = "aes-256-gcm"
				}
			}
			// 处理 plugin_opts -> plugin-opts 字段名转换（兼容不同来源）
			if pluginOpts, ok := proxy["plugin_opts"]; ok {
				proxy["plugin-opts"] = pluginOpts
				delete(proxy, "plugin_opts")
			}
			// 默认启用 UDP
			if _, ok := proxy["udp"]; !ok {
				proxy["udp"] = true
			}

		case "ssr", "shadowsocksr":
			proxy["type"] = "ssr"
			// SSR 必须有 cipher 字段
			if _, ok := proxy["cipher"]; !ok {
				if method, ok := proxy["method"].(string); ok && method != "" {
					proxy["cipher"] = method
					delete(proxy, "method")
				} else {
					proxy["cipher"] = "aes-256-cfb"
				}
			}
			// SSR 必须有 obfs 字段
			if _, ok := proxy["obfs"]; !ok {
				proxy["obfs"] = "plain"
			}
			// SSR 必须有 protocol 字段
			if _, ok := proxy["protocol"]; !ok {
				proxy["protocol"] = "origin"
			}
			// 默认启用 UDP
			if _, ok := proxy["udp"]; !ok {
				proxy["udp"] = true
			}

		case "trojan":
			// Trojan 需要 password
			// 默认启用 UDP
			if _, ok := proxy["udp"]; !ok {
				proxy["udp"] = true
			}
			// 客户端指纹
			if _, ok := proxy["client-fingerprint"]; !ok {
				proxy["client-fingerprint"] = "chrome"
			}

		case "tuic":
			// TUIC 协议 - 字段名转换
			// udp_relay_mode -> udp-relay-mode
			if mode, ok := proxy["udp_relay_mode"].(string); ok {
				proxy["udp-relay-mode"] = mode
				delete(proxy, "udp_relay_mode")
			}
			if _, ok := proxy["udp-relay-mode"]; !ok {
				proxy["udp-relay-mode"] = "native" // native 或 quic
			}
			// congestion_control -> congestion-controller
			if cc, ok := proxy["congestion_control"].(string); ok {
				proxy["congestion-controller"] = cc
				delete(proxy, "congestion_control")
			}
			if _, ok := proxy["congestion-controller"]; !ok {
				proxy["congestion-controller"] = "bbr" // cubic, new_reno, bbr
			}
			// zero_rtt_handshake -> reduce-rtt
			if zeroRTT, ok := proxy["zero_rtt_handshake"].(bool); ok {
				proxy["reduce-rtt"] = zeroRTT
				delete(proxy, "zero_rtt_handshake")
			}
			if _, ok := proxy["reduce-rtt"]; !ok {
				proxy["reduce-rtt"] = true
			}
			// 默认启用 UDP
			if _, ok := proxy["udp"]; !ok {
				proxy["udp"] = true
			}

		case "anytls":
			// AnyTLS 协议 (官方文档: https://wiki.metacubex.one/en/config/proxies/anytls/)
			// 注意: AnyTLS 不需要显式 tls: true，TLS 是隐含的
			proxy["type"] = "anytls"
			// 默认启用 UDP
			if _, ok := proxy["udp"]; !ok {
				proxy["udp"] = true
			}
			// 处理 TLS 配置对象
			if tls, ok := proxy["tls"].(map[string]interface{}); ok {
				if sni, ok := tls["server_name"].(string); ok && sni != "" {
					proxy["sni"] = sni
				}
				if insecure, ok := tls["insecure"].(bool); ok {
					proxy["skip-cert-verify"] = insecure
				}
				// 提取 ALPN
				if alpn, ok := tls["alpn"].([]interface{}); ok && len(alpn) > 0 {
					proxy["alpn"] = alpn
				} else if alpn, ok := tls["alpn"].([]string); ok && len(alpn) > 0 {
					proxy["alpn"] = alpn
				}
				// 提取 UTLS fingerprint
				if utls, ok := tls["utls"].(map[string]interface{}); ok {
					if fp, ok := utls["fingerprint"].(string); ok && fp != "" {
						proxy["client-fingerprint"] = fp
					}
				}
				// 删除 tls 对象（AnyTLS 不需要 tls: true）
				delete(proxy, "tls")
			}
			// 客户端指纹（默认 chrome）
			if _, ok := proxy["client-fingerprint"]; !ok {
				if fp, ok := proxy["fingerprint"].(string); ok && fp != "" {
					proxy["client-fingerprint"] = fp
					delete(proxy, "fingerprint")
				} else {
					proxy["client-fingerprint"] = "chrome"
				}
			}
			// 处理 Reality 配置
			if reality, ok := proxy["reality"].(map[string]interface{}); ok {
				if enabled, ok := reality["enabled"].(bool); ok && enabled {
					realityOpts := make(map[string]interface{})
					if pubKey, ok := reality["public_key"].(string); ok {
						realityOpts["public-key"] = pubKey
					}
					if shortID, ok := reality["short_id"].(string); ok && shortID != "" {
						realityOpts["short-id"] = shortID
					}
					proxy["reality-opts"] = realityOpts
				}
				delete(proxy, "reality")
			}

		case "wireguard", "wg":
			proxy["type"] = "wireguard"
			// WireGuard 默认 UDP
			if _, ok := proxy["udp"]; !ok {
				proxy["udp"] = true
			}

		case "http", "https":
			// HTTP/HTTPS 代理

		case "socks5":
			// SOCKS5 代理
			if _, ok := proxy["udp"]; !ok {
				proxy["udp"] = true
			}
		}

		// 🔧 处理 Reality 配置
		if reality, ok := proxy["reality"].(map[string]interface{}); ok {
			if enabled, ok := reality["enabled"].(bool); ok && enabled {
				realityOpts := make(map[string]interface{})
				if pubKey, ok := reality["public_key"].(string); ok {
					realityOpts["public-key"] = pubKey
				}
				if shortID, ok := reality["short_id"].(string); ok && shortID != "" {
					realityOpts["short-id"] = shortID
				}
				proxy["reality-opts"] = realityOpts
				// Reality 也需要 tls: true
				proxy["tls"] = true
			}
			delete(proxy, "reality")
		}

		// 🔧 通用 TLS 字段转换（适用于所有协议）
		// 将订阅解析的 tls 对象转换为 Mihomo 需要的扁平字段
		if tls, ok := proxy["tls"].(map[string]interface{}); ok {
			// 记录是否需要启用 TLS
			tlsEnabled := false
			if enabled, ok := tls["enabled"].(bool); ok && enabled {
				tlsEnabled = true
			}

			// tls.server_name -> servername（VLESS/VMess 使用 servername）
			if sni, ok := tls["server_name"].(string); ok && sni != "" {
				if _, exists := proxy["servername"]; !exists {
					proxy["servername"] = sni
				}
			}

			// tls.insecure -> skip-cert-verify
			if insecure, ok := tls["insecure"].(bool); ok {
				proxy["skip-cert-verify"] = insecure
			}

			// tls.alpn -> alpn (支持 []interface{} 和 []string 两种类型)
			if _, exists := proxy["alpn"]; !exists {
				if alpn, ok := tls["alpn"].([]interface{}); ok && len(alpn) > 0 {
					proxy["alpn"] = alpn
				} else if alpn, ok := tls["alpn"].([]string); ok && len(alpn) > 0 {
					proxy["alpn"] = alpn
				}
			}

			// tls.fingerprint -> fingerprint
			if fp, ok := tls["fingerprint"].(string); ok && fp != "" {
				if _, exists := proxy["fingerprint"]; !exists {
					proxy["fingerprint"] = fp
				}
			}

			// tls.utls -> client-fingerprint
			if utls, ok := tls["utls"].(map[string]interface{}); ok {
				if fp, ok := utls["fingerprint"].(string); ok && fp != "" {
					if _, exists := proxy["client-fingerprint"]; !exists {
						proxy["client-fingerprint"] = fp
					}
				}
			}

			// 🔧 先删除 tls 对象，再设置 tls: true（修复顺序问题）
			delete(proxy, "tls")
			if tlsEnabled {
				proxy["tls"] = true
			}
		}

		// 🔧 server_name -> servername 的通用转换（VLESS/VMess 使用 servername）
		if sn, ok := proxy["server_name"].(string); ok && sn != "" {
			if _, exists := proxy["servername"]; !exists {
				proxy["servername"] = sn
			}
			delete(proxy, "server_name")
		}

		// 🔧 为需要 TLS 的协议默认启用 skip-cert-verify（提高兼容性）
		needsTLS := node.Type == "trojan" || node.Type == "vless" || node.Type == "vmess"
		if needsTLS {
			if _, exists := proxy["skip-cert-verify"]; !exists {
				proxy["skip-cert-verify"] = true // 默认跳过证书验证
			}
		}

		// 🔧 Transport 配置转换为 Mihomo 格式
		// 将 transport 对象转换为 network + xxx-opts 格式
		if transport, ok := proxy["transport"].(map[string]interface{}); ok {
			if netType, ok := transport["type"].(string); ok && netType != "" {
				proxy["network"] = netType

				switch netType {
				case "ws":
					wsOpts := make(map[string]interface{})
					if path, ok := transport["path"].(string); ok {
						wsOpts["path"] = path
					}
					if headers, ok := transport["headers"].(map[string]interface{}); ok {
						wsOpts["headers"] = headers
					}
					if len(wsOpts) > 0 {
						proxy["ws-opts"] = wsOpts
					}

				case "grpc":
					grpcOpts := make(map[string]interface{})
					// 尝试从 grpc_options 获取 (VMess/Trojan 格式)
					if grpcOptions, ok := transport["grpc_options"].(map[string]interface{}); ok {
						if sn, ok := grpcOptions["service_name"].(string); ok {
							grpcOpts["grpc-service-name"] = sn
						}
					}
					// 也支持直接在 transport 层级的 service_name (VLESS 格式)
					if sn, ok := transport["service_name"].(string); ok && sn != "" {
						grpcOpts["grpc-service-name"] = sn
					}
					if len(grpcOpts) > 0 {
						proxy["grpc-opts"] = grpcOpts
					}

				case "http", "h2":
					proxy["network"] = "h2"
					h2Opts := make(map[string]interface{})
					if httpOptions, ok := transport["http_options"].(map[string]interface{}); ok {
						if host, ok := httpOptions["host"].([]interface{}); ok {
							h2Opts["host"] = host
						}
						if path, ok := httpOptions["path"].(string); ok {
							h2Opts["path"] = path
						}
					}
					if len(h2Opts) > 0 {
						proxy["h2-opts"] = h2Opts
					}
				}
			}
			delete(proxy, "transport")
		}

		// 🔧 处理直接在根级别的 network 相关字段
		// tcp 和 raw 是默认值，不需要显式设置
		if network, ok := proxy["network"].(string); ok {
			if network == "" || network == "tcp" || network == "raw" || network == "none" {
				delete(proxy, "network")
			}
		}

		proxies = append(proxies, proxy)
	}

	return proxies
}

// generateProxyGroups 生成代理组（自动按地区分类节点）
// 顺序：基础分组 -> 功能分组 -> 特殊分组 -> 地区分组（放最后）
func (g *ConfigGenerator) generateProxyGroups(nodes []ProxyNode) []ProxyGroup {
	var nodeNames []string
	for _, node := range nodes {
		nodeNames = append(nodeNames, node.Name)
	}

	// 按地区分类节点
	regionNodes := ClassifyNodesByRegion(nodeNames)
	regionNames := GetRegionNames(nodeNames)

	// 构建地区分组名称列表
	regionGroupNames := append([]string{}, regionNames...)

	// 常用地区（用于功能分组）
	commonRegions := []string{}
	preferredRegions := []string{"🇭🇰 香港节点", "🇨🇳 台湾节点", "🇯🇵 日本节点", "🇺🇲 美国节点", "🇸🇬 狮城节点", "🇰🇷 韩国节点"}
	for _, r := range preferredRegions {
		for _, rn := range regionNames {
			if rn == r {
				commonRegions = append(commonRegions, r)
				break
			}
		}
	}

	// 1. 基础分组（节点选择、自动选择、故障转移）
	groups := []ProxyGroup{
		{
			Name:    "🚀 节点选择",
			Type:    "select",
			Proxies: append(append([]string{"♻️ 自动选择", "🔯 故障转移"}, regionGroupNames...), "DIRECT"),
		},
		{
			Name:    "♻️ 自动选择",
			Type:    "select",
			Proxies: nodeNames,
		},
		{
			Name:     "🔯 故障转移",
			Type:     "fallback",
			Proxies:  nodeNames,
			URL:      "http://www.gstatic.com/generate_204",
			Interval: 300,
		},
	}

	// 2. 功能分组（只显示国家分组，不显示全部节点）
	groups = append(groups, []ProxyGroup{
		{
			Name:    "🎬 国外媒体",
			Type:    "select",
			Proxies: append([]string{"🚀 节点选择"}, commonRegions...),
		},
		{
			Name:    "🎮 游戏平台",
			Type:    "select",
			Proxies: append(append([]string{"🚀 节点选择", "🔯 故障转移"}, commonRegions...), "DIRECT"),
		},
		{
			Name:    "📱 即时通讯",
			Type:    "select",
			Proxies: append([]string{"🚀 节点选择", "🔯 故障转移"}, commonRegions...),
		},
		{
			Name:    "🤖 AI平台",
			Type:    "select",
			Proxies: []string{"🇯🇵 日本节点", "🇺🇲 美国节点", "🇸🇬 狮城节点", "🇰🇷 韩国节点", "🚀 节点选择", "🔯 故障转移"},
		},
		{
			Name:    "🔧 GitHub",
			Type:    "select",
			Proxies: append(append([]string{"🚀 节点选择", "🔯 故障转移"}, commonRegions...), "DIRECT"),
		},
		{
			Name:    "Ⓜ️ 微软服务",
			Type:    "select",
			Proxies: append(append([]string{"🚀 节点选择"}, commonRegions...), "DIRECT"),
		},
		{
			Name:    "🍎 苹果服务",
			Type:    "select",
			Proxies: append(append([]string{"🚀 节点选择"}, commonRegions...), "DIRECT"),
		},
		{
			Name:    "📢 谷歌服务",
			Type:    "select",
			Proxies: append([]string{"🚀 节点选择"}, commonRegions...),
		},
	}...)

	// 3. 特殊分组
	groups = append(groups, []ProxyGroup{
		{
			Name:    "🎯 全球直连",
			Type:    "select",
			Proxies: []string{"DIRECT", "🚀 节点选择"},
		},
		{
			Name:    "🛑 广告拦截",
			Type:    "select",
			Proxies: []string{"REJECT", "DIRECT"},
		},
		{
			Name:    "🍃 应用净化",
			Type:    "select",
			Proxies: []string{"REJECT", "DIRECT"},
		},
		{
			Name:    "🆎 AdBlock",
			Type:    "select",
			Proxies: []string{"REJECT", "DIRECT"},
		},
		{
			Name:    "🛡️ 隐私防护",
			Type:    "select",
			Proxies: []string{"REJECT", "DIRECT"},
		},
		{
			Name:    "🐟 漏网之鱼",
			Type:    "select",
			Proxies: []string{"🚀 节点选择", "🎯 全球直连", "♻️ 自动选择", "🔯 故障转移"},
		},
	}...)

	// 4. 地区分组放最后（香港、日本、美国等）
	for _, regionName := range regionNames {
		if matched, ok := regionNodes[regionName]; ok && len(matched) > 0 {
			groups = append(groups, ProxyGroup{
				Name:    regionName,
				Type:    "select",
				Proxies: matched,
			})
		}
	}

	return groups
}

// generateRuleProviders 生成规则提供者（优先使用本地文件，使用绝对路径）
func (g *ConfigGenerator) generateRuleProviders() map[string]RuleProvider {
	rulesetDir := filepath.Join(g.dataDir, "ruleset")
	baseURL := "https://testingcf.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@meta/geo"

	// 规则定义
	rules := []struct {
		name     string
		behavior string
		urlPath  string
	}{
		{"private-domain", "domain", "/geosite/private.mrs"},
		{"private-ip", "ipcidr", "/geoip/private.mrs"},
		{"ai-domain", "domain", "/geosite/openai.mrs"},
		{"youtube-domain", "domain", "/geosite/youtube.mrs"},
		{"google-domain", "domain", "/geosite/google.mrs"},
		{"google-ip", "ipcidr", "/geoip/google.mrs"},
		{"telegram-domain", "domain", "/geosite/telegram.mrs"},
		{"telegram-ip", "ipcidr", "/geoip/telegram.mrs"},
		{"twitter-domain", "domain", "/geosite/twitter.mrs"},
		{"twitter-ip", "ipcidr", "/geoip/twitter.mrs"},
		{"facebook-domain", "domain", "/geosite/facebook.mrs"},
		{"facebook-ip", "ipcidr", "/geoip/facebook.mrs"},
		{"github-domain", "domain", "/geosite/github.mrs"},
		{"apple-domain", "domain", "/geosite/apple.mrs"},
		{"apple-cn-domain", "domain", "/geosite/apple-cn.mrs"},
		{"microsoft-domain", "domain", "/geosite/microsoft.mrs"},
		{"netflix-domain", "domain", "/geosite/netflix.mrs"},
		{"netflix-ip", "ipcidr", "/geoip/netflix.mrs"},
		{"spotify-domain", "domain", "/geosite/spotify.mrs"},
		{"tiktok-domain", "domain", "/geosite/tiktok.mrs"},
		{"bilibili-domain", "domain", "/geosite/bilibili.mrs"},
		{"steam-domain", "domain", "/geosite/steam.mrs"},
		{"epic-domain", "domain", "/geosite/epicgames.mrs"},
		{"cn-domain", "domain", "/geosite/cn.mrs"},
		{"cn-ip", "ipcidr", "/geoip/cn.mrs"},
		{"geolocation-!cn", "domain", "/geosite/geolocation-!cn.mrs"},
		{"ads-domain", "domain", "/geosite/category-ads-all.mrs"},
	}

	providers := make(map[string]RuleProvider)

	for _, r := range rules {
		localPath := filepath.Join(rulesetDir, r.name+".mrs")

		// 检查本地文件是否存在
		if _, err := os.Stat(localPath); err == nil {
			// 本地文件存在，使用 file 类型和绝对路径
			providers[r.name] = RuleProvider{
				Type:     "file",
				Behavior: r.behavior,
				Path:     localPath,
				Format:   "mrs",
			}
		} else {
			// 本地文件不存在，使用 http 类型下载
			providers[r.name] = RuleProvider{
				Type:     "http",
				Behavior: r.behavior,
				URL:      baseURL + r.urlPath,
				Path:     localPath,
				Interval: 86400,
				Format:   "mrs",
			}
		}
	}

	return providers
}

// getGeoxURL 获取 GEO 数据文件 URL（优先使用本地文件）
func (g *ConfigGenerator) getGeoxURL() (*GeoxURL, bool) {
	baseURL := "https://testingcf.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release"

	// 本地文件路径
	geoipPath := filepath.Join(g.dataDir, "geoip.dat")
	geositePath := filepath.Join(g.dataDir, "geosite.dat")
	mmdbPath := filepath.Join(g.dataDir, "country.mmdb")
	asnPath := filepath.Join(g.dataDir, "GeoLite2-ASN.mmdb")

	geox := &GeoxURL{}
	hasLocal := false

	// GeoIP
	if _, err := os.Stat(geoipPath); err == nil {
		geox.GeoIP = geoipPath
		hasLocal = true
	} else {
		geox.GeoIP = baseURL + "/geoip.dat"
	}

	// GeoSite
	if _, err := os.Stat(geositePath); err == nil {
		geox.GeoSite = geositePath
		hasLocal = true
	} else {
		geox.GeoSite = baseURL + "/geosite.dat"
	}

	// MMDB
	if _, err := os.Stat(mmdbPath); err == nil {
		geox.MMDB = mmdbPath
		hasLocal = true
	} else {
		geox.MMDB = baseURL + "/country.mmdb"
	}

	// ASN
	if _, err := os.Stat(asnPath); err == nil {
		geox.ASN = asnPath
		hasLocal = true
	} else {
		geox.ASN = baseURL + "/GeoLite2-ASN.mmdb"
	}

	return geox, hasLocal
}

// generateRules 生成规则 (使用 RULE-SET 引用远程规则)
// 使用中文代理组名称，与 config_template.go 中的定义保持一致
func (g *ConfigGenerator) generateRules() []string {
	return []string{
		// 私有网络直连
		"RULE-SET,private-domain,全球直连",
		"RULE-SET,private-ip,全球直连,no-resolve",

		// 广告拦截
		"RULE-SET,ads-domain,广告拦截",

		// AI 平台 (OpenAI, Claude, etc.)
		"RULE-SET,ai-domain,AI服务",

		// Telegram
		"RULE-SET,telegram-domain,电报消息",
		"RULE-SET,telegram-ip,电报消息,no-resolve",

		// YouTube
		"RULE-SET,youtube-domain,国外媒体",

		// Google
		"RULE-SET,google-domain,谷歌服务",
		"RULE-SET,google-ip,谷歌服务,no-resolve",

		// Twitter/X
		"RULE-SET,twitter-domain,推特消息",
		"RULE-SET,twitter-ip,推特消息,no-resolve",

		// Facebook
		"RULE-SET,facebook-domain,脸书服务",
		"RULE-SET,facebook-ip,脸书服务,no-resolve",

		// GitHub
		"RULE-SET,github-domain,GitHub",

		// Microsoft
		"RULE-SET,microsoft-domain,微软服务",

		// Apple
		"RULE-SET,apple-cn-domain,全球直连",
		"RULE-SET,apple-domain,苹果服务",

		// Netflix
		"RULE-SET,netflix-domain,国外媒体",
		"RULE-SET,netflix-ip,国外媒体,no-resolve",

		// Spotify
		"RULE-SET,spotify-domain,国外媒体",

		// TikTok
		"RULE-SET,tiktok-domain,国外媒体",

		// 游戏平台
		"RULE-SET,steam-domain,游戏平台",
		"RULE-SET,epic-domain,游戏平台",

		// Bilibili
		"RULE-SET,bilibili-domain,哔哩哔哩",

		// 国内直连
		"RULE-SET,cn-domain,全球直连",

		// 国外代理
		"RULE-SET,geolocation-!cn,节点选择",

		// GeoIP 规则
		"GEOIP,LAN,全球直连,no-resolve",
		"GEOIP,CN,全球直连,no-resolve",

		// 兜底规则
		"MATCH,漏网之鱼",
	}
}

// SaveConfig 保存配置到文件
func (g *ConfigGenerator) SaveConfig(config *MihomoConfig, filename string) (string, error) {
	configDir := filepath.Join(g.dataDir, "configs")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	if !strings.HasSuffix(filename, ".yaml") {
		filename += ".yaml"
	}

	filePath := filepath.Join(configDir, filename)

	// 序列化为 YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}

	// 解码 Unicode 转义序列 (如 \U0001F1ED -> 🇭🇰)
	yamlStr := decodeUnicodeEscapes(string(data))

	if err := os.WriteFile(filePath, []byte(yamlStr), 0644); err != nil {
		return "", err
	}

	return filePath, nil
}

// decodeUnicodeEscapes 将 YAML 中的 Unicode 转义序列转换回原始字符
func decodeUnicodeEscapes(s string) string {
	// 处理 \UXXXXXXXX 格式 (8位 Unicode)
	result := s
	for {
		idx := strings.Index(result, "\\U")
		if idx == -1 {
			break
		}
		if idx+10 <= len(result) {
			hexStr := result[idx+2 : idx+10]
			if codePoint, err := strconv.ParseInt(hexStr, 16, 32); err == nil {
				char := string(rune(codePoint))
				result = result[:idx] + char + result[idx+10:]
				continue
			}
		}
		// 无法解析，跳过
		result = result[:idx] + result[idx+2:]
	}
	return result
}

// LoadConfig 从文件加载配置
func (g *ConfigGenerator) LoadConfig(filename string) (*MihomoConfig, error) {
	configDir := filepath.Join(g.dataDir, "configs")
	filePath := filepath.Join(configDir, filename)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config MihomoConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetDefaultOptions 获取默认配置选项
func GetDefaultOptions() ConfigGeneratorOptions {
	return ConfigGeneratorOptions{
		MixedPort:          7890,
		AllowLan:           true,
		Mode:               "rule",
		LogLevel:           "info",
		IPv6:               false,
		EnableTProxy:       false,
		TProxyPort:         7893,
		EnableTUN:          false,
		EnableDNS:          true,
		DNSListen:          "0.0.0.0:1053",
		EnhancedMode:       "fake-ip",
		ExternalController: "127.0.0.1:9090",
	}
}

// generateProxyGroupsFromTemplate 从模板生成代理组
func (g *ConfigGenerator) generateProxyGroupsFromTemplate(nodes []ProxyNode, templates []ProxyGroupTemplate) []ProxyGroup {
	var nodeNames []string
	var manualNodeNames []string
	for _, node := range nodes {
		nodeNames = append(nodeNames, node.Name)
		if node.IsManual {
			manualNodeNames = append(manualNodeNames, node.Name)
		}
	}

	var groups []ProxyGroup

	for _, t := range templates {
		if !t.Enabled && t.Description != "" {
			// 跳过禁用的分组（但允许新建的默认分组）
			continue
		}

		group := ProxyGroup{
			Name:     t.Name,
			Type:     t.Type,
			URL:      t.URL,
			Interval: t.Interval,
		}

		// 处理代理列表
		if t.UseAll {
			// 特殊处理：手动节点分组
			if t.Filter == "__MANUAL__" {
				group.Proxies = manualNodeNames
			} else if t.Filter != "" {
				// 使用模板中的 Filter 正则过滤节点
				re, err := regexp.Compile(t.Filter)
				if err == nil {
					for _, nodeName := range nodeNames {
						if re.MatchString(nodeName) {
							group.Proxies = append(group.Proxies, nodeName)
						}
					}
				}
				// 如果没匹配到任何节点，使用全部节点
				if len(group.Proxies) == 0 {
					group.Proxies = nodeNames
				}
			} else {
				group.Proxies = nodeNames
			}
		} else {
			// 使用模板中定义的代理列表
			group.Proxies = append(group.Proxies, t.Proxies...)
		}

		// 确保有代理
		if len(group.Proxies) == 0 {
			group.Proxies = []string{"DIRECT"}
		}
		group.Proxies = normalizeProxyReferences(group.Proxies)
		if len(group.Proxies) == 0 {
			group.Proxies = []string{"DIRECT"}
		}

		groups = append(groups, group)
	}

	// 不再自动添加额外的地区分组，只使用模板中定义的分组
	// 这样可以保持模板中的顺序和只生成用户需要的分组

	return groups
}

// generateRulesFromTemplate 从模板生成规则
func (g *ConfigGenerator) generateRulesFromTemplate(templates []RuleTemplate) []string {
	var rules []string

	for _, t := range templates {
		var rule string
		// MATCH 规则不需要 Payload
		if t.Type == "MATCH" {
			rule = t.Type + "," + t.Proxy
		} else if t.NoResolve {
			rule = t.Type + "," + t.Payload + "," + t.Proxy + ",no-resolve"
		} else {
			rule = t.Type + "," + t.Payload + "," + t.Proxy
		}
		rules = append(rules, rule)
	}

	return rules
}
