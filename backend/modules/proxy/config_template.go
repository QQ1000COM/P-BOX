package proxy

import "strings"

// ProxyGroupTemplate 代理组模板
type ProxyGroupTemplate struct {
	Name        string   `json:"name" yaml:"name"`
	Type        string   `json:"type" yaml:"type"`           // select, url-test, fallback, load-balance
	Icon        string   `json:"icon" yaml:"icon,omitempty"` // 图标
	Description string   `json:"description" yaml:"-"`       // 说明 (仅前端显示)
	Enabled     bool     `json:"enabled" yaml:"-"`           // 是否启用 (默认 true)
	Proxies     []string `json:"proxies" yaml:"proxies"`     // 代理列表
	URL         string   `json:"url,omitempty" yaml:"url,omitempty"`
	Interval    int      `json:"interval,omitempty" yaml:"interval,omitempty"`
	Tolerance   int      `json:"tolerance,omitempty" yaml:"tolerance,omitempty"`
	Lazy        bool     `json:"lazy,omitempty" yaml:"lazy,omitempty"`
	Hidden      bool     `json:"hidden,omitempty" yaml:"hidden,omitempty"`
	Filter      string   `json:"filter,omitempty" yaml:"filter,omitempty"` // 节点过滤正则
	UseAll      bool     `json:"useAll,omitempty" yaml:"-"`                // 使用所有节点
}

// RuleTemplate 规则模板
type RuleTemplate struct {
	Type        string `json:"type"`        // DOMAIN, DOMAIN-SUFFIX, DOMAIN-KEYWORD, IP-CIDR, GEOIP, RULE-SET, MATCH
	Payload     string `json:"payload"`     // 规则内容
	Proxy       string `json:"proxy"`       // 代理组名称
	NoResolve   bool   `json:"noResolve"`   // 不解析域名
	Description string `json:"description"` // 说明
}

// RuleProviderTemplate 规则提供者模板
type RuleProviderTemplate struct {
	Name        string `json:"name"`
	Type        string `json:"type"`     // http, file
	Behavior    string `json:"behavior"` // domain, ipcidr, classical
	URL         string `json:"url"`
	Path        string `json:"path"`
	Interval    int    `json:"interval"`
	Format      string `json:"format"` // yaml, mrs
	Description string `json:"description"`
}

// ConfigTemplate 完整配置模板
type ConfigTemplate struct {
	ProxyGroups   []ProxyGroupTemplate   `json:"proxyGroups"`
	Rules         []RuleTemplate         `json:"rules"`
	RuleProviders []RuleProviderTemplate `json:"ruleProviders"`
}

func normalizeProxyReference(name string) string {
	trimmed := strings.TrimSpace(name)
	switch strings.ToLower(trimmed) {
	case "direct":
		return "DIRECT"
	case "reject", "block":
		return "REJECT"
	}

	switch trimmed {
	case "直连", "直接连接":
		return "DIRECT"
	case "拒绝", "阻止", "拦截":
		return "REJECT"
	default:
		return trimmed
	}
}

func normalizeProxyReferences(proxies []string) []string {
	normalized := make([]string, 0, len(proxies))
	seen := make(map[string]bool, len(proxies))
	for _, proxy := range proxies {
		name := normalizeProxyReference(proxy)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		normalized = append(normalized, name)
	}
	return normalized
}

