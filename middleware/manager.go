package middleware

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"sync"
)

// DefaultPluginManager 默认插件管理器实现
type DefaultPluginManager struct {
	plugins   map[string]Plugin
	pluginDir string
	mu        sync.RWMutex
}

// NewPluginManager 创建新的插件管理器
func NewPluginManager(pluginDir string) PluginManager {
	return &DefaultPluginManager{
		plugins:   make(map[string]Plugin),
		pluginDir: pluginDir,
	}
}

// LoadPlugin 加载插件
func (dpm *DefaultPluginManager) LoadPlugin(pluginPath string) error {
	pluginName := filepath.Base(pluginPath)

	dpm.mu.Lock()
	defer dpm.mu.Unlock()

	// 检查插件是否已加载
	if _, exists := dpm.plugins[pluginName]; exists {
		return fmt.Errorf("plugin '%s' is already loaded", pluginName)
	}

	// 读取插件元数据
	metadataPath := filepath.Join(pluginPath, "plugin.json")
	metadata, err := dpm.loadPluginMetadata(metadataPath)
	if err != nil {
		return err
	}

	// 验证插件配置
	if err := ValidatePluginConfig(metadata.Config, GetPluginSchema(metadata.Type)); err != nil {
		return fmt.Errorf("invalid plugin configuration: %v", err)
	}

	// 检查插件是否启用
	if !metadata.Enabled {
		log.Printf("Plugin '%s' is disabled, skipping", pluginName)
		return nil
	}

	// 加载插件SO文件
	soPath := filepath.Join(pluginPath, "plugin.so")
	if _, err := os.Stat(soPath); os.IsNotExist(err) {
		// 如果SO文件不存在，尝试加载Go源文件
		return dpm.loadPluginFromSource(pluginPath, metadata)
	}

	// 加载插件
	p, err := plugin.Open(soPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %v", err)
	}

	// 查找插件入口函数，固定为PluginMain
	symbol, err := p.Lookup("PluginMain")
	if err != nil {
		return fmt.Errorf("failed to lookup entry point 'PluginMain': %v", err)
	}

	// 类型断言
	pluginMain, ok := symbol.(func(config map[string]interface{}) (Middleware, error))
	if !ok {
		return fmt.Errorf("invalid plugin entry point signature")
	}

	// 创建中间件
	middleware, err := pluginMain(metadata.Config)
	if err != nil {
		return fmt.Errorf("failed to create middleware: %v", err)
	}

	// 创建插件包装器
	pluginWrapper := &PluginWrapper{
		name:        metadata.Name,
		version:     metadata.Version,
		description: metadata.Description,
		middleware:  middleware,
		config:      metadata.Config,
		plugin:      p,
	}

	// 存储插件
	dpm.plugins[pluginName] = pluginWrapper

	log.Printf("Successfully loaded plugin '%s' version %s", metadata.Name, metadata.Version)
	return nil
}

// loadPluginFromSource 从Go源文件加载插件
func (dpm *DefaultPluginManager) loadPluginFromSource(pluginPath string, metadata *PluginMetadata) error {
	// 这里可以实现从Go源文件编译并加载插件的逻辑
	// 由于Go的plugin包限制，这通常需要在构建时预编译插件
	log.Printf("Plugin source loading not implemented for '%s', skipping", metadata.Name)
	return fmt.Errorf("plugin source loading not implemented")
}

// loadPluginMetadata 加载插件元数据
func (dpm *DefaultPluginManager) loadPluginMetadata(metadataPath string) (*PluginMetadata, error) {
	data, err := ioutil.ReadFile(metadataPath)
	if err != nil {
		return nil, err
	}

	var metadata PluginMetadata
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return nil, err
	}

	// 返回元数据
	return &metadata, nil
}

// UnloadPlugin 卸载插件
func (dpm *DefaultPluginManager) UnloadPlugin(pluginName string) error {
	dpm.mu.Lock()
	defer dpm.mu.Unlock()

	plugin, exists := dpm.plugins[pluginName]
	if !exists {
		return fmt.Errorf("plugin '%s' not found", pluginName)
	}

	// 停止插件
	if err := plugin.Stop(); err != nil {
		log.Printf("Error stopping plugin '%s': %v", pluginName, err)
	}

	// 从内存中移除
	delete(dpm.plugins, pluginName)

	log.Printf("Successfully unloaded plugin '%s'", pluginName)
	return nil
}

