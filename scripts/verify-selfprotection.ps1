# Kennel自我防护功能验证脚本
# 版本: v1.0
# 用途: 验证自我防护功能的完整性和正确性

param(
    [switch]$Verbose,           # 详细输出
    [switch]$SkipBuild,         # 跳过构建步骤
    [switch]$SkipTests,         # 跳过测试步骤
    [string]$OutputDir = "bin"  # 输出目录
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
    Write-ColorOutput "    Kennel自我防护功能验证工具 v1.0" "Cyan"
    Write-ColorOutput "    验证自我防护功能的完整性和正确性" "Cyan"
    Write-ColorOutput "================================================" "Cyan"
    Write-Host ""
}

# 检查环境
function Test-Environment {
    Write-ColorOutput "1. 检查构建环境..." "Green"

    # 检查Go环境
    try {
        $goVersion = go version
        Write-Host "  ✓ Go版本: $goVersion"
    }
    catch {
        Write-ColorOutput "  ✗ 错误: 未找到Go环境" "Red"
        exit 1
    }

    # 检查项目结构
    $requiredFiles = @(
        "go.mod",
        "pkg/core/selfprotect/types.go",
        "pkg/core/selfprotect/protection.go",
        "cmd/selfprotect-test/main.go"
    )

    foreach ($file in $requiredFiles) {
        if (Test-Path $file) {
            Write-Host "  ✓ 文件存在: $file"
        } else {
            Write-ColorOutput "  ✗ 文件缺失: $file" "Red"
            exit 1
        }
    }

    Write-ColorOutput "  环境检查完成" "Green"
    Write-Host ""
}

# 验证代码编译
function Test-Compilation {
    Write-ColorOutput "2. 验证代码编译..." "Green"

    # 测试不启用自我防护的编译
    Write-Host "  测试标准编译（不启用自我防护）..."
    try {
        $env:GOOS = "windows"
        $env:GOARCH = "amd64"
        go build -o "$OutputDir/agent_standard.exe" cmd/agent/main.go
        if (Test-Path "$OutputDir/agent_standard.exe") {
            Write-Host "  ✓ 标准编译成功"
        } else {
            throw "编译失败，未生成可执行文件"
        }
    }
    catch {
        Write-ColorOutput "  X 标准编译失败: $_" "Red"
        return $false
    }

    # 测试启用自我防护的编译
    Write-Host "  测试自我防护编译（启用selfprotect标签）..."
    try {
        go build -tags="selfprotect" -o "$OutputDir/agent_protected.exe" cmd/agent/main.go
        if (Test-Path "$OutputDir/agent_protected.exe") {
            Write-Host "  ✓ 自我防护编译成功"
        } else {
            throw "编译失败，未生成可执行文件"
        }
    }
    catch {
        Write-ColorOutput "  ✗ 自我防护编译失败: $_" "Red"
        return $false
    }

    # 测试测试工具编译
    Write-Host "  测试自我防护测试工具编译..."
    try {
        go build -tags="selfprotect" -o "$OutputDir/selfprotect-test.exe" cmd/selfprotect-test/main.go
        if (Test-Path "$OutputDir/selfprotect-test.exe") {
            Write-Host "  ✓ 测试工具编译成功"
        } else {
            throw "编译失败，未生成可执行文件"
        }
    }
    catch {
        Write-ColorOutput "  ✗ 测试工具编译失败: $_" "Red"
        return $false
    }

    # 测试集成示例编译
    Write-Host "  测试集成示例编译..."
    try {
        go build -tags="selfprotect" -o "$OutputDir/integration-example.exe" examples/selfprotect-integration/main.go
        if (Test-Path "$OutputDir/integration-example.exe") {
            Write-Host "  ✓ 集成示例编译成功"
        } else {
            throw "编译失败，未生成可执行文件"
        }
    }
    catch {
        Write-ColorOutput "  ✗ 集成示例编译失败: $_" "Red"
        return $false
    }

    Write-ColorOutput "  代码编译验证完成" "Green"
    Write-Host ""
    return $true
}

# 验证包导入
function Test-PackageImports {
    Write-ColorOutput "3. 验证包导入..." "Green"

    # 测试包导入
    Write-Host "  测试自我防护包导入..."
    try {
        $testCode = @"
package main

import (
    "fmt"
    "github.com/lomehong/kennel/pkg/core/selfprotect"
    "github.com/hashicorp/go-hclog"
)

func main() {
    logger := hclog.Default()
    config := selfprotect.DefaultProtectionConfig()
    manager := selfprotect.NewProtectionManager(config, logger)
    fmt.Printf("防护管理器创建成功: %v\n", manager != nil)
}
"@

        $testFile = "test_import.go"
        $testCode | Out-File -FilePath $testFile -Encoding UTF8

        go run -tags="selfprotect" $testFile
        Remove-Item $testFile -Force

        Write-Host "  ✓ 包导入测试成功"
    }
    catch {
        Write-ColorOutput "  ✗ 包导入测试失败: $_" "Red"
        if (Test-Path $testFile) {
            Remove-Item $testFile -Force
        }
        return $false
    }

    Write-ColorOutput "  包导入验证完成" "Green"
    Write-Host ""
    return $true
}

