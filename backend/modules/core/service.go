package core

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type CoreType string

const (
	CoreTypeMihomo  CoreType = "mihomo"
	CoreTypeSingbox CoreType = "singbox"
)

// CDN 镜像地址
const (
	MihomoReleaseAPI     = "https://api.github.com/repos/MetaCubeX/mihomo/releases/latest"
	SingboxReleaseAPI    = "https://api.github.com/repos/SagerNet/sing-box/releases/latest"
	GitHubDownloadMirror = "https://ghfast.top/"
)

type CoreStatus struct {
	CurrentCore CoreType         `json:"currentCore"`
	Cores       map[string]*Core `json:"cores"`
}

type Core struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	LatestVersion string `json:"latestVersion"`
	Installed     bool   `json:"installed"`
	Path          string `json:"path"`
}

type DownloadProgress struct {
	Downloading bool    `json:"downloading"`
	Progress    float64 `json:"progress"`
	Speed       int64   `json:"speed"`
	Error       string  `json:"error,omitempty"`
}

type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []GitHubAsset `json:"assets"`
}

type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Service struct {
	dataDir          string
	currentCore      CoreType
	cores            map[string]*Core
	downloadProgress map[string]*DownloadProgress
	mu               sync.RWMutex
	onCoreSwitch     func(coreType string) // 核心切换回调
}

// 持久化状态
type SavedCoreStatus struct {
	CurrentCore    string            `json:"currentCore"`
	Versions       map[string]string `json:"versions"`
	LatestVersions map[string]string `json:"latestVersions"`
	LastChecked    time.Time         `json:"lastChecked"`
}

func NewService(dataDir string) *Service {
	s := &Service{
		dataDir:          dataDir,
		currentCore:      CoreTypeMihomo,
		cores:            make(map[string]*Core),
		downloadProgress: make(map[string]*DownloadProgress),
	}

	s.cores["mihomo"] = &Core{
		Name:      "Mihomo",
		Installed: false,
		Path:      filepath.Join(dataDir, "cores", "mihomo"),
	}
	s.cores["singbox"] = &Core{
		Name:      "sing-box",
		Installed: false,
		Path:      filepath.Join(dataDir, "cores", "sing-box"),
	}
	s.cores["mihomo"].Path = s.getCoreBinaryPath("mihomo")
	s.cores["singbox"].Path = s.getCoreBinaryPath("singbox")

	s.loadSavedStatus()
	s.checkInstalledCores()
	return s
}

func (s *Service) loadSavedStatus() {
	filePath := filepath.Join(s.dataDir, "core_status.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}

	var saved SavedCoreStatus
	if err := json.Unmarshal(data, &saved); err != nil {
		return
	}

	if saved.CurrentCore != "" {
		s.currentCore = CoreType(saved.CurrentCore)
	}

	for name, version := range saved.Versions {
		if core, ok := s.cores[name]; ok {
			core.Version = version
			if version != "" {
				core.Installed = true
			}
		}
	}

	// 加载保存的最新版本信息
	for name, latestVersion := range saved.LatestVersions {
		if core, ok := s.cores[name]; ok {
			core.LatestVersion = latestVersion
		}
	}
}

func (s *Service) saveStatus() error {
	s.mu.RLock()
	saved := SavedCoreStatus{
		CurrentCore:    string(s.currentCore),
		Versions:       make(map[string]string),
		LatestVersions: make(map[string]string),
		LastChecked:    time.Now(),
	}
	for name, core := range s.cores {
		if core.Installed {
			saved.Versions[name] = core.Version
		}
		if core.LatestVersion != "" {
			saved.LatestVersions[name] = core.LatestVersion
		}
	}
	s.mu.RUnlock()

	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(s.dataDir, "core_status.json")
	return os.WriteFile(filePath, data, 0644)
}

func (s *Service) checkInstalledCores() {
	for name, core := range s.cores {
		binPath := s.getCoreBinaryPath(name)
		if _, err := os.Stat(binPath); err == nil {
			core.Installed = true
			core.Version = s.getCoreVersion(name)
		}
	}
}

func (s *Service) getCoreBinaryPath(coreType string) string {
	arch := runtime.GOARCH
	goos := runtime.GOOS

	var binName string
	switch coreType {
	case "mihomo":
		binName = fmt.Sprintf("mihomo-%s-%s", goos, arch)
	case "singbox":
		binName = fmt.Sprintf("sing-box-%s-%s", goos, arch)
	}

	if goos == "windows" {
		binName += ".exe"
	}

	return filepath.Join(s.dataDir, "cores", binName)
}

func (s *Service) getCoreVersion(coreType string) string {
	binPath := s.getCoreBinaryPath(coreType)

	// 执行核心获取版本
	var cmd *exec.Cmd
	switch coreType {
	case "mihomo":
		cmd = exec.Command(binPath, "-v")
	case "singbox":
		cmd = exec.Command(binPath, "version")
	default:
		return "unknown"
	}

	output, err := cmd.Output()
	if err != nil {
		// 如果有保存的版本，使用保存的
		if core, ok := s.cores[coreType]; ok && core.Version != "" {
			return core.Version
		}
		return "unknown"
	}

	// 解析版本号
	outputStr := string(output)
	version := s.parseVersionFromOutput(coreType, outputStr)
	if version != "" {
		return version
	}

	return "unknown"
}

// parseVersionFromOutput 从输出中解析版本号
func (s *Service) parseVersionFromOutput(coreType, output string) string {
	lines := strings.Split(output, "\n")

	switch coreType {
	case "mihomo":
		// Mihomo v1.18.10 darwin arm64 with go1.23.2
		for _, line := range lines {
			if strings.Contains(line, "Mihomo") || strings.Contains(line, "mihomo") {
				parts := strings.Fields(line)
				for _, part := range parts {
					if strings.HasPrefix(part, "v") || strings.HasPrefix(part, "V") {
						return strings.TrimPrefix(strings.TrimPrefix(part, "v"), "V")
					}
				}
			}
		}
	case "singbox":
		// sing-box version 1.10.5
		for _, line := range lines {
			if strings.Contains(line, "version") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					return parts[len(parts)-1]
				}
			}
			// 或者直接输出版本号
			line = strings.TrimSpace(line)
			if line != "" && !strings.Contains(line, " ") {
				return line
			}
		}
	}

	return ""
}

