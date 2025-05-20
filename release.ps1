# 本地发布脚本

# 设置错误处理
$ErrorActionPreference = "Stop"

# 定义日志函数
function Write-InfoLog {
    param([string]$Message)
    Write-Host $Message -ForegroundColor Cyan
}

function Write-SuccessLog {
    param([string]$Message)
    Write-Host $Message -ForegroundColor Green
}

function Write-ErrorLog {
    param([string]$Message)
    Write-Host $Message -ForegroundColor Red
}

function Write-WarningLog {
    param([string]$Message)
    Write-Host $Message -ForegroundColor Yellow
}

Write-SuccessLog "开始本地发布..."

# 检查GoReleaser是否已安装
try {
    $goreleaser = Get-Command goreleaser -ErrorAction Stop
    Write-SuccessLog "找到goreleaser: $($goreleaser.Path)"
} catch {
    Write-ErrorLog "未找到goreleaser，请安装GoReleaser"
    Write-WarningLog "安装命令: go install github.com/goreleaser/goreleaser@latest"
    exit 1
}

# 清理dist目录
if (Test-Path dist) {
    Write-InfoLog "清理dist目录..."
    Remove-Item -Path dist -Recurse -Force
}

# 构建Web前端
Write-InfoLog "构建Web前端..."
try {
    # 检查Node.js是否安装
    $nodeVersion = node -v
    if ($LASTEXITCODE -ne 0) {
        Write-WarningLog "未找到Node.js，跳过Web前端构建"
    } else {
        Write-InfoLog "检测到Node.js版本: $nodeVersion"

        # 切换到web目录
        Push-Location web

        # 安装依赖
        Write-InfoLog "安装Web前端依赖..."
        Write-InfoLog "使用--legacy-peer-deps参数以解决依赖冲突问题..."
        npm install --legacy-peer-deps
        if ($LASTEXITCODE -ne 0) {
            Write-WarningLog "使用--legacy-peer-deps参数安装失败，尝试使用--force参数..."
            npm install --force
            if ($LASTEXITCODE -ne 0) {
                Pop-Location
                throw "安装Web前端依赖失败"
            }
        }

        # 构建前端
        Write-InfoLog "编译Web前端..."
        npm run build
        if ($LASTEXITCODE -ne 0) {
            Pop-Location
            throw "编译Web前端失败"
        }

        # 返回上级目录
        Pop-Location

        Write-SuccessLog "Web前端构建成功"
    }
} catch {
    if ((Get-Location).Path -ne $PWD.Path) {
        Pop-Location
    }
    Write-ErrorLog "构建Web前端失败: $_"
    # Web前端构建失败不中断整个发布过程
}

# 确保Go依赖已安装
Write-InfoLog "确保Go依赖已安装..."

# 设置GOPROXY环境变量以解决网络问题
$originalGoproxy = $env:GOPROXY
Write-InfoLog "设置GOPROXY环境变量以解决网络问题..."
$env:GOPROXY = "https://goproxy.cn,https://goproxy.io,direct"

# 添加重试机制
$maxRetries = 3
$retryCount = 0
$success = $false

while (-not $success -and $retryCount -lt $maxRetries) {
    try {
        Write-InfoLog "尝试获取依赖 (尝试 $($retryCount + 1)/$maxRetries)..."
        # 修复：移除 -timeout 参数，它不是 go get 命令的有效参数
        go get -v github.com/mitchellh/mapstructure github.com/Masterminds/semver/v3
        go get -v ./...
        $success = $true
        Write-SuccessLog "依赖获取成功"
    } catch {
        $retryCount++
        if ($retryCount -lt $maxRetries) {
            Write-WarningLog "获取依赖失败，将在5秒后重试: $_"
            Start-Sleep -Seconds 5
        } else {
            Write-ErrorLog "获取依赖失败，已达到最大重试次数: $_"
            Write-InfoLog "尝试继续发布过程..."
        }
    }
}

# 恢复原始GOPROXY环境变量
$env:GOPROXY = $originalGoproxy