# 运行功能测试
function Test-Functionality {
    Write-ColorOutput "4. 运行功能测试..." "Green"

    if ($SkipTests) {
        Write-ColorOutput "  跳过功能测试（--SkipTests）" "Yellow"
        return $true
    }

    # 运行自我防护测试工具
    Write-Host "  运行自我防护测试工具..."
    try {
        if (Test-Path "$OutputDir/selfprotect-test.exe") {
            $testResult = & "$OutputDir/selfprotect-test.exe" -verbose
            if ($LASTEXITCODE -eq 0) {
                Write-Host "  ✓ 自我防护功能测试通过"
                if ($Verbose) {
                    Write-Host "  测试输出:"
                    $testResult | ForEach-Object { Write-Host "    $_" }
                }
            } else {
                throw "测试失败，退出代码: $LASTEXITCODE"
            }
        } else {
            throw "测试工具不存在: $OutputDir/selfprotect-test.exe"
        }
    }
    catch {
        Write-ColorOutput "  ✗ 自我防护功能测试失败: $_" "Red"
        return $false
    }

    # 运行Go单元测试
    Write-Host "  运行Go单元测试..."
    try {
        $testCmd = "go test -tags=`"selfprotect`" ./pkg/core/selfprotect/..."
        if ($Verbose) {
            $testCmd += " -v"
        }

        Invoke-Expression $testCmd
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  ✓ Go单元测试通过"
        } else {
            throw "单元测试失败，退出代码: $LASTEXITCODE"
        }
    }
    catch {
        Write-ColorOutput "  ✗ Go单元测试失败: $_" "Red"
        return $false
    }

    Write-ColorOutput "  功能测试验证完成" "Green"
    Write-Host ""
    return $true
}

# 验证配置文件
function Test-Configuration {
    Write-ColorOutput "5. 验证配置文件..." "Green"

    # 创建测试配置文件
    $testConfig = @"
# 测试配置文件
self_protection:
  enabled: true
  level: "basic"
  emergency_disable: ".emergency_disable"
  check_interval: "5s"
  restart_delay: "3s"
  max_restart_attempts: 3

  whitelist:
    enabled: true
    processes: ["taskmgr.exe"]
    users: ["SYSTEM"]

  process_protection:
    enabled: true
    protected_processes: ["agent.exe"]
    monitor_children: true
    prevent_debug: true
    prevent_dump: true

  file_protection:
    enabled: true
    protected_files: ["config.yaml"]
    protected_dirs: ["app"]
    check_integrity: true
    backup_enabled: true
    backup_dir: "backup"

  registry_protection:
    enabled: true
    protected_keys:
      - "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Services\\KennelAgent"
    monitor_changes: true

  service_protection:
    enabled: true
    service_name: "KennelAgent"
    auto_restart: true
    prevent_disable: true
"@

    $configFile = "test_config.yaml"
    try {
        $testConfig | Out-File -FilePath $configFile -Encoding UTF8

        # 测试配置加载
        $testCode = @"
package main

import (
    "fmt"
    "io/ioutil"
    "github.com/lomehong/kennel/pkg/core/selfprotect"
)

func main() {
    data, err := ioutil.ReadFile("$configFile")
    if err != nil {
        panic(err)
    }

    config, err := selfprotect.LoadProtectionConfigFromYAML(data)
    if err != nil {
        panic(err)
    }

    err = selfprotect.ValidateProtectionConfig(config)
    if err != nil {
        panic(err)
    }

    fmt.Println("配置文件验证成功")
}
"@

        $testFile = "test_config.go"
        $testCode | Out-File -FilePath $testFile -Encoding UTF8

        go run -tags="selfprotect" $testFile

        Write-Host "  ✓ 配置文件验证成功"

        # 清理测试文件
        Remove-Item $configFile -Force
        Remove-Item $testFile -Force
    }
    catch {
        Write-ColorOutput "  ✗ 配置文件验证失败: $_" "Red"

        # 清理测试文件
        if (Test-Path $configFile) { Remove-Item $configFile -Force }
        if (Test-Path $testFile) { Remove-Item $testFile -Force }

        return $false
    }

    Write-ColorOutput "  配置文件验证完成" "Green"
    Write-Host ""
    return $true
}

# 验证文档完整性
function Test-Documentation {
    Write-ColorOutput "6. 验证文档完整性..." "Green"

    $requiredDocs = @(
        "README-SelfProtection.md",
        "docs/self-protection-implementation-report.md",
        "docs/self-protection-usage-guide.md",
        "docs/self-protection-summary.md"
    )

    $allDocsExist = $true
    foreach ($doc in $requiredDocs) {
        if (Test-Path $doc) {
            Write-Host "  ✓ 文档存在: $doc"
        } else {
            Write-ColorOutput "  ✗ 文档缺失: $doc" "Red"
            $allDocsExist = $false
        }
    }

    if ($allDocsExist) {
        Write-ColorOutput "  文档完整性验证完成" "Green"
    } else {
        Write-ColorOutput "  文档完整性验证失败" "Red"
        return $false
    }

    Write-Host ""
    return $true
}

# 生成验证报告
function New-VerificationReport {
    param(
        [bool]$CompilationResult,
        [bool]$ImportResult,
        [bool]$FunctionalityResult,
        [bool]$ConfigurationResult,
        [bool]$DocumentationResult
    )

    Write-ColorOutput "7. 生成验证报告..." "Green"

    $report = @"
# Kennel自我防护功能验证报告

## 验证时间
$(Get-Date -Format "yyyy-MM-dd HH:mm:ss")

## 验证结果

### 1. 代码编译验证
状态: $(if ($CompilationResult) { "✓ 通过" } else { "✗ 失败" })
- 标准编译: $(if (Test-Path "$OutputDir/agent_standard.exe") { "✓" } else { "✗" })
- 自我防护编译: $(if (Test-Path "$OutputDir/agent_protected.exe") { "✓" } else { "✗" })
- 测试工具编译: $(if (Test-Path "$OutputDir/selfprotect-test.exe") { "✓" } else { "✗" })
- 集成示例编译: $(if (Test-Path "$OutputDir/integration-example.exe") { "✓" } else { "✗" })

### 2. 包导入验证
状态: $(if ($ImportResult) { "✓ 通过" } else { "✗ 失败" })

### 3. 功能测试验证
状态: $(if ($FunctionalityResult) { "✓ 通过" } else { "✗ 失败" })

### 4. 配置文件验证
状态: $(if ($ConfigurationResult) { "✓ 通过" } else { "✗ 失败" })

### 5. 文档完整性验证
状态: $(if ($DocumentationResult) { "✓ 通过" } else { "✗ 失败" })

## 总体结果
$(if ($CompilationResult -and $ImportResult -and $FunctionalityResult -and $ConfigurationResult -and $DocumentationResult) { "🎉 所有验证项目通过！自我防护功能实施完整且正确。" } else { "⚠️ 部分验证项目失败，请检查上述详细信息。" })

## 生成的文件
$(if (Test-Path "$OutputDir") { (Get-ChildItem $OutputDir -Name) -join "`n" } else { "无" })
"@

    $reportFile = "verification-report.md"
    $report | Out-File -FilePath $reportFile -Encoding UTF8

    Write-Host "  ✓ 验证报告已生成: $reportFile"
    Write-ColorOutput "  验证报告生成完成" "Green"
    Write-Host ""
}

# 显示最终结果
function Show-FinalResult {
    param(
        [bool]$AllPassed
    )

    Write-Host ""
    Write-ColorOutput "================================================" "Cyan"

    if ($AllPassed) {
        Write-ColorOutput "    🎉 验证完成！所有测试通过！" "Green"
        Write-ColorOutput "    Kennel自我防护功能实施完整且正确" "Green"
    } else {
        Write-ColorOutput "    ⚠️ 验证失败！部分测试未通过" "Red"
        Write-ColorOutput "    请检查上述错误信息并修复问题" "Red"
    }

    Write-ColorOutput "================================================" "Cyan"
    Write-Host ""

    Write-ColorOutput "生成的文件:" "Yellow"
    if (Test-Path $OutputDir) {
        Get-ChildItem $OutputDir | ForEach-Object {
            $size = if ($_.Length -gt 1MB) {
                "{0:F1} MB" -f ($_.Length / 1MB)
            } else {
                "{0:F0} KB" -f ($_.Length / 1KB)
            }
            Write-Host "  $($_.Name) ($size)"
        }
    }

    Write-Host ""
    Write-ColorOutput "查看详细报告: verification-report.md" "Cyan"
}

# 主函数
function Main {
    Show-Banner

    # 创建输出目录
    if (-not (Test-Path $OutputDir)) {
        New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
    }

    # 执行验证步骤
    $results = @{}

    try {
        Test-Environment

        if (-not $SkipBuild) {
            $results.Compilation = Test-Compilation
        } else {
            Write-ColorOutput "跳过构建步骤（--SkipBuild）" "Yellow"
            $results.Compilation = $true
        }

        $results.Import = Test-PackageImports
        $results.Functionality = Test-Functionality
        $results.Configuration = Test-Configuration
        $results.Documentation = Test-Documentation

        # 生成验证报告
        New-VerificationReport -CompilationResult $results.Compilation -ImportResult $results.Import -FunctionalityResult $results.Functionality -ConfigurationResult $results.Configuration -DocumentationResult $results.Documentation

        # 检查所有结果
        $allPassed = $results.Values | ForEach-Object { $_ } | Where-Object { $_ -eq $false } | Measure-Object | Select-Object -ExpandProperty Count
        $allPassed = $allPassed -eq 0

        Show-FinalResult -AllPassed $allPassed

        if (-not $allPassed) {
            exit 1
        }

    }
    catch {
        Write-ColorOutput "验证过程中发生错误: $_" "Red"
        exit 1
    }
}

# 执行主函数
Main
