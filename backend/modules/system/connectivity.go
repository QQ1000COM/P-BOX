package system

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

type ConnectivityResult struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	Success    bool   `json:"success"`
	StatusCode int    `json:"statusCode,omitempty"`
	LatencyMs  int64  `json:"latencyMs"`
	Error      string `json:"error,omitempty"`
}

type DiagnosticItem struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func (h *Handler) TestConnectivity(c *gin.Context) {
	targets := []struct {
		Name string
		URL  string
	}{
		{Name: "Google", URL: "https://www.google.com/generate_204"},
		{Name: "ChatGPT", URL: "https://chatgpt.com/cdn-cgi/trace"},
	}

	client := createProxyClient()
	results := make([]ConnectivityResult, 0, len(targets))

	for _, target := range targets {
		start := time.Now()
		result := ConnectivityResult{Name: target.Name, URL: target.URL}

		resp, err := client.Get(target.URL)
		result.LatencyMs = time.Since(start).Milliseconds()
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}

		result.StatusCode = resp.StatusCode
		result.Success = resp.StatusCode > 0 && resp.StatusCode < 500
		resp.Body.Close()
		results = append(results, result)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"proxy":   "127.0.0.1:7890",
			"targets": results,
		},
	})
}

func (h *Handler) GetDiagnostics(c *gin.Context) {
	items := []DiagnosticItem{
		checkTCPPort("P-BOX panel", "127.0.0.1:8666"),
		checkTCPPort("Mihomo proxy port", "127.0.0.1:7890"),
		checkTCPPort("Mihomo controller", "127.0.0.1:9090"),
		checkFile("Runtime config", filepath.Join(h.service.dataDir, "configs", "config.yaml")),
		checkFile("Proxy settings", filepath.Join(h.service.dataDir, "proxy_settings.json")),
		checkFile("Mihomo core", filepath.Join(h.service.dataDir, "cores", "mihomo-linux-amd64")),
	}

	if _, err := net.LookupHost("chatgpt.com"); err != nil {
		items = append(items, DiagnosticItem{Name: "DNS lookup", Status: "error", Detail: err.Error()})
	} else {
		items = append(items, DiagnosticItem{Name: "DNS lookup", Status: "ok", Detail: "chatgpt.com resolves"})
	}

	status := "ok"
	for _, item := range items {
		if item.Status == "error" {
			status = "error"
			break
		}
		if item.Status == "warn" && status == "ok" {
			status = "warn"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"status": status,
			"items":  items,
		},
	})
}

func (h *Handler) CheckChatGPT(c *gin.Context) {
	targets := []struct {
		Name string
		URL  string
	}{
		{Name: "ChatGPT web", URL: "https://chatgpt.com/cdn-cgi/trace"},
		{Name: "OpenAI API", URL: "https://api.openai.com/v1/models"},
		{Name: "OpenAI static", URL: "https://oaistatic.com/"},
		{Name: "OpenAI content", URL: "https://files.oaiusercontent.com/"},
	}

	client := createProxyClient()
	results := make([]ConnectivityResult, 0, len(targets))
	for _, target := range targets {
		start := time.Now()
		result := ConnectivityResult{Name: target.Name, URL: target.URL}
		resp, err := client.Get(target.URL)
		result.LatencyMs = time.Since(start).Milliseconds()
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			continue
		}
		result.StatusCode = resp.StatusCode
		result.Success = resp.StatusCode > 0 && resp.StatusCode < 500
		resp.Body.Close()
		results = append(results, result)
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"proxy":   "127.0.0.1:7890",
			"targets": results,
		},
	})
}

func checkTCPPort(name, address string) DiagnosticItem {
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return DiagnosticItem{Name: name, Status: "error", Detail: err.Error()}
	}
	conn.Close()
	return DiagnosticItem{Name: name, Status: "ok", Detail: address + " is listening"}
}

func checkFile(name, path string) DiagnosticItem {
	info, err := os.Stat(path)
	if err != nil {
		return DiagnosticItem{Name: name, Status: "error", Detail: err.Error()}
	}
	if info.Size() == 0 {
		return DiagnosticItem{Name: name, Status: "warn", Detail: path + " is empty"}
	}
	return DiagnosticItem{Name: name, Status: "ok", Detail: path}
}