func (s *Service) GetStatus() *CoreStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &CoreStatus{
		CurrentCore: s.currentCore,
		Cores:       s.cores,
	}
}

func (s *Service) GetLatestVersions() (map[string]string, error) {
	versions := make(map[string]string)

	mihomoVersion, err := s.fetchMihomoLatestVersion()
	if err == nil {
		versions["mihomo"] = mihomoVersion
		s.mu.Lock()
		s.cores["mihomo"].LatestVersion = mihomoVersion
		s.mu.Unlock()
	}

	singboxVersion, err := s.fetchSingboxLatestVersion()
	if err == nil {
		versions["singbox"] = singboxVersion
		s.mu.Lock()
		s.cores["singbox"].LatestVersion = singboxVersion
		s.mu.Unlock()
	}

	return versions, nil
}

func (s *Service) fetchMihomoLatestVersion() (string, error) {
	release, err := s.fetchLatestRelease(MihomoReleaseAPI)
	if err != nil {
		return "", err
	}
	return normalizeVersion(release.TagName), nil
}

func (s *Service) fetchSingboxLatestVersion() (string, error) {
	release, err := s.fetchLatestRelease(SingboxReleaseAPI)
	if err != nil {
		return "", err
	}

	return normalizeVersion(release.TagName), nil
}

func (s *Service) fetchLatestRelease(apiURL string) (*GitHubRelease, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "P-BOX")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github release api returned HTTP %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	if release.TagName == "" {
		return nil, fmt.Errorf("release tag not found")
	}
	return &release, nil
}

func normalizeVersion(tag string) string {
	return strings.TrimPrefix(strings.TrimPrefix(tag, "v"), "V")
}

func (s *Service) SwitchCore(coreType string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	core, ok := s.cores[coreType]
	if !ok {
		return fmt.Errorf("unknown core type: %s", coreType)
	}

	if !core.Installed {
		return fmt.Errorf("core %s is not installed", coreType)
	}

	s.currentCore = CoreType(coreType)

	// 通知 proxy 模块切换核心
	if s.onCoreSwitch != nil {
		s.onCoreSwitch(coreType)
	}

	// 持久化保存
	go s.saveStatus()

	return nil
}

// SetOnCoreSwitch 设置核心切换回调
func (s *Service) SetOnCoreSwitch(callback func(coreType string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onCoreSwitch = callback
}

// GetCurrentCore 获取当前核心类型
func (s *Service) GetCurrentCore() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return string(s.currentCore)
}

func (s *Service) DownloadCore(coreType string) error {
	s.mu.Lock()
	s.downloadProgress[coreType] = &DownloadProgress{Downloading: true}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.downloadProgress[coreType].Downloading = false
		s.mu.Unlock()
	}()

	downloadURLs, err := s.getCoreDownloadURLs(coreType)
	if err != nil {
		s.mu.Lock()
		s.downloadProgress[coreType].Error = err.Error()
		s.mu.Unlock()
		return err
	}

	var lastErr error
	for _, downloadURL := range downloadURLs {
		fmt.Printf("Downloading %s from %s\n", coreType, downloadURL)
		if err = s.downloadFromURL(coreType, downloadURL); err == nil {
			fmt.Printf("%s download completed\n", coreType)
			return nil
		}
		lastErr = err
		fmt.Printf("Download failed: %v\n", err)
	}

	s.mu.Lock()
	s.downloadProgress[coreType].Error = lastErr.Error()
	s.mu.Unlock()
	return fmt.Errorf("download failed: %v", lastErr)
}

