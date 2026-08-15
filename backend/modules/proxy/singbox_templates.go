package proxy

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ============================================================================
// Sing-Box 配置模板
// ============================================================================

// SingBoxProxyGroupTemplate Sing-Box 代理组模板
type SingBoxProxyGroupTemplate struct {
	Tag         string   `json:"tag"`
	Type        string   `json:"type"` // selector, urltest
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Enabled     bool     `json:"enabled"`
	Outbounds   []string `json:"outbounds"`
	Default     string   `json:"default,omitempty"`
	URL         string   `json:"url,omitempty"`
	Interval    string   `json:"interval,omitempty"`
	Tolerance   int      `json:"tolerance,omitempty"`
}

// SingBoxRuleTemplate Sing-Box 规则模板
type SingBoxRuleTemplate struct {
	RuleSet  interface{} `json:"rule_set,omitempty"` // string or []string
	Outbound string      `json:"outbound,omitempty"`
	Action   string      `json:"action,omitempty"`
}

// SingBoxRuleSetTemplate Sing-Box 规则集模板
type SingBoxRuleSetTemplate struct {
	Tag    string `json:"tag"`
	Type   string `json:"type"`   // local, remote
	Format string `json:"format"` // binary, source
	Path   string `json:"path,omitempty"`
	URL    string `json:"url,omitempty"`
}

// SingBoxTemplate Sing-Box 配置模板
type SingBoxTemplate struct {
	ProxyGroups []SingBoxProxyGroupTemplate `json:"proxyGroups"`
	Rules       []SingBoxRuleTemplate       `json:"rules"`
	RuleSets    []SingBoxRuleSetTemplate    `json:"ruleSets"`
}

// GetSingBoxTUNTemplate 获取 TUN 模式配置模板
func GetSingBoxTUNTemplate(opts SingBoxGeneratorOptions) *SingBoxConfig {
	mixedPort := opts.MixedPort
	if mixedPort == 0 {
		mixedPort = 7890
	}

	clashAPIAddr := opts.ClashAPIAddr
	if clashAPIAddr == "" {
		clashAPIAddr = "127.0.0.1:9090"
	}

	tunStack := opts.TUNStack
	if tunStack == "" {
		tunStack = "system"
	}

	tunMTU := opts.TUNMTU
	if tunMTU == 0 {
		tunMTU = 9000
	}

	logLevel := opts.LogLevel
	if logLevel == "" {
		logLevel = "info"
	}

	dnsStrategy := opts.DNSStrategy
	if dnsStrategy == "" {
		dnsStrategy = "prefer_ipv4"
	}

	// 默认值处理
	autoRedirect := opts.AutoRedirect
	strictRoute := opts.StrictRoute
	sniff := opts.Sniff
	sniffOverride := opts.SniffOverrideDestination

	// 如果没有明确设置，使用默认最优值
	if !autoRedirect && !strictRoute && !sniff && !sniffOverride {
		autoRedirect = true
		strictRoute = true
		sniff = true
		sniffOverride = true
	}

	config := &SingBoxConfig{
		Log: &SBLog{
			Level:     logLevel,
			Timestamp: true,
		},
		Experimental: &SBExperimental{
			ClashAPI: &SBClashAPI{
				ExternalController: clashAPIAddr,
				Secret:             opts.ClashAPISecret,
				DefaultMode:        "rule",
			},
			CacheFile: &SBCacheFile{
				Enabled:     true,
				StoreFakeIP: opts.FakeIP,
				StoreRDRC:   true,
				RDRCTimeout: "7d",
			},
		},
		Inbounds: []SBInbound{
			{
				Tag:                      "tun-in",
				Type:                     "tun",
				Address:                  []string{"172.19.0.0/30", "fdfe:dcba:9876::0/126"},
				MTU:                      tunMTU,
				AutoRoute:                true,
				AutoRedirect:             autoRedirect,
				StrictRoute:              strictRoute,
				Stack:                    tunStack,
				UDPTimeout:               "5m",
				Sniff:                    sniff,
				SniffOverrideDestination: sniffOverride,
				Platform: &SBPlatform{
					HTTPProxy: &SBHTTPProxy{
						Enabled:    true,
						Server:     "127.0.0.1",
						ServerPort: mixedPort,
					},
				},
			},
			{
				Tag:                      "mixed-in",
				Type:                     "mixed",
				Listen:                   "127.0.0.1",
				ListenPort:               mixedPort,
				Sniff:                    sniff,
				SniffOverrideDestination: sniffOverride,
			},
		},
		Route: &SBRoute{
			AutoDetectInterface: true,
			Final:               "节点选择",
			DefaultDomainResolver: &SBDomainResolver{
				Server: "local",
			},
		},
	}

	// DNS 配置
	if opts.FakeIP {
		config.DNS = getFakeIPDNS(dnsStrategy)
	} else {
		config.DNS = getRealIPDNS(dnsStrategy)
	}

	return config
}