// GetDefaultProxyGroups 获取默认代理组
// 顺序严格按照前端配置生成页面的显示顺序
// 使用现代扁平化 Apple 风格图标
func GetDefaultProxyGroups() []ProxyGroupTemplate {
	groups := []ProxyGroupTemplate{
		// 0. GLOBAL - 内置代理组，用于 global 模式和 Web 面板排序
		{
			Name:        "GLOBAL",
			Type:        "select",
			Icon:        "globe",
			Description: "全局代理模式默认出口，包含所有策略组",
			Proxies:     []string{"节点选择", "自动选择", "故障转移", "香港节点", "台湾节点", "日本节点", "新加坡节点", "美国节点", "手动节点", "其他节点", "DIRECT"},
			UseAll:      false,
		},
		// 1. 自动选择 - 放在最前面
		{
			Name:        "自动选择",
			Type:        "select",
			Icon:        "zap",
			Description: "手动选择节点（切换后保持固定，不会自动跳转）",
			Proxies:     []string{},
			UseAll:      true,
		},
		// 2. 故障转移
		{
			Name:        "故障转移",
			Type:        "fallback",
			Icon:        "shield",
			Description: "按顺序检测节点可用性，自动切换到可用节点",
			Proxies:     []string{"香港节点", "台湾节点", "日本节点", "新加坡节点", "美国节点", "手动节点"},
			URL:         "https://www.gstatic.com/generate_204",
			Interval:    300,
			Lazy:        true,
			UseAll:      false,
		},
		// 3. 节点选择
		{
			Name:        "节点选择",
			Type:        "select",
			Icon:        "rocket",
			Description: "手动选择代理节点，是所有分流的默认出口",
			Proxies:     []string{"自动选择", "故障转移", "香港节点", "台湾节点", "日本节点", "新加坡节点", "美国节点", "手动节点", "其他节点", "DIRECT"},
			UseAll:      false,
		},
		// 3. 全球直连
		{
			Name:        "全球直连",
			Type:        "select",
			Icon:        "target",
			Description: "国内网站、私有网络直接连接",
			Proxies:     []string{"DIRECT", "节点选择"},
		},
		// 4. AI 服务
		{
			Name:        "AI服务",
			Type:        "select",
			Icon:        "bot",
			Description: "ChatGPT、Claude、Gemini 等 AI 服务",
			Proxies:     []string{"节点选择", "美国节点", "日本节点", "新加坡节点", "台湾节点", "手动节点", "自动选择"},
		},
		// 5. 国外媒体
		{
			Name:        "国外媒体",
			Type:        "select",
			Icon:        "globe",
			Description: "YouTube、Spotify、TikTok 等",
			Proxies:     []string{"节点选择", "香港节点", "台湾节点", "日本节点", "新加坡节点", "美国节点", "手动节点", "自动选择"},
		},
		// 6. Netflix
		{
			Name:        "Netflix",
			Type:        "select",
			Icon:        "film",
			Description: "Netflix 奈飞流媒体",
			Proxies:     []string{"节点选择", "新加坡节点", "香港节点", "台湾节点", "日本节点", "美国节点", "手动节点", "自动选择"},
		},
		// 6. 电报消息
		{
			Name:        "电报消息",
			Type:        "select",
			Icon:        "message-circle",
			Description: "Telegram 电报消息服务",
			Proxies:     []string{"节点选择", "香港节点", "台湾节点", "新加坡节点", "手动节点", "自动选择"},
		},
		// 7. 谷歌服务
		{
			Name:        "谷歌服务",
			Type:        "select",
			Icon:        "search",
			Description: "Google 搜索、Gmail、Google Drive 等",
			Proxies:     []string{"节点选择", "香港节点", "台湾节点", "日本节点", "美国节点", "手动节点", "自动选择"},
		},
		// 8. 推特消息
		{
			Name:        "推特消息",
			Type:        "select",
			Icon:        "twitter",
			Description: "Twitter/X 社交平台",
			Proxies:     []string{"节点选择", "香港节点", "台湾节点", "日本节点", "美国节点", "手动节点", "自动选择"},
		},
		// 9. Facebook
		{
			Name:        "脸书服务",
			Type:        "select",
			Icon:        "facebook",
			Description: "Facebook、Instagram、WhatsApp",
			Proxies:     []string{"节点选择", "香港节点", "台湾节点", "美国节点", "手动节点", "自动选择"},
		},
		// 10. 游戏平台
		{
			Name:        "游戏平台",
			Type:        "select",
			Icon:        "gamepad-2",
			Description: "Steam、Epic、PlayStation、Xbox 等游戏平台",
			Proxies:     []string{"节点选择", "香港节点", "台湾节点", "日本节点", "手动节点", "DIRECT"},
		},
		// 11. 哔哩哔哩
		{
			Name:        "哔哩哔哩",
			Type:        "select",
			Icon:        "tv",
			Description: "哔哩哔哩，港澳台番剧解锁",
			Proxies:     []string{"DIRECT", "香港节点", "台湾节点", "手动节点"},
		},
		// 12. 微软服务
		{
			Name:        "微软服务",
			Type:        "select",
			Icon:        "square",
			Description: "Microsoft 365、OneDrive、Azure 等",
			Proxies:     []string{"DIRECT", "节点选择", "香港节点", "美国节点", "手动节点"},
			Enabled:     true,
		},
		// 13. 苹果服务
		{
			Name:        "苹果服务",
			Type:        "select",
			Icon:        "apple",
			Description: "Apple 服务、App Store、iCloud",
			Proxies:     []string{"DIRECT", "节点选择", "美国节点", "手动节点"},
			Enabled:     true,
		},
		// 14. GitHub
		{
			Name:        "GitHub",
			Type:        "select",
			Icon:        "github",
			Description: "GitHub 代码托管平台",
			Proxies:     []string{"节点选择", "DIRECT", "手动节点", "自动选择"},
		},
		// 15. 广告拦截
		{
			Name:        "广告拦截",
			Type:        "select",
			Icon:        "ban",
			Description: "拦截广告、隐私追踪、恶意网站",
			Proxies:     []string{"REJECT", "DIRECT"},
		},
		// 16. 漏网之鱼
		{
			Name:        "漏网之鱼",
			Type:        "select",
			Icon:        "fish",
			Description: "未匹配到任何规则的流量",
			Proxies:     []string{"节点选择", "自动选择", "手动节点", "DIRECT"},
		},
		// === 地区节点分组 ===
		// 17. 香港节点
		{
			Name:        "香港节点",
			Type:        "select",
			Icon:        "flag",
			Description: "香港节点手动选择",
			Proxies:     []string{},
			Filter:      "(?i)香港|沪港|呼港|中港|HKT|HKBN|HGC|WTT|CMI|穗港|广港|京港|🇭🇰|HK|Hongkong|Hong Kong|HongKong|HONG KONG",
			UseAll:      true,
		},
		// 18. 台湾节点
		{
			Name:        "台湾节点",
			Type:        "select",
			Icon:        "flag",
			Description: "台湾节点手动选择",
			Proxies:     []string{},
			Filter:      "(?i)台湾|台灣|臺灣|台北|台中|新北|彰化|CHT|HINET|🇨🇳|🇹🇼|TW|Taiwan|TAIWAN",
			UseAll:      true,
		},
		// 19. 日本节点
		{
			Name:        "日本节点",
			Type:        "select",
			Icon:        "flag",
			Description: "日本节点手动选择",
			Proxies:     []string{},
			Filter:      "(?i)日本|东京|東京|大阪|埼玉|京日|苏日|沪日|广日|上日|穗日|川日|中日|泉日|杭日|深日|🇯🇵|JP|Japan|JAPAN",
			UseAll:      true,
		},
		// 20. 新加坡节点
		{
			Name:        "新加坡节点",
			Type:        "select",
			Icon:        "flag",
			Description: "新加坡节点手动选择",
			Proxies:     []string{},
			Filter:      "(?i)新加坡|狮城|獅城|沪新|京新|泉新|穗新|深新|杭新|广新|廣新|滬新|🇸🇬|SG|Singapore|SINGAPORE",
			UseAll:      true,
		},
		// 21. 美国节点
		{
			Name:        "美国节点",
			Type:        "select",
			Icon:        "flag",
			Description: "美国节点手动选择",
			Proxies:     []string{},
			Filter:      "(?i)美国|美國|京美|硅谷|凤凰城|洛杉矶|西雅图|圣何塞|芝加哥|哥伦布|纽约|广美|🇺🇸|US|USA|America|United States",
			UseAll:      true,
		},
		// 22. 手动节点
		{
			Name:        "手动节点",
			Type:        "select",
			Icon:        "plus-circle",
			Description: "手动添加的节点",
			Proxies:     []string{},
			Filter:      "__MANUAL__", // 特殊标记，用于识别手动节点
			UseAll:      true,
		},
		// 23. 其他节点
		{
			Name:        "其他节点",
			Type:        "select",
			Icon:        "globe",
			Description: "其他地区节点",
			Proxies:     []string{},
			UseAll:      true,
		},
	}

	// 默认全部启用
	for i := range groups {
		groups[i].Enabled = true
	}
	return groups
}