// downloadFromURL 从指定 URL 下载核心
func (s *Service) downloadFromURL(coreType, downloadURL string) error {
	// 创建带超时的 HTTP 客户端
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	os.MkdirAll(filepath.Join(s.dataDir, "cores"), 0755)

	// 下载到临时文件
	tmpFile := filepath.Join(s.dataDir, "cores", "download.tmp")
	out, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	totalSize := resp.ContentLength
	written := int64(0)

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			written += int64(n)

			s.mu.Lock()
			if totalSize > 0 {
				s.downloadProgress[coreType].Progress = float64(written) / float64(totalSize) * 100
			}
			s.mu.Unlock()
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			out.Close()
			os.Remove(tmpFile)
			return err
		}
	}
	out.Close()

	// 解压文件
	binPath := s.getCoreBinaryPath(coreType)
	if err := s.extractCore(tmpFile, binPath, coreType); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("解压失败: %v", err)
	}
	os.Remove(tmpFile)

	// 设置执行权限
	os.Chmod(binPath, 0755)

	s.mu.Lock()
	s.cores[coreType].Installed = true
	s.cores[coreType].Version = s.cores[coreType].LatestVersion
	s.mu.Unlock()

	// 持久化保存
	s.saveStatus()

	return nil
}

// extractCore 解压核心文件
func (s *Service) extractCore(archivePath, destPath, coreType string) error {
	if s.isZipFile(archivePath) {
		return s.extractZipCore(archivePath, destPath, coreType)
	}

	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 创建 gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("gzip open failed: %v", err)
	}
	defer gzr.Close()

	// Mihomo 是单文件 .gz，sing-box 是 .tar.gz
	if coreType == "mihomo" {
		// 直接解压 gzip
		outFile, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, gzr)
		return err
	}

	// sing-box: tar.gz 格式
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// 查找可执行文件
		if header.Typeflag == tar.TypeReg && strings.Contains(header.Name, "sing-box") {
			outFile, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, tr)
			return err
		}
	}

	return fmt.Errorf("executable not found in archive")
}

func (s *Service) isZipFile(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	signature := make([]byte, 4)
	if _, err := io.ReadFull(file, signature); err != nil {
		return false
	}
	return string(signature) == "PK\x03\x04"
}

func (s *Service) extractZipCore(archivePath, destPath, coreType string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("zip open failed: %v", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		if file.FileInfo().IsDir() || !s.isCoreExecutableName(file.Name, coreType) {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}

		outFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return err
		}

		_, copyErr := io.Copy(outFile, rc)
		closeErr := outFile.Close()
		rc.Close()

		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}

	return fmt.Errorf("executable not found in archive")
}

func (s *Service) isCoreExecutableName(name, coreType string) bool {
	base := strings.ToLower(filepath.Base(name))
	switch coreType {
	case "mihomo":
		return base == "mihomo" || base == "mihomo.exe"
	case "singbox":
		return base == "sing-box" || base == "sing-box.exe"
	default:
		return false
	}
}

// getCoreDownloadURLs 获取下载 URL（CDN 优先，官方备用）
func (s *Service) getCoreDownloadURLs(coreType string) ([]string, error) {
	arch := runtime.GOARCH
	goos := runtime.GOOS

	s.mu.RLock()
	core, ok := s.cores[coreType]
	if !ok {
		s.mu.RUnlock()
		return nil, fmt.Errorf("unknown core type")
	}
	version := core.LatestVersion
	s.mu.RUnlock()

	if version == "" {
		return nil, fmt.Errorf("version not found, please check latest version first")
	}

	var apiURL string
	switch coreType {
	case "mihomo":
		apiURL = MihomoReleaseAPI
	case "singbox":
		apiURL = SingboxReleaseAPI
	default:
		return nil, fmt.Errorf("unknown core type")
	}

	release, err := s.fetchLatestRelease(apiURL)
	if err != nil {
		return nil, err
	}

	releaseVersion := normalizeVersion(release.TagName)
	if releaseVersion != "" && releaseVersion != version {
		version = releaseVersion
		s.mu.Lock()
		if core, ok := s.cores[coreType]; ok {
			core.LatestVersion = version
		}
		s.mu.Unlock()
	}

	assetName := s.expectedAssetName(coreType, goos, arch, version)
	assetURL := s.findAssetURL(release.Assets, assetName, coreType, goos, arch, version)
	if assetURL == "" {
		return nil, fmt.Errorf("no %s asset found for %s/%s in %s", coreType, goos, arch, release.TagName)
	}

	if strings.HasPrefix(assetURL, "https://github.com/") {
		return []string{GitHubDownloadMirror + assetURL, assetURL}, nil
	}
	return []string{assetURL}, nil
}