# 运行GoReleaser
Write-InfoLog "运行GoReleaser..."
try {
    $goreleaseOutput = goreleaser release --snapshot --clean --skip=publish 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorLog "GoReleaser执行失败，错误输出:"
        Write-Host $goreleaseOutput -ForegroundColor Red

        # 尝试诊断问题
        if ($goreleaseOutput -match "no required module provides package") {
            Write-InfoLog "检测到缺少依赖，尝试安装..."
            $missingPackages = [regex]::Matches($goreleaseOutput, "no required module provides package ([^;]+)")
            foreach ($package in $missingPackages) {
                $packageName = $package.Groups[1].Value.Trim()
                Write-WarningLog "缺少依赖包: $packageName"
                Write-InfoLog "尝试安装缺少的依赖: $packageName"
                go get -v $packageName
            }

            # 重新尝试运行GoReleaser
            Write-InfoLog "重新尝试运行GoReleaser..."
            goreleaser release --snapshot --clean --skip=publish
            if ($LASTEXITCODE -ne 0) {
                throw "重新运行GoReleaser失败"
            }
        } else {
            throw "GoReleaser执行失败"
        }
    }
    Write-SuccessLog "GoReleaser执行成功"
} catch {
    Write-ErrorLog "GoReleaser执行失败: $_"
    Write-ErrorLog "请检查错误信息，或者尝试手动运行 'goreleaser release --snapshot --clean --skip=publish' 查看详细错误信息"
    # 继续执行后续步骤，尝试完成尽可能多的工作
}

# 后处理：移动插件到正确的目录
Write-InfoLog "移动插件到正确的目录..."

# 获取版本号（使用快照版本）
$version = "snapshot"

# 创建目录
New-Item -Path "dist/app/assets" -ItemType Directory -Force | Out-Null
New-Item -Path "dist/app/device" -ItemType Directory -Force | Out-Null
New-Item -Path "dist/app/dlp" -ItemType Directory -Force | Out-Null
New-Item -Path "dist/app/control" -ItemType Directory -Force | Out-Null
New-Item -Path "dist/app/audit" -ItemType Directory -Force | Out-Null
New-Item -Path "dist/web" -ItemType Directory -Force | Out-Null

# 处理Windows和macOS
foreach ($os in @("windows", "darwin")) {
    foreach ($arch in @("amd64", "arm64")) {
        $ext = if ($os -eq "windows") { ".exe" } else { "" }

        # 创建目标目录
        $targetDir = "dist/appframework_${version}_${os}_${arch}"

        # 资产管理插件
        $assetsBin = "dist/assets_${os}_${arch}${ext}"
        if (Test-Path $assetsBin) {
            New-Item -Path "${targetDir}/app/assets" -ItemType Directory -Force | Out-Null
            Copy-Item -Path $assetsBin -Destination "${targetDir}/app/assets/assets${ext}" -Force
        }

        # 设备管理插件
        $deviceBin = "dist/device_${os}_${arch}${ext}"
        if (Test-Path $deviceBin) {
            New-Item -Path "${targetDir}/app/device" -ItemType Directory -Force | Out-Null
            Copy-Item -Path $deviceBin -Destination "${targetDir}/app/device/device${ext}" -Force
        }

        # 数据防泄漏插件
        $dlpBin = "dist/dlp_${os}_${arch}${ext}"
        if (Test-Path $dlpBin) {
            New-Item -Path "${targetDir}/app/dlp" -ItemType Directory -Force | Out-Null
            Copy-Item -Path $dlpBin -Destination "${targetDir}/app/dlp/dlp${ext}" -Force
        }

        # 终端管控插件
        $controlBin = "dist/control_${os}_${arch}${ext}"
        if (Test-Path $controlBin) {
            New-Item -Path "${targetDir}/app/control" -ItemType Directory -Force | Out-Null
            Copy-Item -Path $controlBin -Destination "${targetDir}/app/control/control${ext}" -Force
        }

        # 安全审计插件
        $auditBin = "dist/audit_${os}_${arch}${ext}"
        if (Test-Path $auditBin) {
            New-Item -Path "${targetDir}/app/audit" -ItemType Directory -Force | Out-Null
            Copy-Item -Path $auditBin -Destination "${targetDir}/app/audit/audit${ext}" -Force
        }
    }
}

# 复制Web前端到发布目录
Write-InfoLog "复制Web前端到发布目录..."
if (Test-Path "web/dist") {
    # 复制到通用web目录
    Copy-Item -Path "web/dist" -Destination "dist/web" -Recurse -Force

    # 复制到各个平台特定目录
    foreach ($os in @("windows", "darwin")) {
        foreach ($arch in @("amd64", "arm64")) {
            $targetDir = "dist/appframework_${version}_${os}_${arch}"
            if (Test-Path $targetDir) {
                New-Item -Path "${targetDir}/web" -ItemType Directory -Force | Out-Null
                Copy-Item -Path "web/dist" -Destination "${targetDir}/web" -Recurse -Force

                # 复制配置文件
                Copy-Item -Path "config.yaml" -Destination "${targetDir}/config.yaml" -Force
            }
        }
    }

    Write-SuccessLog "Web前端复制完成"
} else {
    Write-WarningLog "未找到Web前端构建结果，跳过复制"
}

Write-SuccessLog "本地发布完成！"
Write-InfoLog "发布文件位于dist目录"