// GetDefaultRuleProviders 获取默认规则提供者
func GetDefaultRuleProviders() []RuleProviderTemplate {
	baseURL := "https://testingcf.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@meta/geo"

	return []RuleProviderTemplate{
		// 基础规则
		{Name: "private-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/private.mrs", Path: "./ruleset/private-domain.mrs", Interval: 86400, Format: "mrs", Description: "私有网络域名"},
		{Name: "private-ip", Type: "http", Behavior: "ipcidr", URL: baseURL + "/geoip/private.mrs", Path: "./ruleset/private-ip.mrs", Interval: 86400, Format: "mrs", Description: "私有网络 IP"},
		{Name: "ads-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/category-ads-all.mrs", Path: "./ruleset/ads-domain.mrs", Interval: 86400, Format: "mrs", Description: "广告域名"},

		// AI 服务 (OpenAI, Claude, Gemini 等)
		{Name: "ai-domain", Type: "http", Behavior: "domain", URL: "https://testingcf.jsdelivr.net/gh/QuixoticHeart/rule-set@ruleset/meta/domain/ai.mrs", Path: "./ruleset/ai-domain.mrs", Interval: 86400, Format: "mrs", Description: "AI 平台域名 (OpenAI, Claude, Gemini)"},

		// 社交媒体
		{Name: "telegram-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/telegram.mrs", Path: "./ruleset/telegram-domain.mrs", Interval: 86400, Format: "mrs", Description: "Telegram 域名"},
		{Name: "telegram-ip", Type: "http", Behavior: "ipcidr", URL: baseURL + "/geoip/telegram.mrs", Path: "./ruleset/telegram-ip.mrs", Interval: 86400, Format: "mrs", Description: "Telegram IP"},
		{Name: "twitter-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/twitter.mrs", Path: "./ruleset/twitter-domain.mrs", Interval: 86400, Format: "mrs", Description: "Twitter/X 域名"},
		{Name: "twitter-ip", Type: "http", Behavior: "ipcidr", URL: baseURL + "/geoip/twitter.mrs", Path: "./ruleset/twitter-ip.mrs", Interval: 86400, Format: "mrs", Description: "Twitter/X IP"},
		{Name: "facebook-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/facebook.mrs", Path: "./ruleset/facebook-domain.mrs", Interval: 86400, Format: "mrs", Description: "Facebook 域名"},

		// 流媒体
		{Name: "youtube-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/youtube.mrs", Path: "./ruleset/youtube-domain.mrs", Interval: 86400, Format: "mrs", Description: "YouTube 域名"},
		{Name: "netflix-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/netflix.mrs", Path: "./ruleset/netflix-domain.mrs", Interval: 86400, Format: "mrs", Description: "Netflix 域名"},
		{Name: "spotify-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/spotify.mrs", Path: "./ruleset/spotify-domain.mrs", Interval: 86400, Format: "mrs", Description: "Spotify 域名"},
		{Name: "tiktok-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/tiktok.mrs", Path: "./ruleset/tiktok-domain.mrs", Interval: 86400, Format: "mrs", Description: "TikTok 域名"},
		{Name: "bilibili-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/bilibili.mrs", Path: "./ruleset/bilibili-domain.mrs", Interval: 86400, Format: "mrs", Description: "哔哩哔哩域名"},

		// Google
		{Name: "google-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/google.mrs", Path: "./ruleset/google-domain.mrs", Interval: 86400, Format: "mrs", Description: "Google 域名"},
		{Name: "google-ip", Type: "http", Behavior: "ipcidr", URL: baseURL + "/geoip/google.mrs", Path: "./ruleset/google-ip.mrs", Interval: 86400, Format: "mrs", Description: "Google IP"},

		// 其他服务
		{Name: "github-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/github.mrs", Path: "./ruleset/github-domain.mrs", Interval: 86400, Format: "mrs", Description: "GitHub 域名"},
		{Name: "microsoft-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/microsoft.mrs", Path: "./ruleset/microsoft-domain.mrs", Interval: 86400, Format: "mrs", Description: "Microsoft 域名"},
		{Name: "apple-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/apple.mrs", Path: "./ruleset/apple-domain.mrs", Interval: 86400, Format: "mrs", Description: "Apple 域名"},
		{Name: "steam-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/steam.mrs", Path: "./ruleset/steam-domain.mrs", Interval: 86400, Format: "mrs", Description: "Steam 域名"},
		{Name: "epic-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/epicgames.mrs", Path: "./ruleset/epic-domain.mrs", Interval: 86400, Format: "mrs", Description: "Epic Games 域名"},

		// 地区规则
		{Name: "cn-domain", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/cn.mrs", Path: "./ruleset/cn-domain.mrs", Interval: 86400, Format: "mrs", Description: "国内域名"},
		{Name: "geolocation-!cn", Type: "http", Behavior: "domain", URL: baseURL + "/geosite/geolocation-!cn.mrs", Path: "./ruleset/proxy-domain.mrs", Interval: 86400, Format: "mrs", Description: "国外域名"},
	}
}

// GetDefaultRules 获取默认规则
// 注意：这里引用的代理组必须在 GetDefaultProxyGroups 中定义
func GetDefaultRules() []RuleTemplate {
	return []RuleTemplate{
		// 私有网络直连
		{Type: "RULE-SET", Payload: "private-domain", Proxy: "全球直连", Description: "私有网络域名直连"},
		{Type: "RULE-SET", Payload: "private-ip", Proxy: "全球直连", NoResolve: true, Description: "私有网络 IP 直连"},

		// 广告拦截
		{Type: "RULE-SET", Payload: "ads-domain", Proxy: "广告拦截", Description: "广告域名拦截"},

		// AI 服务
		{Type: "RULE-SET", Payload: "ai-domain", Proxy: "AI服务", Description: "AI 平台走代理"},

		// 社交媒体
		{Type: "RULE-SET", Payload: "telegram-domain", Proxy: "电报消息", Description: "Telegram 域名"},
		{Type: "RULE-SET", Payload: "telegram-ip", Proxy: "电报消息", NoResolve: true, Description: "Telegram IP"},
		{Type: "RULE-SET", Payload: "twitter-domain", Proxy: "推特消息", Description: "Twitter 域名"},
		{Type: "RULE-SET", Payload: "twitter-ip", Proxy: "推特消息", NoResolve: true, Description: "Twitter IP"},
		{Type: "RULE-SET", Payload: "facebook-domain", Proxy: "脸书服务", Description: "Facebook 域名"},

		// 流媒体
		{Type: "RULE-SET", Payload: "youtube-domain", Proxy: "国外媒体", Description: "YouTube 域名"},
		{Type: "RULE-SET", Payload: "netflix-domain", Proxy: "Netflix", Description: "Netflix 域名"},
		{Type: "RULE-SET", Payload: "spotify-domain", Proxy: "国外媒体", Description: "Spotify 域名"},
		{Type: "RULE-SET", Payload: "tiktok-domain", Proxy: "国外媒体", Description: "TikTok 域名"},
		{Type: "RULE-SET", Payload: "bilibili-domain", Proxy: "哔哩哔哩", Description: "哔哩哔哩域名"},

		// Google
		{Type: "RULE-SET", Payload: "google-domain", Proxy: "谷歌服务", Description: "Google 域名"},
		{Type: "RULE-SET", Payload: "google-ip", Proxy: "谷歌服务", NoResolve: true, Description: "Google IP"},

		// 其他服务
		{Type: "RULE-SET", Payload: "github-domain", Proxy: "GitHub", Description: "GitHub 域名"},
		{Type: "RULE-SET", Payload: "microsoft-domain", Proxy: "微软服务", Description: "Microsoft 域名"},
		{Type: "RULE-SET", Payload: "apple-domain", Proxy: "苹果服务", Description: "Apple 域名"},
		{Type: "RULE-SET", Payload: "steam-domain", Proxy: "游戏平台", Description: "Steam 域名"},
		{Type: "RULE-SET", Payload: "epic-domain", Proxy: "游戏平台", Description: "Epic Games 域名"},

		// 国内直连
		{Type: "RULE-SET", Payload: "cn-domain", Proxy: "全球直连", Description: "国内域名直连"},

		// 国外代理
		{Type: "RULE-SET", Payload: "geolocation-!cn", Proxy: "节点选择", Description: "国外域名走代理"},

		// GeoIP 规则
		{Type: "GEOIP", Payload: "LAN", Proxy: "全球直连", NoResolve: true, Description: "局域网直连"},
		{Type: "GEOIP", Payload: "CN", Proxy: "全球直连", NoResolve: true, Description: "国内 IP 直连"},

		// 兜底规则
		{Type: "MATCH", Payload: "", Proxy: "漏网之鱼", Description: "未匹配规则走漏网之鱼"},
	}
}

// GetDefaultConfigTemplate 获取默认配置模板
func GetDefaultConfigTemplate() *ConfigTemplate {
	return &ConfigTemplate{
		ProxyGroups:   GetDefaultProxyGroups(),
		Rules:         GetDefaultRules(),
		RuleProviders: GetDefaultRuleProviders(),
	}
}