func (s *Service) expectedAssetName(coreType, goos, arch, version string) string {
	switch coreType {
	case "mihomo":
		if goos == "windows" {
			return fmt.Sprintf("mihomo-%s-%s-v%s.zip", goos, arch, version)
		}
		return fmt.Sprintf("mihomo-%s-%s-v%s.gz", goos, arch, version)
	case "singbox":
		if goos == "windows" {
			return fmt.Sprintf("sing-box-%s-%s-%s.zip", version, goos, arch)
		}
		return fmt.Sprintf("sing-box-%s-%s-%s.tar.gz", version, goos, arch)
	default:
		return ""
	}
}

func (s *Service) findAssetURL(assets []GitHubAsset, expectedName, coreType, goos, arch, version string) string {
	for _, asset := range assets {
		if asset.Name == expectedName {
			return asset.BrowserDownloadURL
		}
	}

	prefix := ""
	suffix := ""
	switch coreType {
	case "mihomo":
		prefix = fmt.Sprintf("mihomo-%s-%s-", goos, arch)
		if goos == "windows" {
			suffix = fmt.Sprintf("-v%s.zip", version)
		} else {
			suffix = fmt.Sprintf("-v%s.gz", version)
		}
	case "singbox":
		prefix = fmt.Sprintf("sing-box-%s-%s-%s", version, goos, arch)
		if goos == "windows" {
			suffix = ".zip"
		} else {
			suffix = ".tar.gz"
		}
	}

	for _, asset := range assets {
		if strings.HasPrefix(asset.Name, prefix) && strings.HasSuffix(asset.Name, suffix) {
			name := strings.ToLower(asset.Name)
			if strings.Contains(name, "legacy") || strings.Contains(name, "glibc") ||
				strings.Contains(name, "musl") || strings.Contains(name, "go12") {
				continue
			}
			return asset.BrowserDownloadURL
		}
	}

	for _, asset := range assets {
		if strings.HasPrefix(asset.Name, prefix) && strings.HasSuffix(asset.Name, suffix) {
			return asset.BrowserDownloadURL
		}
	}

	return ""
}

func (s *Service) GetDownloadProgress(coreType string) *DownloadProgress {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if progress, ok := s.downloadProgress[coreType]; ok {
		return progress
	}
	return &DownloadProgress{}
}

// Initialize 启动时自动初始化（延迟执行）
// delaySeconds: 启动后延迟多少秒执行检测
func (s *Service) Initialize(delaySeconds int) {
	go func() {
		// 延迟执行
		time.Sleep(time.Duration(delaySeconds) * time.Second)
		fmt.Printf("🔍 开始自动检测核心版本...\n")

		// 1. 检测最新版本
		s.GetLatestVersions()

		// 2. Ensure both built-in cores are available.
		for _, coreType := range []string{"mihomo", "singbox"} {
			s.mu.RLock()
			coreInstalled := s.cores[coreType].Installed
			coreLatestVersion := s.cores[coreType].LatestVersion
			s.mu.RUnlock()

			if !coreInstalled && coreLatestVersion != "" {
				fmt.Printf("Auto downloading missing %s core...\n", coreType)
				fmt.Printf("   platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
				if err := s.DownloadCore(coreType); err != nil {
					fmt.Printf("Auto download %s failed: %v\n", coreType, err)
				} else {
					fmt.Printf("%s core auto download completed\n", coreType)
				}
			}
		}

		// 保存状态
		s.saveStatus()
		fmt.Printf("✅ 核心版本检测完成\n")
	}()
}

// RefreshVersions 手动刷新版本信息（前端点击刷新时调用）
func (s *Service) RefreshVersions() (map[string]string, error) {
	fmt.Printf("🔄 手动刷新核心版本信息...\n")

	versions, err := s.GetLatestVersions()
	if err != nil {
		return nil, err
	}

	// 保存到文件
	s.saveStatus()

	fmt.Printf("✅ 版本信息已更新并保存\n")
	return versions, nil
}

// GetPlatformInfo 获取当前平台信息
func (s *Service) GetPlatformInfo() map[string]string {
	return map[string]string{
		"os":   runtime.GOOS,
		"arch": runtime.GOARCH,
	}
}
