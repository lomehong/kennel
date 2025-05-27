# Kennel项目构建脚本（支持自我防护）
# 版本: v2.0
# 支持自我防护功能的编译

param(
    [string]$Target = "all",           # 构建目标: all, agent, plugins, web, clean
    [string]$Platform = "windows",     # 目标平台: windows, linux, darwin
    [string]$Arch = "amd64",          # 目标架构: amd64, 386, arm64
    [switch]$EnableProtection,        # 启用自我防护功能
    [switch]$Release,                 # 发布模式
    [switch]$Verbose,                 # 详细输出
    [switch]$Clean,                   # 清理构建
    [string]$OutputDir = "bin"        # 输出目录
)

# 设置错误处理
$ErrorActionPreference = "Stop"

# 颜色输出函数
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    
    $originalColor = $Host.UI.RawUI.ForegroundColor
    $Host.UI.RawUI.ForegroundColor = $Color
    Write-Host $Message
    $Host.UI.RawUI.ForegroundColor = $originalColor
}

# 显示横幅
function Show-Banner {
    Write-ColorOutput "================================================" "Cyan"
    Write-ColorOutput "    Kennel终端安全管理系统构建工具 v2.0" "Cyan"
    Write-ColorOutput "    支持自我防护功能的高级构建" "Cyan"
    Write-ColorOutput "================================================" "Cyan"
    Write-Host ""
}

# 显示构建信息
function Show-BuildInfo {
    Write-ColorOutput "构建配置:" "Yellow"
    Write-Host "  目标: $Target"
    Write-Host "  平台: $Platform"
    Write-Host "  架构: $Arch"
    Write-Host "  自我防护: $(if ($EnableProtection) { '启用' } else { '禁用' })"
    Write-Host "  发布模式: $(if ($Release) { '是' } else { '否' })"
    Write-Host "  输出目录: $OutputDir"
    Write-Host ""
}

# 检查环境
function Test-Environment {
    Write-ColorOutput "检查构建环境..." "Green"
    
    # 检查Go环境
    try {
        $goVersion = go version
        Write-Host "  Go版本: $goVersion"
    }
    catch {
        Write-ColorOutput "错误: 未找到Go环境，请先安装Go" "Red"
        exit 1
    }
    
    # 检查Node.js环境（用于Web前端）
    try {
        $nodeVersion = node --version
        Write-Host "  Node.js版本: $nodeVersion"
    }
    catch {
        Write-ColorOutput "警告: 未找到Node.js环境，将跳过Web前端构建" "Yellow"
    }
    
    # 检查项目结构
    if (-not (Test-Path "go.mod")) {
        Write-ColorOutput "错误: 当前目录不是Go项目根目录" "Red"
        exit 1
    }
    
    Write-ColorOutput "环境检查完成" "Green"
    Write-Host ""
}

# 清理构建目录
function Clear-BuildDirectory {
    Write-ColorOutput "清理构建目录..." "Green"
    
    if (Test-Path $OutputDir) {
        Remove-Item -Recurse -Force $OutputDir
        Write-Host "  已清理: $OutputDir"
    }
    
    # 清理临时文件
    Get-ChildItem -Path . -Recurse -Name "*.exe" | Where-Object { $_ -like "*_temp*" } | Remove-Item -Force
    Get-ChildItem -Path . -Recurse -Name "*.log" | Where-Object { $_ -like "*build*" } | Remove-Item -Force
    
    Write-ColorOutput "清理完成" "Green"
    Write-Host ""
}

# 创建输出目录
function New-OutputDirectory {
    Write-ColorOutput "创建输出目录..." "Green"
    
    $directories = @(
        $OutputDir,
        "$OutputDir/app",
        "$OutputDir/app/assets",
        "$OutputDir/app/audit", 
        "$OutputDir/app/control",
        "$OutputDir/app/device",
        "$OutputDir/app/dlp",
        "$OutputDir/web",
        "$OutputDir/config",
        "$OutputDir/logs",
        "$OutputDir/backup"
    )
    
    foreach ($dir in $directories) {
        if (-not (Test-Path $dir)) {
            New-Item -ItemType Directory -Path $dir -Force | Out-Null
            Write-Host "  创建目录: $dir"
        }
    }
    
    Write-ColorOutput "目录创建完成" "Green"
    Write-Host ""
}

# 设置构建标签
function Get-BuildTags {
    $tags = @()
    
    if ($EnableProtection) {
        $tags += "selfprotect"
        Write-ColorOutput "启用自我防护功能" "Yellow"
    }
    
    if ($Release) {
        $tags += "release"
    }
    
    return $tags -join ","
}

