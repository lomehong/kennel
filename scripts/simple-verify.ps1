# Kennel自我防护功能简单验证脚本
param(
    [string]$OutputDir = "bin"
)

$ErrorActionPreference = "Stop"

function Write-Status {
    param([string]$Message, [string]$Status = "INFO")
    $timestamp = Get-Date -Format "HH:mm:ss"
    switch ($Status) {
        "OK" { Write-Host "[$timestamp] [OK] $Message" -ForegroundColor Green }
        "ERROR" { Write-Host "[$timestamp] [ERROR] $Message" -ForegroundColor Red }
        "WARN" { Write-Host "[$timestamp] [WARN] $Message" -ForegroundColor Yellow }
        default { Write-Host "[$timestamp] [INFO] $Message" -ForegroundColor White }
    }
}

Write-Host "================================================"
Write-Host "    Kennel自我防护功能验证工具"
Write-Host "================================================"
Write-Host ""

# 创建输出目录
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
    Write-Status "创建输出目录: $OutputDir" "OK"
}

$allPassed = $true

# 1. 检查Go环境
Write-Status "检查Go环境..."
try {
    $goVersion = go version
    Write-Status "Go版本: $goVersion" "OK"
}
catch {
    Write-Status "未找到Go环境" "ERROR"
    $allPassed = $false
}

# 2. 检查必要文件
Write-Status "检查项目文件..."
$requiredFiles = @(
    "go.mod",
    "pkg/core/selfprotect/types.go",
    "pkg/core/selfprotect/protection.go",
    "cmd/selfprotect-test/main.go"
)

foreach ($file in $requiredFiles) {
    if (Test-Path $file) {
        Write-Status "文件存在: $file" "OK"
    } else {
        Write-Status "文件缺失: $file" "ERROR"
        $allPassed = $false
    }
}

# 3. 测试编译
Write-Status "测试编译..."

# 标准编译
Write-Status "编译标准版本..."
try {
    go build -o "$OutputDir/agent_standard.exe" cmd/agent/main.go
    if (Test-Path "$OutputDir/agent_standard.exe") {
        Write-Status "标准编译成功" "OK"
    } else {
        throw "未生成可执行文件"
    }
}
catch {
    Write-Status "标准编译失败: $_" "ERROR"
    $allPassed = $false
}

# 自我防护编译
Write-Status "编译自我防护版本..."
try {
    go build -tags="selfprotect" -o "$OutputDir/agent_protected.exe" cmd/agent/main.go
    if (Test-Path "$OutputDir/agent_protected.exe") {
        Write-Status "自我防护编译成功" "OK"
    } else {
        throw "未生成可执行文件"
    }
}
catch {
    Write-Status "自我防护编译失败: $_" "ERROR"
    $allPassed = $false
}

# 测试工具编译
Write-Status "编译测试工具..."
try {
    go build -tags="selfprotect" -o "$OutputDir/selfprotect-test.exe" cmd/selfprotect-test/main.go
    if (Test-Path "$OutputDir/selfprotect-test.exe") {
        Write-Status "测试工具编译成功" "OK"
    } else {
        throw "未生成可执行文件"
    }
}
catch {
    Write-Status "测试工具编译失败: $_" "ERROR"
    $allPassed = $false
}

# 4. 运行功能测试
Write-Status "运行功能测试..."
if (Test-Path "$OutputDir/selfprotect-test.exe") {
    try {
        $testResult = & "$OutputDir/selfprotect-test.exe"
        if ($LASTEXITCODE -eq 0) {
            Write-Status "功能测试通过" "OK"
        } else {
            throw "测试失败，退出代码: $LASTEXITCODE"
        }
    }
    catch {
        Write-Status "功能测试失败: $_" "ERROR"
        $allPassed = $false
    }
} else {
    Write-Status "测试工具不存在，跳过功能测试" "WARN"
}

# 5. 运行Go测试
Write-Status "运行Go单元测试..."
try {
    go test -tags="selfprotect" ./pkg/core/selfprotect/...
    if ($LASTEXITCODE -eq 0) {
        Write-Status "Go单元测试通过" "OK"
    } else {
        throw "单元测试失败，退出代码: $LASTEXITCODE"
    }
}
catch {
    Write-Status "Go单元测试失败: $_" "ERROR"
    $allPassed = $false
}

# 6. 检查文档
Write-Status "检查文档文件..."
$docs = @(
    "README-SelfProtection.md",
    "docs/self-protection-implementation-report.md",
    "docs/self-protection-usage-guide.md",
    "docs/self-protection-summary.md"
)

foreach ($doc in $docs) {
    if (Test-Path $doc) {
        Write-Status "文档存在: $doc" "OK"
    } else {
        Write-Status "文档缺失: $doc" "WARN"
    }
}

# 显示结果
Write-Host ""
Write-Host "================================================"
if ($allPassed) {
    Write-Status "所有验证通过！自我防护功能实施完整" "OK"
} else {
    Write-Status "部分验证失败，请检查上述错误" "ERROR"
}
Write-Host "================================================"

# 显示生成的文件
Write-Host ""
Write-Status "生成的文件:"
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

if (-not $allPassed) {
    exit 1
}