// GetSingBoxSystemTemplate 获取系统代理模式配置模板
func GetSingBoxSystemTemplate(opts SingBoxGeneratorOptions) *SingBoxConfig {
	mixedPort := opts.MixedPort
	if mixedPort == 0 {
		mixedPort = 7890
	}

	httpPort := opts.HTTPPort
	if httpPort == 0 {
		httpPort = 7891
	}

	socksPort := opts.SocksPort
	if socksPort == 0 {
		socksPort = 7892
	}

	clashAPIAddr := opts.ClashAPIAddr
	if clashAPIAddr == "" {
		clashAPIAddr = "127.0.0.1:9090"
	}

	logLevel := opts.LogLevel
	if logLevel == "" {
		logLevel = "info"
	}

	dnsStrategy := opts.DNSStrategy
	if dnsStrategy == "" {
		dnsStrategy = "prefer_ipv4"
	}

	// 默认值处理
	sniff := opts.Sniff
	sniffOverride := opts.SniffOverrideDestination

	// 如果没有明确设置，使用默认最优值
	if !sniff && !sniffOverride {
		sniff = true
		sniffOverride = true
	}

	config := &SingBoxConfig{
		Log: &SBLog{
			Level:     logLevel,
			Timestamp: true,
		},
		Experimental: &SBExperimental{
			ClashAPI: &SBClashAPI{
				ExternalController: clashAPIAddr,
				Secret:             opts.ClashAPISecret,
				DefaultMode:        "rule",
			},
			CacheFile: &SBCacheFile{
				Enabled:     true,
				StoreRDRC:   true,
				RDRCTimeout: "7d",
			},
		},
		DNS: getRealIPDNS(dnsStrategy),
		Inbounds: []SBInbound{
			{
				Tag:                      "mixed-in",
				Type:                     "mixed",
				Listen:                   "127.0.0.1",
				ListenPort:               mixedPort,
				Sniff:                    sniff,
				SniffOverrideDestination: sniffOverride,
			},
			{
				Tag:                      "http-in",
				Type:                     "http",
				Listen:                   "127.0.0.1",
				ListenPort:               httpPort,
				Sniff:                    sniff,
				SniffOverrideDestination: sniffOverride,
			},
			{
				Tag:                      "socks-in",
				Type:                     "socks",
				Listen:                   "127.0.0.1",
				ListenPort:               socksPort,
				Sniff:                    sniff,
				SniffOverrideDestination: sniffOverride,
			},
		},
		Route: &SBRoute{
			AutoDetectInterface: true,
			Final:               "节点选择",
			DefaultDomainResolver: &SBDomainResolver{
				Server: "localDns",
			},
		},
	}

	return config
}

// ============================================================================
// DNS 模板
// ============================================================================

