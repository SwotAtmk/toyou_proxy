package middleware

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
)

// AutoPluginManager 自动插件管理器，负责自动编译和加载插件
type AutoPluginManager struct {
	plugins       map[string]*plugin.Plugin
	pluginSources map[string]string // 插件源代码路径
	cacheDir      string             // 缓存目录
	sourceDir     string             // 插件源代码目录
	mu            sync.RWMutex
}

// NewAutoPluginManager 创建新的自动插件管理器
func NewAutoPluginManager(sourceDir, cacheDir string) *AutoPluginManager {
	// 确保缓存目录存在
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Printf("Failed to create cache directory: %v", err)
	}

	return &AutoPluginManager{
		plugins:       make(map[string]*plugin.Plugin),
		pluginSources: make(map[string]string),
		cacheDir:      cacheDir,
		sourceDir:     sourceDir,
	}
}

// LoadPlugin 加载插件，如果缓存中没有则自动编译
func (apm *AutoPluginManager) LoadPlugin(pluginName string) (*plugin.Plugin, error) {
	apm.mu.Lock()
	defer apm.mu.Unlock()

	// 检查插件是否已经加载
	if p, exists := apm.plugins[pluginName]; exists {
		return p, nil
	}

	// 检查缓存目录中是否有编译好的so文件
	cachePath := filepath.Join(apm.cacheDir, pluginName+".so")
	if _, err := os.Stat(cachePath); err == nil {
		// 缓存文件存在，直接加载
		log.Printf("Loading plugin '%s' from cache", pluginName)
		return apm.loadPluginFromCache(pluginName, cachePath)
	}

	// 缓存文件不存在，尝试从源代码编译
	sourcePath := filepath.Join(apm.sourceDir, pluginName)
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin source directory '%s' does not exist", sourcePath)
	}

	// 编译插件
	log.Printf("Compiling plugin '%s' from source", pluginName)
	if err := apm.compilePlugin(pluginName, sourcePath, cachePath); err != nil {
		return nil, fmt.Errorf("failed to compile plugin '%s': %v", pluginName, err)
	}

	// 从缓存加载编译好的插件
	return apm.loadPluginFromCache(pluginName, cachePath)
}

// loadPluginFromCache 从缓存加载插件
func (apm *AutoPluginManager) loadPluginFromCache(pluginName, cachePath string) (*plugin.Plugin, error) {
	p, err := plugin.Open(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin from cache: %v", err)
	}

	// 存储插件引用
	apm.plugins[pluginName] = p
	apm.pluginSources[pluginName] = cachePath

	log.Printf("Successfully loaded plugin '%s' from cache", pluginName)
	return p, nil
}

// compilePlugin 编译插件
func (apm *AutoPluginManager) compilePlugin(pluginName, sourcePath, cachePath string) error {
	// 查找插件源文件
	goFiles, err := apm.findGoFiles(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to find Go source files: %v", err)
	}

	if len(goFiles) == 0 {
		return fmt.Errorf("no Go source files found in %s", sourcePath)
	}

	// 确保缓存目录存在
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %v", err)
	}

	// 准备编译命令，使用绝对路径
	absCachePath, err := filepath.Abs(cachePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for cache: %v", err)
	}

	args := []string{"build", "-buildmode=plugin", "-o", absCachePath}
	
	// 添加源文件的完整路径
	for _, goFile := range goFiles {
		absGoFile := filepath.Join(sourcePath, goFile)
		args = append(args, absGoFile)
	}

	// 执行编译命令，不在插件源代码目录中执行，而是在项目根目录
	cmd := exec.Command("go", args...)

	// 捕获输出
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compilation failed: %v\nOutput: %s", err, string(output))
	}

	log.Printf("Successfully compiled plugin '%s' to %s", pluginName, cachePath)
	return nil
}

// findGoFiles 查找目录中的所有Go源文件
func (apm *AutoPluginManager) findGoFiles(dir string) ([]string, error) {
	var goFiles []string

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".go") {
			goFiles = append(goFiles, file.Name())
		}
	}

	return goFiles, nil
}