// GetPlugin 获取插件
func (dpm *DefaultPluginManager) GetPlugin(pluginName string) (Plugin, bool) {
	dpm.mu.RLock()
	defer dpm.mu.RUnlock()

	plugin, exists := dpm.plugins[pluginName]
	return plugin, exists
}

// ListPlugins 列出所有插件
func (dpm *DefaultPluginManager) ListPlugins() []Plugin {
	dpm.mu.RLock()
	defer dpm.mu.RUnlock()

	plugins := make([]Plugin, 0, len(dpm.plugins))
	for _, plugin := range dpm.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// ReloadPlugin 重新加载插件
func (dpm *DefaultPluginManager) ReloadPlugin(pluginName string) error {
	// 获取插件路径
	pluginPath := filepath.Join(dpm.pluginDir, pluginName)

	// 卸载现有插件
	if err := dpm.UnloadPlugin(pluginName); err != nil {
		return fmt.Errorf("failed to unload plugin '%s': %v", pluginName, err)
	}

	// 重新加载插件
	if err := dpm.LoadPlugin(pluginPath); err != nil {
		return fmt.Errorf("failed to reload plugin '%s': %v", pluginName, err)
	}

	return nil
}

// DiscoverPlugins 发现插件目录中的所有插件
func (dpm *DefaultPluginManager) DiscoverPlugins() ([]string, error) {
	if _, err := os.Stat(dpm.pluginDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin directory '%s' does not exist", dpm.pluginDir)
	}

	files, err := ioutil.ReadDir(dpm.pluginDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin directory: %v", err)
	}

	var plugins []string
	for _, file := range files {
		if file.IsDir() {
			pluginPath := filepath.Join(dpm.pluginDir, file.Name())
			// 检查是否是有效的插件目录
			if dpm.isValidPluginDir(pluginPath) {
				plugins = append(plugins, file.Name())
			}
		}
	}

	return plugins, nil
}

// LoadAllPlugins 加载插件目录中的所有插件
func (dpm *DefaultPluginManager) LoadAllPlugins() error {
	plugins, err := dpm.DiscoverPlugins()
	if err != nil {
		return err
	}

	for _, pluginName := range plugins {
		pluginPath := filepath.Join(dpm.pluginDir, pluginName)
		if err := dpm.LoadPlugin(pluginPath); err != nil {
			log.Printf("Failed to load plugin '%s': %v", pluginName, err)
		}
	}

	return nil
}

// GetPluginDir 获取插件目录
func (dpm *DefaultPluginManager) GetPluginDir() string {
	return dpm.pluginDir
}

// isValidPluginDir 检查是否是有效的插件目录
func (dpm *DefaultPluginManager) isValidPluginDir(pluginPath string) bool {
	// 检查是否存在plugin.json文件
	metadataPath := filepath.Join(pluginPath, "plugin.json")
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return false
	}

	// 检查是否存在plugin.so或plugin.go文件
	soPath := filepath.Join(pluginPath, "plugin.so")
	goPath := filepath.Join(pluginPath, "plugin.go")

	if _, err := os.Stat(soPath); os.IsNotExist(err) {
		if _, err := os.Stat(goPath); os.IsNotExist(err) {
			return false
		}
	}

	return true
}

// PluginWrapper 插件包装器，用于将中间件包装为插件
type PluginWrapper struct {
	name        string
	version     string
	description string
	middleware  Middleware
	config      map[string]interface{}
	plugin      *plugin.Plugin
}

// Name 返回插件名称
func (pw *PluginWrapper) Name() string {
	return pw.name
}

// Version 返回插件版本
func (pw *PluginWrapper) Version() string {
	return pw.version
}

// Description 返回插件描述
func (pw *PluginWrapper) Description() string {
	return pw.description
}

// Init 初始化插件
func (pw *PluginWrapper) Init(config map[string]interface{}) error {
	// 插件已在加载时初始化
	return nil
}

// CreateMiddleware 创建中间件实例
func (pw *PluginWrapper) CreateMiddleware() (Middleware, error) {
	return pw.middleware, nil
}

// Stop 停止插件
func (pw *PluginWrapper) Stop() error {
	// 插件是中间件，通常不需要特殊停止逻辑
	return nil
}