// getFakeIPDNS 获取 FakeIP DNS 配置
// sing-box 1.12+ DNS 服务器格式
func getFakeIPDNS(strategy string) *SBDNS {
	return &SBDNS{
		Servers: []SBDNSServer{
			{
				Tag:    "google",
				Type:   "https",
				Server: "8.8.8.8", // 使用 IP 避免域名解析循环
			},
			{
				Tag:    "local",
				Type:   "https",
				Server: "223.5.5.5", // 阿里 DNS IP
			},
			{
				Tag:        "fakeip",
				Type:       "fakeip",
				Inet4Range: "198.18.0.0/15",
				Inet6Range: "fc00::/18",
			},
		},
		Rules: []SBDNSRule{
			{
				ClashMode: "direct",
				Server:    "local",
			},
			{
				ClashMode: "global",
				Server:    "google",
			},
			{
				QueryType: []string{"A", "AAAA"},
				RuleSet:   "geosite-cn",
				Server:    "fakeip",
			},
			{
				RuleSet: "geosite-cn",
				Server:  "local",
			},
			{
				Type: "logical",
				Mode: "and",
				Rules: []SBDNSRule{
					{
						RuleSet: "geosite-geolocation-!cn",
						Invert:  true,
					},
					{
						RuleSet: "geoip-cn",
					},
				},
				Server: "google",
			},
			{
				QueryType: []string{"A", "AAAA"},
				Server:    "fakeip",
			},
		},
		IndependentCache: true,
		Strategy:         strategy,
	}
}

// getRealIPDNS 获取真实 IP DNS 配置
// sing-box 1.12+ DNS 服务器格式
func getRealIPDNS(strategy string) *SBDNS {
	return &SBDNS{
		Servers: []SBDNSServer{
			{
				Tag:    "proxyDns",
				Type:   "https",
				Server: "8.8.8.8", // 使用 IP 避免域名解析循环
			},
			{
				Tag:    "localDns",
				Type:   "https",
				Server: "223.5.5.5", // 阿里 DNS IP
			},
		},
		Rules: []SBDNSRule{
			// 注: outbound: "any" 已弃用，使用 route.default_domain_resolver 替代
			{
				RuleSet: "geosite-cn",
				Server:  "localDns",
			},
			{
				ClashMode: "direct",
				Server:    "localDns",
			},
			{
				ClashMode: "global",
				Server:    "proxyDns",
			},
			{
				RuleSet: "geosite-geolocation-!cn",
				Server:  "proxyDns",
			},
		},
		Final:    "localDns",
		Strategy: strategy,
	}
}

// ============================================================================
// 路由规则模板
// ============================================================================

// GetDefaultRouteRules 获取默认路由规则
func GetDefaultRouteRules() []SBRouteRule {
	return []SBRouteRule{
		// Sniff
		{
			Inbound: []string{"tun-in", "mixed-in"},
			Action:  "sniff",
		},
		// DNS 劫持
		{
			Type: "logical",
			Mode: "or",
			Rules: []SBRouteRule{
				{Port: 53},
				{Protocol: "dns"},
			},
			Action: "hijack-dns",
		},
		// 广告拦截 (指向广告拦截分组，用户可在面板中切换 block/direct)
		{
			RuleSet:  "geosite-category-ads-all",
			Outbound: "广告拦截",
		},
		// Direct 模式 - 使用 action: "direct" 替代 outbound
		{
			ClashMode: "direct",
			Action:    "direct",
		},
		// Global 模式
		{
			ClashMode: "global",
			Outbound:  "节点选择",
		},
		// 面板直连
		{
			Domain: []string{
				"clash.razord.top",
				"yacd.metacubex.one",
				"yacd.haishan.me",
				"d.metacubex.one",
			},
			Outbound: "direct",
		},
		// 私有 IP 直连
		{
			IPIsPrivate: true,
			Outbound:    "direct",
		},
		// ⭐ 中国域名直连 (优先匹配，避免被后面规则覆盖)
		{
			RuleSet:  "geosite-cn",
			Outbound: "direct",
		},
		// AI 服务 (OpenAI, Claude, Gemini, Cursor)
		{
			RuleSet:  []string{"geosite-openai", "geosite-anthropic", "geosite-google-gemini", "geosite-cursor", "geosite-category-ai-!cn"},
			Outbound: "AI服务",
		},
		// 游戏平台
		{
			RuleSet:  []string{"geosite-steam", "geosite-epicgames"},
			Outbound: "游戏平台",
		},
		// 国外媒体 (YouTube, Netflix, Spotify)
		{
			RuleSet:  []string{"geosite-youtube", "geosite-netflix", "geosite-spotify", "geosite-disney"},
			Outbound: "国外媒体",
		},
		// 社交媒体 (Telegram, Twitter, Facebook, Instagram)
		{
			RuleSet:  []string{"geosite-telegram", "geosite-twitter", "geosite-facebook", "geosite-instagram"},
			Outbound: "社交媒体",
		},
		// 海外聊天 (Discord, WhatsApp)
		{
			RuleSet:  []string{"geosite-discord", "geosite-whatsapp"},
			Outbound: "海外聊天",
		},
		// 谷歌服务
		{
			RuleSet:  "geosite-google",
			Outbound: "谷歌服务",
		},
		// GitHub
		{
			RuleSet:  "geosite-github",
			Outbound: "GitHub",
		},
		// 微软服务
		{
			RuleSet:  "geosite-microsoft",
			Outbound: "微软服务",
		},
		// 苹果服务
		{
			RuleSet:  "geosite-apple",
			Outbound: "苹果服务",
		},
		// 哔哩哔哩
		{
			RuleSet:  "geosite-bilibili",
			Outbound: "哔哩哔哩",
		},
		// ⭐ 中国 IP 直连 (解析后的 IP 如果是中国则直连)
		{
			RuleSet:  "geoip-cn",
			Outbound: "direct",
		},
		// 非中国域名 -> 漏网之鱼
		{
			RuleSet:  "geosite-geolocation-!cn",
			Outbound: "漏网之鱼",
		},
	}
}