# 设置构建参数
function Get-BuildFlags {
    $flags = @()
    
    if ($Release) {
        $flags += "-ldflags=-s -w"  # 去除调试信息，减小文件大小
    }
    
    $buildTags = Get-BuildTags
    if ($buildTags) {
        $flags += "-tags=`"$buildTags`""
    }
    
    return $flags
}

# 构建主程序
function Build-Agent {
    Write-ColorOutput "构建主程序 (agent.exe)..." "Green"
    
    $buildFlags = Get-BuildFlags
    $outputPath = "$OutputDir/agent.exe"
    
    $env:GOOS = $Platform
    $env:GOARCH = $Arch
    
    $buildCmd = "go build $($buildFlags -join ' ') -o `"$outputPath`" cmd/agent/main.go"
    
    if ($Verbose) {
        Write-Host "  执行命令: $buildCmd"
    }
    
    try {
        Invoke-Expression $buildCmd
        
        if (Test-Path $outputPath) {
            $fileInfo = Get-Item $outputPath
            Write-Host "  构建成功: $outputPath ($(($fileInfo.Length / 1MB).ToString('F2')) MB)"
        }
        else {
            throw "构建失败，未生成可执行文件"
        }
    }
    catch {
        Write-ColorOutput "主程序构建失败: $_" "Red"
        throw
    }
}

# 构建插件
function Build-Plugins {
    Write-ColorOutput "构建插件..." "Green"
    
    $plugins = @("assets", "audit", "control", "device", "dlp")
    $buildFlags = Get-BuildFlags
    
    $env:GOOS = $Platform
    $env:GOARCH = $Arch
    
    foreach ($plugin in $plugins) {
        Write-Host "  构建插件: $plugin"
        
        $sourcePath = "app/$plugin/main.go"
        $outputPath = "$OutputDir/app/$plugin/$plugin.exe"
        
        if (-not (Test-Path $sourcePath)) {
            Write-ColorOutput "    警告: 源文件不存在 $sourcePath" "Yellow"
            continue
        }
        
        $buildCmd = "go build $($buildFlags -join ' ') -o `"$outputPath`" `"$sourcePath`""
        
        if ($Verbose) {
            Write-Host "    执行命令: $buildCmd"
        }
        
        try {
            Invoke-Expression $buildCmd
            
            if (Test-Path $outputPath) {
                $fileInfo = Get-Item $outputPath
                Write-Host "    构建成功: $outputPath ($(($fileInfo.Length / 1MB).ToString('F2')) MB)"
            }
            else {
                Write-ColorOutput "    构建失败: $plugin" "Red"
            }
        }
        catch {
            Write-ColorOutput "    插件构建失败 ($plugin): $_" "Red"
        }
    }
}

# 构建Web前端
function Build-WebFrontend {
    Write-ColorOutput "构建Web前端..." "Green"
    
    $webDir = "web"
    $distDir = "$webDir/dist"
    $outputDir = "$OutputDir/web/dist"
    
    if (-not (Test-Path $webDir)) {
        Write-ColorOutput "  跳过Web前端构建（目录不存在）" "Yellow"
        return
    }
    
    # 检查Node.js
    try {
        node --version | Out-Null
    }
    catch {
        Write-ColorOutput "  跳过Web前端构建（Node.js未安装）" "Yellow"
        return
    }
    
    Push-Location $webDir
    
    try {
        # 安装依赖
        if (Test-Path "package.json") {
            Write-Host "  安装依赖..."
            npm install
        }
        
        # 构建
        Write-Host "  构建前端..."
        if ($Release) {
            npm run build:prod
        }
        else {
            npm run build
        }
        
        # 复制构建结果
        if (Test-Path $distDir) {
            Copy-Item -Recurse -Force "$distDir/*" $outputDir
            Write-Host "  Web前端构建成功"
        }
        else {
            Write-ColorOutput "  Web前端构建失败" "Red"
        }
    }
    catch {
        Write-ColorOutput "  Web前端构建失败: $_" "Red"
    }
    finally {
        Pop-Location
    }
}

# 复制配置文件
function Copy-ConfigFiles {
    Write-ColorOutput "复制配置文件..." "Green"
    
    $configFiles = @(
        @{ Source = "config.yaml"; Dest = "$OutputDir/config.yaml" },
        @{ Source = "app/assets/config.yaml"; Dest = "$OutputDir/app/assets/config.yaml" },
        @{ Source = "app/audit/config.yaml"; Dest = "$OutputDir/app/audit/config.yaml" },
        @{ Source = "app/control/config.yaml"; Dest = "$OutputDir/app/control/config.yaml" },
        @{ Source = "app/device/config.yaml"; Dest = "$OutputDir/app/device/config.yaml" },
        @{ Source = "app/dlp/config.yaml"; Dest = "$OutputDir/app/dlp/config.yaml" }
    )
    
    foreach ($config in $configFiles) {
        if (Test-Path $config.Source) {
            Copy-Item -Force $config.Source $config.Dest
            Write-Host "  复制: $($config.Source) -> $($config.Dest)"
        }
        else {
            Write-ColorOutput "  警告: 配置文件不存在 $($config.Source)" "Yellow"
        }
    }
    
    Write-ColorOutput "配置文件复制完成" "Green"
}

# 生成版本信息
function New-VersionInfo {
    Write-ColorOutput "生成版本信息..." "Green"
    
    $version = "1.0.0"
    $buildTime = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $gitCommit = ""
    
    try {
        $gitCommit = git rev-parse --short HEAD 2>$null
    }
    catch {
        $gitCommit = "unknown"
    }
    
    $versionInfo = @{
        Version = $version
        BuildTime = $buildTime
        GitCommit = $gitCommit
        Platform = $Platform
        Arch = $Arch
        ProtectionEnabled = $EnableProtection
        ReleaseMode = $Release
    }
    
    $versionJson = $versionInfo | ConvertTo-Json -Depth 2
    $versionPath = "$OutputDir/version.json"
    
    $versionJson | Out-File -FilePath $versionPath -Encoding UTF8
    Write-Host "  版本信息: $versionPath"
    
    Write-ColorOutput "版本信息生成完成" "Green"
}

# 运行测试
function Invoke-Tests {
    Write-ColorOutput "运行测试..." "Green"
    
    $buildTags = Get-BuildTags
    $testCmd = "go test ./..."
    
    if ($buildTags) {
        $testCmd += " -tags=`"$buildTags`""
    }
    
    if ($Verbose) {
        $testCmd += " -v"
    }
    
    Write-Host "  执行命令: $testCmd"
    
    try {
        Invoke-Expression $testCmd
        Write-ColorOutput "测试通过" "Green"
    }
    catch {
        Write-ColorOutput "测试失败: $_" "Red"
        throw
    }
}

# 显示构建结果
function Show-BuildResult {
    Write-ColorOutput "构建完成!" "Green"
    Write-Host ""
    
    Write-ColorOutput "构建结果:" "Yellow"
    
    if (Test-Path "$OutputDir/agent.exe") {
        $agentSize = (Get-Item "$OutputDir/agent.exe").Length
        Write-Host "  主程序: agent.exe ($(($agentSize / 1MB).ToString('F2')) MB)"
    }
    
    $plugins = @("assets", "audit", "control", "device", "dlp")
    foreach ($plugin in $plugins) {
        $pluginPath = "$OutputDir/app/$plugin/$plugin.exe"
        if (Test-Path $pluginPath) {
            $pluginSize = (Get-Item $pluginPath).Length
            Write-Host "  插件: $plugin.exe ($(($pluginSize / 1KB).ToString('F0')) KB)"
        }
    }
    
    if (Test-Path "$OutputDir/web/dist") {
        $webFiles = Get-ChildItem -Recurse "$OutputDir/web/dist" | Measure-Object -Property Length -Sum
        Write-Host "  Web前端: $(($webFiles.Sum / 1MB).ToString('F2')) MB"
    }
    
    Write-Host ""
    Write-ColorOutput "输出目录: $OutputDir" "Cyan"
    
    if ($EnableProtection) {
        Write-Host ""
        Write-ColorOutput "注意: 已启用自我防护功能" "Yellow"
        Write-ColorOutput "运行时需要管理员权限以获得完整的防护能力" "Yellow"
    }
}

# 主函数
function Main {
    Show-Banner
    Show-BuildInfo
    
    try {
        Test-Environment
        
        if ($Clean -or $Target -eq "clean") {
            Clear-BuildDirectory
            if ($Target -eq "clean") {
                return
            }
        }
        
        New-OutputDirectory
        
        switch ($Target) {
            "all" {
                Build-Agent
                Build-Plugins
                Build-WebFrontend
                Copy-ConfigFiles
                New-VersionInfo
            }
            "agent" {
                Build-Agent
                Copy-ConfigFiles
                New-VersionInfo
            }
            "plugins" {
                Build-Plugins
                Copy-ConfigFiles
            }
            "web" {
                Build-WebFrontend
            }
            "test" {
                Invoke-Tests
                return
            }
            default {
                Write-ColorOutput "未知的构建目标: $Target" "Red"
                exit 1
            }
        }
        
        Show-BuildResult
        
    }
    catch {
        Write-ColorOutput "构建失败: $_" "Red"
        exit 1
    }
}

# 执行主函数
Main
