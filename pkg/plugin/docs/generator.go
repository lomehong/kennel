package docs

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/hashicorp/go-hclog"
	"github.com/lomehong/kennel/pkg/plugin/api"
	"github.com/lomehong/kennel/pkg/plugin/registry"
)

// DocGenerator 文档生成器
// 负责生成插件文档
type DocGenerator struct {
	// 插件注册表
	registry registry.PluginRegistry
	
	// 日志记录器
	logger hclog.Logger
	
	// 文档模板
	templates map[string]*template.Template
}

// NewDocGenerator 创建一个新的文档生成器
func NewDocGenerator(registry registry.PluginRegistry, logger hclog.Logger) *DocGenerator {
	if logger == nil {
		logger = hclog.NewNullLogger()
	}
	
	generator := &DocGenerator{
		registry:  registry,
		logger:    logger.Named("doc-generator"),
		templates: make(map[string]*template.Template),
	}
	
	// 加载默认模板
	generator.loadDefaultTemplates()
	
	return generator
}

// loadDefaultTemplates 加载默认模板
func (g *DocGenerator) loadDefaultTemplates() {
	// 插件列表模板
	pluginListTemplate := `# 插件列表

本文档列出了系统中所有可用的插件。

| 插件ID | 名称 | 版本 | 描述 | 作者 |
|--------|------|------|------|------|
{{- range .Plugins }}
| {{ .ID }} | {{ .Name }} | {{ .Version }} | {{ .Description }} | {{ .Author }} |
{{- end }}

## 插件依赖关系

以下是插件之间的依赖关系：

{{- range .Plugins }}
### {{ .Name }} ({{ .ID }})

{{- if .Dependencies }}
依赖于以下插件：

{{- range .Dependencies }}
- {{ .ID }}{{ if .Version }} (版本要求: {{ .Version }}){{ end }}{{ if .Optional }} (可选){{ end }}
{{- end }}
{{- else }}
没有依赖其他插件。
{{- end }}

{{- end }}
`

	// 插件详情模板
	pluginDetailTemplate := `# {{ .Name }} ({{ .ID }})

**版本:** {{ .Version }}

**作者:** {{ .Author }}

**许可证:** {{ .License }}

## 描述

{{ .Description }}

## 标签

{{- if .Tags }}
{{- range .Tags }}
- {{ . }}
{{- end }}
{{- else }}
无标签。
{{- end }}

## 能力

{{- if .Capabilities }}
{{- range $key, $value := .Capabilities }}
- {{ $key }}: {{ $value }}
{{- end }}
{{- else }}
无特殊能力。
{{- end }}

## 依赖

{{- if .Dependencies }}
本插件依赖于以下插件：

{{- range .Dependencies }}
- {{ .ID }}{{ if .Version }} (版本要求: {{ .Version }}){{ end }}{{ if .Optional }} (可选){{ end }}
{{- end }}
{{- else }}
本插件没有依赖其他插件。
{{- end }}

## 配置

以下是插件的配置示例：

` + "```yaml" + `
# {{ .Name }} 配置
id: {{ .ID }}
name: {{ .Name }}
version: {{ .Version }}
enabled: true

# 自定义配置
settings:
  # 在此处添加插件特定的配置项
  option1: value1
  option2: value2
` + "```" + `

## 使用方法

在此处添加插件的使用说明。

## API

在此处添加插件的API文档。
`

	// 解析模板
	g.templates["plugin_list"] = template.Must(template.New("plugin_list").Parse(pluginListTemplate))
	g.templates["plugin_detail"] = template.Must(template.New("plugin_detail").Parse(pluginDetailTemplate))
}

// LoadTemplate 加载模板
func (g *DocGenerator) LoadTemplate(name, content string) error {
	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		return fmt.Errorf("解析模板失败: %w", err)
	}
	
	g.templates[name] = tmpl
	return nil
}

// LoadTemplateFromFile 从文件加载模板
func (g *DocGenerator) LoadTemplateFromFile(name, path string) error {
	// 读取文件
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取模板文件失败: %w", err)
	}
	
	return g.LoadTemplate(name, string(content))
}

// GeneratePluginListDoc 生成插件列表文档
func (g *DocGenerator) GeneratePluginListDoc() (string, error) {
	// 获取所有插件
	plugins := g.registry.ListPlugins()
	
	// 排序插件
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].ID < plugins[j].ID
	})
	
	// 准备数据
	data := map[string]interface{}{
		"Plugins": plugins,
	}
	
	// 渲染模板
	var buf bytes.Buffer
	if err := g.templates["plugin_list"].Execute(&buf, data); err != nil {
		return "", fmt.Errorf("渲染模板失败: %w", err)
	}
	
	return buf.String(), nil
}

// GeneratePluginDetailDoc 生成插件详情文档
func (g *DocGenerator) GeneratePluginDetailDoc(pluginID string) (string, error) {
	// 获取插件元数据
	metadata, exists := g.registry.GetPluginMetadata(pluginID)
	if !exists {
		return "", fmt.Errorf("插件 %s 未注册", pluginID)
	}
	
	// 渲染模板
	var buf bytes.Buffer
	if err := g.templates["plugin_detail"].Execute(&buf, metadata); err != nil {
		return "", fmt.Errorf("渲染模板失败: %w", err)
	}
	
	return buf.String(), nil
}