// ============================================================================
// 规则集模板
// ============================================================================

// GetDefaultRuleSets 获取默认规则集（优先使用本地文件）
func GetDefaultRuleSets() []SBRuleSet {
	// 官方规则仓库 URL
	baseURL := "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set"
	geoipURL := "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set"

	// 本地规则路径
	localDir := GetSingBoxRulesetDir()

	// 规则定义 (共29个，与前端保持一致)
	rules := []struct {
		tag     string
		url     string
		isGeoIP bool
	}{
		// 广告
		{"geosite-category-ads-all", baseURL + "/geosite-category-ads-all.srs", false},
		// AI 服务 (6个)
		{"geosite-openai", baseURL + "/geosite-openai.srs", false},
		{"geosite-anthropic", baseURL + "/geosite-anthropic.srs", false},
		{"geosite-google-gemini", baseURL + "/geosite-google-gemini.srs", false},
		{"geosite-cursor", baseURL + "/geosite-cursor.srs", false},
		{"geosite-category-ai-!cn", baseURL + "/geosite-category-ai-!cn.srs", false},
		// 游戏平台
		{"geosite-steam", baseURL + "/geosite-steam.srs", false},
		{"geosite-epicgames", baseURL + "/geosite-epicgames.srs", false},
		// 流媒体
		{"geosite-netflix", baseURL + "/geosite-netflix.srs", false},
		{"geosite-disney", baseURL + "/geosite-disney.srs", false},
		{"geosite-youtube", baseURL + "/geosite-youtube.srs", false},
		{"geosite-spotify", baseURL + "/geosite-spotify.srs", false},
		// 社交媒体
		{"geosite-twitter", baseURL + "/geosite-twitter.srs", false},
		{"geosite-facebook", baseURL + "/geosite-facebook.srs", false},
		{"geosite-instagram", baseURL + "/geosite-instagram.srs", false},
		// 海外聊天
		{"geosite-telegram", baseURL + "/geosite-telegram.srs", false},
		{"geosite-whatsapp", baseURL + "/geosite-whatsapp.srs", false},
		{"geosite-discord", baseURL + "/geosite-discord.srs", false},
		// Google
		{"geosite-google", baseURL + "/geosite-google.srs", false},
		// 开发者
		{"geosite-github", baseURL + "/geosite-github.srs", false},
		// Microsoft
		{"geosite-microsoft", baseURL + "/geosite-microsoft.srs", false},
		// Apple
		{"geosite-apple", baseURL + "/geosite-apple.srs", false},
		// 中国直连
		{"geosite-bilibili", baseURL + "/geosite-bilibili.srs", false},
		{"geosite-iqiyi", baseURL + "/geosite-iqiyi.srs", false},
		{"geosite-alibaba", baseURL + "/geosite-alibaba.srs", false},
		{"geosite-cn", baseURL + "/geosite-cn.srs", false},
		{"geoip-cn", geoipURL + "/geoip-cn.srs", true},
		// 其他海外
		{"geosite-geolocation-!cn", baseURL + "/geosite-geolocation-!cn.srs", false},
	}

	result := make([]SBRuleSet, 0, len(rules))
	for _, r := range rules {
		localPath := localDir + "/" + r.tag + ".srs"

		// 检查本地文件是否存在
		if fileExists(localPath) {
			// 使用本地文件
			result = append(result, SBRuleSet{
				Tag:    r.tag,
				Type:   "local",
				Format: "binary",
				Path:   localPath,
			})
		} else {
			// 使用远程 URL (不指定 download_detour，使用默认出站)
			result = append(result, SBRuleSet{
				Tag:    r.tag,
				Type:   "remote",
				Format: "binary",
				URL:    r.url,
			})
		}
	}

	return result
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ============================================================================
// 代理组模板
// ============================================================================

// RegionFilter 地区过滤器
type RegionFilter struct {
	Tag      string
	Keywords []string
	Default  string
}

// GetDefaultRegionFilters 获取默认地区过滤器
func GetDefaultRegionFilters() []RegionFilter {
	return []RegionFilter{
		{Tag: "HongKong", Keywords: []string{"🇭🇰", "HK", "hk", "香港", "港", "HongKong", "Hong Kong"}},
		{Tag: "Taiwan", Keywords: []string{"�🇳", "�🇹🇼", "TW", "tw", "台湾", "臺灣", "台", "Taiwan"}},
		{Tag: "Singapore", Keywords: []string{"🇸🇬", "SG", "sg", "新加坡", "狮", "Singapore"}},
		{Tag: "Japan", Keywords: []string{"🇯🇵", "JP", "jp", "日本", "日", "Japan"}},
		{Tag: "America", Keywords: []string{"🇺🇸", "US", "us", "美国", "美", "United States", "USA"}},
		{Tag: "Korea", Keywords: []string{"🇰🇷", "KR", "kr", "韩国", "韓國", "Korea"}},
	}
}

// GetSingBoxProxyGroups 获取默认代理组定义
// 参考 Mihomo 的代理组结构
func GetSingBoxProxyGroups() []SBOutbound {
	return []SBOutbound{
		// 1. 自动选择
		{
			Tag:       "自动选择",
			Type:      "selector",
			Outbounds: []string{}, // 动态填充所有节点
		},
		// 2. 故障转移 (不引用 手动节点 避免循环依赖)
		{
			Tag:       "故障转移",
			Type:      "urltest",
			Outbounds: []string{"香港节点", "台湾节点", "日本节点", "新加坡节点", "美国节点"},
			URL:       "https://www.gstatic.com/generate_204",
			Interval:  "5m",
		},
		// 3. 节点选择 (主选择器)
		{
			Tag:       "节点选择",
			Type:      "selector",
			Outbounds: []string{"自动选择", "故障转移", "香港节点", "台湾节点", "日本节点", "新加坡节点", "美国节点", "手动节点", "其他节点"},
		},
		// 4. 全球直连
		{
			Tag:       "全球直连",
			Type:      "selector",
			Outbounds: []string{"direct", "节点选择", "自动选择"},
		},
		// 5. 广告拦截 (block=拦截, direct=放行)
		{
			Tag:       "广告拦截",
			Type:      "selector",
			Outbounds: []string{"block", "direct"},
			Default:   "block",
		},
		// 6. AI服务
		{
			Tag:       "AI服务",
			Type:      "selector",
			Outbounds: []string{"节点选择", "美国节点", "日本节点", "新加坡节点", "台湾节点", "手动节点", "自动选择"},
			Default:   "美国节点",
		},
		// 7. 游戏平台
		{
			Tag:       "游戏平台",
			Type:      "selector",
			Outbounds: []string{"节点选择", "direct", "香港节点", "台湾节点", "日本节点", "手动节点"},
		},
		// 8. 国外媒体
		{
			Tag:       "国外媒体",
			Type:      "selector",
			Outbounds: []string{"节点选择", "香港节点", "台湾节点", "日本节点", "新加坡节点", "美国节点", "手动节点", "自动选择"},
		},
		// 9. 社交媒体 (Telegram, Twitter, Facebook, Instagram)
		{
			Tag:       "社交媒体",
			Type:      "selector",
			Outbounds: []string{"节点选择", "香港节点", "台湾节点", "新加坡节点", "美国节点", "手动节点", "自动选择"},
		},
		// 10. 海外聊天 (Discord, WhatsApp)
		{
			Tag:       "海外聊天",
			Type:      "selector",
			Outbounds: []string{"节点选择", "香港节点", "台湾节点", "新加坡节点", "美国节点", "手动节点", "自动选择"},
		},
		// 11. 谷歌服务
		{
			Tag:       "谷歌服务",
			Type:      "selector",
			Outbounds: []string{"节点选择", "香港节点", "台湾节点", "日本节点", "美国节点", "手动节点", "自动选择"},
		},
		// 12. GitHub
		{
			Tag:       "GitHub",
			Type:      "selector",
			Outbounds: []string{"节点选择", "手动节点", "自动选择"},
		},
		// 13. 微软服务
		{
			Tag:       "微软服务",
			Type:      "selector",
			Outbounds: []string{"节点选择", "direct", "香港节点", "美国节点", "手动节点"},
		},
		// 14. 苹果服务
		{
			Tag:       "苹果服务",
			Type:      "selector",
			Outbounds: []string{"节点选择", "direct", "美国节点", "手动节点"},
		},
		// 15. 哔哩哔哩
		{
			Tag:       "哔哩哔哩",
			Type:      "selector",
			Outbounds: []string{"direct", "香港节点", "台湾节点", "手动节点"},
		},
		// 16. 漏网之鱼
		{
			Tag:       "漏网之鱼",
			Type:      "selector",
			Outbounds: []string{"节点选择", "自动选择", "手动节点"},
		},
		// === 地区节点分组 (sing-box urltest 不支持手动切换，改用 selector) ===
		// 17. 香港节点
		{
			Tag:       "香港节点",
			Type:      "selector",
			Outbounds: []string{}, // 动态填充
		},
		// 18. 台湾节点
		{
			Tag:       "台湾节点",
			Type:      "selector",
			Outbounds: []string{},
		},
		// 19. 日本节点
		{
			Tag:       "日本节点",
			Type:      "selector",
			Outbounds: []string{},
		},
		// 20. 新加坡节点
		{
			Tag:       "新加坡节点",
			Type:      "selector",
			Outbounds: []string{},
		},
		// 21. 美国节点
		{
			Tag:       "美国节点",
			Type:      "selector",
			Outbounds: []string{},
		},
		// 22. 手动节点
		{
			Tag:       "手动节点",
			Type:      "selector",
			Outbounds: []string{}, // 动态填充手动添加的节点
		},
		// 23. 其他节点
		{
			Tag:       "其他节点",
			Type:      "selector",
			Outbounds: []string{},
		},
	}
}

// GetDefaultSingBoxTemplate 获取默认 Sing-Box 模板
func GetDefaultSingBoxTemplate() *SingBoxTemplate {
	groups := GetSingBoxProxyGroups()
	proxyGroups := make([]SingBoxProxyGroupTemplate, len(groups))

	// 名称映射
	nameMap := map[string]string{
		"auto": "自动选择", "fallback": "故障转移", "proxy": "节点选择",
		"DIRECT": "全球直连", "AdBlock": "广告拦截", "AI": "AI服务",
		"Gaming": "游戏平台", "Streaming": "国外媒体", "Social": "社交媒体",
		"Chat": "海外聊天", "Google": "谷歌服务", "GitHub": "GitHub",
		"Microsoft": "微软服务", "Apple": "苹果服务", "BiliBili": "哔哩哔哩",
		"Final": "漏网之鱼", "HongKong": "香港节点", "Taiwan": "台湾节点",
		"Japan": "日本节点", "Singapore": "新加坡节点", "America": "美国节点",
		"Manual": "手动节点", "Others": "其他节点",
	}
	iconMap := map[string]string{
		"auto": "⚡", "fallback": "🛡️", "proxy": "🚀", "DIRECT": "🎯",
		"AdBlock": "🚫", "AI": "🤖", "Gaming": "🎮", "Streaming": "📺",
		"Social": "👥", "Chat": "💬", "Google": "🔍", "GitHub": "💻",
		"Microsoft": "🪟", "Apple": "🍎", "BiliBili": "📺", "Final": "🌐",
		"HongKong": "🇭🇰", "Taiwan": "🇨🇳", "Japan": "🇯🇵",
		"Singapore": "🇸🇬", "America": "🇺🇸", "Manual": "✋", "Others": "🌍",
	}

	for i, g := range groups {
		proxyGroups[i] = SingBoxProxyGroupTemplate{
			Tag:         g.Tag,
			Type:        g.Type,
			Name:        nameMap[g.Tag],
			Description: "",
			Icon:        iconMap[g.Tag],
			Enabled:     true,
			Outbounds:   g.Outbounds,
			Default:     g.Default,
			URL:         g.URL,
			Interval:    g.Interval,
			Tolerance:   g.Tolerance,
		}
		if proxyGroups[i].Name == "" {
			proxyGroups[i].Name = g.Tag
		}
	}

	// 默认规则
	rules := []SingBoxRuleTemplate{
		{RuleSet: "geosite-category-ads-all", Outbound: "AdBlock"},
		{RuleSet: []string{"geosite-openai", "geosite-anthropic"}, Outbound: "AI"},
		{RuleSet: []string{"geosite-steam", "geosite-epicgames"}, Outbound: "Gaming"},
		{RuleSet: []string{"geosite-youtube", "geosite-netflix", "geosite-spotify"}, Outbound: "Streaming"},
		{RuleSet: []string{"geosite-telegram", "geosite-twitter", "geosite-facebook"}, Outbound: "Social"},
		{RuleSet: []string{"geosite-discord", "geosite-whatsapp"}, Outbound: "Chat"},
		{RuleSet: "geosite-google", Outbound: "Google"},
		{RuleSet: "geosite-github", Outbound: "GitHub"},
		{RuleSet: "geosite-microsoft", Outbound: "Microsoft"},
		{RuleSet: "geosite-apple", Outbound: "Apple"},
		{RuleSet: "geosite-bilibili", Outbound: "BiliBili"},
		{RuleSet: []string{"geoip-cn", "geosite-cn"}, Outbound: "DIRECT"},
		{RuleSet: "geosite-geolocation-!cn", Outbound: "Final"},
	}

	// 获取规则集
	defaultRuleSets := GetDefaultRuleSets()
	ruleSets := make([]SingBoxRuleSetTemplate, len(defaultRuleSets))
	for i, rs := range defaultRuleSets {
		ruleSets[i] = SingBoxRuleSetTemplate{
			Tag:    rs.Tag,
			Type:   rs.Type,
			Format: rs.Format,
			Path:   rs.Path,
			URL:    rs.URL,
		}
	}

	return &SingBoxTemplate{
		ProxyGroups: proxyGroups,
		Rules:       rules,
		RuleSets:    ruleSets,
	}
}

// LoadSingBoxTemplate 从文件加载模板
func LoadSingBoxTemplate(dataDir string) *SingBoxTemplate {
	path := filepath.Join(dataDir, "singbox_template.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return GetDefaultSingBoxTemplate()
	}

	var template SingBoxTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return GetDefaultSingBoxTemplate()
	}
	return &template
}

// SaveSingBoxTemplate 保存模板到文件
func SaveSingBoxTemplate(dataDir string, template *SingBoxTemplate) error {
	path := filepath.Join(dataDir, "singbox_template.json")
	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
