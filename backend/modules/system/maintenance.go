package system

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type SnapshotInfo struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"createdAt"`
	Files     int       `json:"files"`
}

type SecurityAuditItem struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func (h *Handler) DownloadBackup(c *gin.Context) {
	name := fmt.Sprintf("p-box-backup-%s.zip", time.Now().Format("20060102-150405"))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", name))

	writer := zip.NewWriter(c.Writer)
	defer writer.Close()
	_ = addDirToZip(writer, h.service.dataDir, h.service.dataDir, func(path string) bool {
		clean := filepath.Clean(path)
		return !strings.Contains(clean, string(filepath.Separator)+"cores"+string(filepath.Separator))
	})
}

func (h *Handler) ListHistory(c *gin.Context) {
	items, err := h.listSnapshots()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

func (h *Handler) CreateSnapshot(c *gin.Context) {
	var req struct {
		Label string `json:"label"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Label == "" {
		req.Label = "手动快照"
	}

	id := time.Now().Format("20060102-150405")
	target := filepath.Join(h.historyDir(), id)
	if err := os.MkdirAll(target, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}

	files, err := copyDir(h.service.dataDir, target, func(path string) bool {
		clean := filepath.Clean(path)
		return !strings.Contains(clean, string(filepath.Separator)+"cores"+string(filepath.Separator)) &&
			!strings.Contains(clean, string(filepath.Separator)+"history"+string(filepath.Separator))
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}

	info := SnapshotInfo{ID: id, Label: req.Label, CreatedAt: time.Now(), Files: files}
	data, _ := json.MarshalIndent(info, "", "  ")
	_ = os.WriteFile(filepath.Join(target, "snapshot.json"), data, 0644)

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": info})
}

func (h *Handler) RestoreSnapshot(c *gin.Context) {
	id := filepath.Base(c.Param("id"))
	source := filepath.Join(h.historyDir(), id)
	if _, err := os.Stat(source); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 1, "message": "snapshot not found"})
		return
	}

	_, err := copyDir(source, h.service.dataDir, func(path string) bool {
		return filepath.Base(path) != "snapshot.json"
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 1, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success"})
}

func (h *Handler) SecurityAudit(c *gin.Context) {
	items := []SecurityAuditItem{}
	authPath := filepath.Join(h.service.dataDir, "auth.json")
	if _, err := os.Stat(authPath); err == nil {
		items = append(items, SecurityAuditItem{Name: "登录认证", Status: "ok", Detail: "已找到认证配置文件"})
	} else {
		items = append(items, SecurityAuditItem{Name: "登录认证", Status: "warn", Detail: "未找到认证配置文件，建议在公网环境启用登录认证"})
	}

	coreSources := filepath.Join(h.service.dataDir, "core_sources.json")
	if data, err := os.ReadFile(coreSources); err == nil && strings.Contains(string(data), "https://") {
		items = append(items, SecurityAuditItem{Name: "更新源", Status: "ok", Detail: "核心更新源使用 HTTPS"})
	} else {
		items = append(items, SecurityAuditItem{Name: "更新源", Status: "warn", Detail: "未配置自定义更新源，使用默认 GitHub 镜像"})
	}

	if _, err := os.Stat(filepath.Join(h.service.dataDir, "cores")); err == nil {
		items = append(items, SecurityAuditItem{Name: "核心目录", Status: "ok", Detail: "核心目录存在"})
	} else {
		items = append(items, SecurityAuditItem{Name: "核心目录", Status: "warn", Detail: "核心目录不存在，启动后会自动创建"})
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": items})
}

func (h *Handler) RuleTemplates(c *gin.Context) {
	templates := []gin.H{
		{"name": "国内直连", "target": "DIRECT", "rules": []string{"GEOSITE,cn,DIRECT", "GEOIP,cn,DIRECT", "DOMAIN-SUFFIX,local,DIRECT"}},
		{"name": "AI 服务代理", "target": "PROXY", "rules": []string{"DOMAIN-SUFFIX,openai.com,PROXY", "DOMAIN-SUFFIX,chatgpt.com,PROXY", "DOMAIN-SUFFIX,anthropic.com,PROXY"}},
		{"name": "流媒体代理", "target": "PROXY", "rules": []string{"GEOSITE,netflix,PROXY", "GEOSITE,youtube,PROXY", "GEOSITE,disney,PROXY"}},
		{"name": "广告拦截", "target": "REJECT", "rules": []string{"GEOSITE,category-ads-all,REJECT"}},
		{"name": "开发者服务", "target": "PROXY", "rules": []string{"DOMAIN-SUFFIX,github.com,PROXY", "DOMAIN-SUFFIX,githubusercontent.com,PROXY", "DOMAIN-SUFFIX,npmjs.org,PROXY"}},
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": templates})
}

func (h *Handler) historyDir() string {
	return filepath.Join(h.service.dataDir, "history")
}

func (h *Handler) listSnapshots() ([]SnapshotInfo, error) {
	entries, err := os.ReadDir(h.historyDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []SnapshotInfo{}, nil
		}
		return nil, err
	}
	items := []SnapshotInfo{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(h.historyDir(), entry.Name(), "snapshot.json"))
		if err != nil {
			continue
		}
		var info SnapshotInfo
		if json.Unmarshal(data, &info) == nil {
			items = append(items, info)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func addDirToZip(writer *zip.Writer, root, base string, include func(string) bool) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !include(path) {
			return err
		}
		rel, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		fileWriter, err := writer.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(fileWriter, file)
		return err
	})
}

func copyDir(source, target string, include func(string) bool) (int, error) {
	count := 0
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !include(path) {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dest := filepath.Join(target, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer output.Close()
		if _, err := io.Copy(output, input); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}