// GetPluginCreator 获取插件创建函数
func (apm *AutoPluginManager) GetPluginCreator(pluginName string) (func(map[string]interface{}) (Middleware, error), error) {
	p, err := apm.LoadPlugin(pluginName)
	if err != nil {
		return nil, err
	}

	// 查找插件入口函数
	symbol, err := p.Lookup("PluginMain")
	if err != nil {
		return nil, fmt.Errorf("failed to lookup entry point 'PluginMain': %v", err)
	}

	// 类型断言
	pluginMain, ok := symbol.(func(map[string]interface{}) (Middleware, error))
	if !ok {
		return nil, fmt.Errorf("invalid plugin entry point signature")
	}

	return pluginMain, nil
}

// GetPluginMetadata 获取插件元数据
func (apm *AutoPluginManager) GetPluginMetadata(pluginName string) (*PluginMetadata, error) {
	// 检查插件元数据文件
	metadataPath := filepath.Join(apm.sourceDir, pluginName, "plugin.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		// 如果没有元数据文件，返回默认元数据
		return &PluginMetadata{
			Name:        pluginName,
			Version:     "1.0.0",
			Description: fmt.Sprintf("Auto-loaded plugin: %s", pluginName),
			Type:        "middleware",
			Config:      make(map[string]interface{}),
			Enabled:     true,
		}, nil
	}

	// 读取元数据文件
	data, err := ioutil.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin metadata: %v", err)
	}

	var metadata PluginMetadata
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse plugin metadata: %v", err)
	}

	return &metadata, nil
}

// DiscoverPlugins 发现所有可用的插件
func (apm *AutoPluginManager) DiscoverPlugins() ([]string, error) {
	if _, err := os.Stat(apm.sourceDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin source directory '%s' does not exist", apm.sourceDir)
	}

	files, err := ioutil.ReadDir(apm.sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin source directory: %v", err)
	}

	var plugins []string
	for _, file := range files {
		if file.IsDir() {
			pluginPath := filepath.Join(apm.sourceDir, file.Name())
			// 检查是否是有效的插件目录
			if apm.isValidPluginDir(pluginPath) {
				plugins = append(plugins, file.Name())
			}
		}
	}

	return plugins, nil
}

// isValidPluginDir 检查是否是有效的插件目录
func (apm *AutoPluginManager) isValidPluginDir(pluginPath string) bool {
	// 检查是否存在plugin.go文件
	goPath := filepath.Join(pluginPath, "plugin.go")
	if _, err := os.Stat(goPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// ReloadPlugin 重新加载插件
func (apm *AutoPluginManager) ReloadPlugin(pluginName string) error {
	apm.mu.Lock()
	defer apm.mu.Unlock()

	// 从内存中移除插件
	delete(apm.plugins, pluginName)
	delete(apm.pluginSources, pluginName)

	// 删除缓存文件
	cachePath := filepath.Join(apm.cacheDir, pluginName+".so")
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		log.Printf("Failed to remove cache file for plugin '%s': %v", pluginName, err)
	}

	// 重新加载插件
	_, err := apm.LoadPlugin(pluginName)
	return err
}

// ClearCache 清空缓存目录
func (apm *AutoPluginManager) ClearCache() error {
	apm.mu.Lock()
	defer apm.mu.Unlock()

	// 清空内存中的插件引用
	apm.plugins = make(map[string]*plugin.Plugin)
	apm.pluginSources = make(map[string]string)

	// 删除缓存目录中的所有文件
	files, err := ioutil.ReadDir(apm.cacheDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".so") {
			cachePath := filepath.Join(apm.cacheDir, file.Name())
			if err := os.Remove(cachePath); err != nil {
				log.Printf("Failed to remove cache file '%s': %v", cachePath, err)
			}
		}
	}

	log.Println("Plugin cache cleared")
	return nil
}