// GenerateAllPluginDocs 生成所有插件文档
func (g *DocGenerator) GenerateAllPluginDocs(outputDir string) error {
	// 创建输出目录
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}
	
	// 生成插件列表文档
	listDoc, err := g.GeneratePluginListDoc()
	if err != nil {
		return fmt.Errorf("生成插件列表文档失败: %w", err)
	}
	
	// 写入插件列表文档
	listPath := filepath.Join(outputDir, "plugins.md")
	if err := os.WriteFile(listPath, []byte(listDoc), 0644); err != nil {
		return fmt.Errorf("写入插件列表文档失败: %w", err)
	}
	
	g.logger.Info("生成插件列表文档", "path", listPath)
	
	// 创建插件详情目录
	pluginsDir := filepath.Join(outputDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return fmt.Errorf("创建插件详情目录失败: %w", err)
	}
	
	// 获取所有插件
	plugins := g.registry.ListPlugins()
	
	// 生成插件详情文档
	for _, metadata := range plugins {
		// 生成插件详情文档
		detailDoc, err := g.GeneratePluginDetailDoc(metadata.ID)
		if err != nil {
			g.logger.Error("生成插件详情文档失败", "id", metadata.ID, "error", err)
			continue
		}
		
		// 写入插件详情文档
		detailPath := filepath.Join(pluginsDir, fmt.Sprintf("%s.md", metadata.ID))
		if err := os.WriteFile(detailPath, []byte(detailDoc), 0644); err != nil {
			g.logger.Error("写入插件详情文档失败", "id", metadata.ID, "error", err)
			continue
		}
		
		g.logger.Info("生成插件详情文档", "id", metadata.ID, "path", detailPath)
	}
	
	// 生成索引文档
	indexDoc := g.generateIndexDoc(plugins)
	indexPath := filepath.Join(outputDir, "README.md")
	if err := os.WriteFile(indexPath, []byte(indexDoc), 0644); err != nil {
		return fmt.Errorf("写入索引文档失败: %w", err)
	}
	
	g.logger.Info("生成索引文档", "path", indexPath)
	
	return nil
}

// generateIndexDoc 生成索引文档
func (g *DocGenerator) generateIndexDoc(plugins []api.PluginMetadata) string {
	var buf bytes.Buffer
	
	buf.WriteString("# 插件文档\n\n")
	buf.WriteString("本文档提供了系统中所有插件的详细信息。\n\n")
	
	buf.WriteString("## 插件列表\n\n")
	buf.WriteString("以下是系统中所有可用的插件：\n\n")
	
	// 按类别分组插件
	categories := make(map[string][]api.PluginMetadata)
	for _, plugin := range plugins {
		// 获取插件类别
		category := "其他"
		if len(plugin.Tags) > 0 {
			category = plugin.Tags[0]
		}
		
		categories[category] = append(categories[category], plugin)
	}
	
	// 获取所有类别
	var categoryNames []string
	for category := range categories {
		categoryNames = append(categoryNames, category)
	}
	
	// 排序类别
	sort.Strings(categoryNames)
	
	// 生成类别列表
	for _, category := range categoryNames {
		buf.WriteString(fmt.Sprintf("### %s\n\n", strings.Title(category)))
		
		// 排序插件
		plugins := categories[category]
		sort.Slice(plugins, func(i, j int) bool {
			return plugins[i].Name < plugins[j].Name
		})
		
		// 生成插件列表
		for _, plugin := range plugins {
			buf.WriteString(fmt.Sprintf("- [%s](plugins/%s.md) - %s\n", plugin.Name, plugin.ID, plugin.Description))
		}
		
		buf.WriteString("\n")
	}
	
	buf.WriteString("## 文档\n\n")
	buf.WriteString("- [插件列表](plugins.md) - 所有插件的列表和依赖关系\n")
	
	return buf.String()
}

// GeneratePluginDiagram 生成插件依赖关系图
func (g *DocGenerator) GeneratePluginDiagram(outputPath string) error {
	// 获取所有插件
	plugins := g.registry.ListPlugins()
	
	// 生成DOT格式的图形描述
	var buf bytes.Buffer
	buf.WriteString("digraph PluginDependencies {\n")
	buf.WriteString("  rankdir=LR;\n")
	buf.WriteString("  node [shape=box, style=filled, fillcolor=lightblue];\n\n")
	
	// 添加节点
	for _, plugin := range plugins {
		buf.WriteString(fmt.Sprintf("  \"%s\" [label=\"%s\\n(%s)\"];\n", plugin.ID, plugin.Name, plugin.Version))
	}
	
	buf.WriteString("\n")
	
	// 添加边
	for _, plugin := range plugins {
		for _, dep := range plugin.Dependencies {
			// 检查依赖是否存在
			_, exists := g.registry.GetPluginMetadata(dep.ID)
			if !exists {
				continue
			}
			
			// 添加依赖边
			if dep.Optional {
				buf.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\" [style=dashed, label=\"可选\"];\n", plugin.ID, dep.ID))
			} else {
				buf.WriteString(fmt.Sprintf("  \"%s\" -> \"%s\";\n", plugin.ID, dep.ID))
			}
		}
	}
	
	buf.WriteString("}\n")
	
	// 写入文件
	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("写入依赖关系图失败: %w", err)
	}
	
	g.logger.Info("生成插件依赖关系图", "path", outputPath)
	return nil
}
