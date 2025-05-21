# kennel 构建脚本
# 用于构建主程序、插件模块和Web前端

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

# 定义复制插件函数
function Copy-PluginToAppDir {
    param(
        [string]$PluginName
    )

    $sourcePath = "bin\$PluginName.exe"
    $targetDir = "app\$PluginName"
    $targetPath = "$targetDir\$PluginName.exe"

    # 确保目标目录存在
    if (-not (Test-Path $targetDir)) {
        New-Item -ItemType Directory -Path $targetDir -Force | Out-Null
        Write-InfoLog "创建插件目录: $targetDir"
    }

    # 复制插件可执行文件
    if (Test-Path $sourcePath) {
        Copy-Item -Path $sourcePath -Destination $targetPath -Force
        Write-SuccessLog "复制插件: $sourcePath -> $targetPath"
        return $true
    } else {
        Write-WarningLog "插件可执行文件不存在: $sourcePath"
        return $false
    }
}

Write-SuccessLog "开始构建 kennel..."

# 创建输出目录
Write-InfoLog "创建输出目录..."
New-Item -ItemType Directory -Path bin -Force | Out-Null
New-Item -ItemType Directory -Path bin\app\assets -Force | Out-Null
New-Item -ItemType Directory -Path bin\app\device -Force | Out-Null
New-Item -ItemType Directory -Path bin\app\dlp -Force | Out-Null
New-Item -ItemType Directory -Path bin\app\control -Force | Out-Null
New-Item -ItemType Directory -Path bin\app\audit -Force | Out-Null
New-Item -ItemType Directory -Path bin\web -Force | Out-Null

# 获取Go依赖
Write-InfoLog "获取Go依赖..."
Write-InfoLog "确保必要的依赖已安装..."

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
            Write-InfoLog "尝试继续构建过程..."
        }
    }
}

# 恢复原始GOPROXY环境变量
$env:GOPROXY = $originalGoproxy

# 构建主程序
Write-InfoLog "构建主程序..."
try {
    # 先检查是否有编译错误
    Write-InfoLog "检查编译错误..."
    $buildOutput = go build -o bin\agent.exe cmd\agent\main.go 2>&1

    if ($LASTEXITCODE -ne 0) {
        Write-ErrorLog "构建主程序失败，错误输出:"
        Write-Host $buildOutput -ForegroundColor Red

        # 尝试诊断问题
        Write-InfoLog "尝试诊断问题..."

        # 检查是否有缺少的依赖
        if ($buildOutput -match "no required module provides package") {
            $missingPackages = [regex]::Matches($buildOutput, "no required module provides package ([^;]+)")
            foreach ($package in $missingPackages) {
                $packageName = $package.Groups[1].Value.Trim()
                Write-WarningLog "缺少依赖包: $packageName"
                Write-InfoLog "尝试安装缺少的依赖: $packageName"
                go get -v $packageName
            }

            # 重新尝试构建
            Write-InfoLog "重新尝试构建主程序..."
            go build -o bin\agent.exe cmd\agent\main.go
            if ($LASTEXITCODE -ne 0) {
                throw "重新构建主程序失败"
            }
        } else {
            throw "构建主程序失败"
        }
    }

    Write-SuccessLog "主程序构建成功"
} catch {
    Write-ErrorLog "构建主程序失败: $_"
    Write-ErrorLog "请检查代码中的错误，或者尝试手动运行 'go build cmd/agent/main.go' 查看详细错误信息"
    exit 1
}

# 构建插件
function Build-Plugin {
    param (
        [string]$Name,
        [string]$SourceDir,
        [string]$OutputPath,
        [string]$MainFile = ""
    )

    Write-InfoLog "构建${Name}插件..."
    try {
        # 检查源目录是否存在
        if (-not (Test-Path $SourceDir)) {
            throw "插件源目录不存在: $SourceDir"
        }

        # 切换到插件目录，然后构建
        Push-Location $SourceDir

        # 创建输出目录（如果不存在）
        $outputDir = Split-Path -Parent (Join-Path $PWD.Path $OutputPath)
        if (-not (Test-Path $outputDir)) {
            New-Item -ItemType Directory -Path $outputDir -Force | Out-Null
        }

        # 如果指定了主文件，则使用它进行构建
        if ($MainFile -ne "") {
            $buildOutput = go build -o (Join-Path $PWD.Path $OutputPath) $MainFile 2>&1
        } else {
            $buildOutput = go build -o (Join-Path $PWD.Path $OutputPath) 2>&1
        }

        if ($LASTEXITCODE -ne 0) {
            Write-ErrorLog "构建${Name}插件失败，错误输出:"
            Write-Host $buildOutput -ForegroundColor Red

            # 尝试诊断问题
            Write-InfoLog "尝试诊断${Name}插件构建问题..."

            # 检查是否有缺少的依赖
            if ($buildOutput -match "no required module provides package") {
                $missingPackages = [regex]::Matches($buildOutput, "no required module provides package ([^;]+)")
                foreach ($package in $missingPackages) {
                    $packageName = $package.Groups[1].Value.Trim()
                    Write-WarningLog "缺少依赖包: $packageName"
                    Write-InfoLog "尝试安装缺少的依赖: $packageName"
                    go get -v $packageName
                }

                # 重新尝试构建
                Write-InfoLog "重新尝试构建${Name}插件..."
                if ($MainFile -ne "") {
                    go build -o (Join-Path $PWD.Path $OutputPath) $MainFile
                } else {
                    go build -o (Join-Path $PWD.Path $OutputPath)
                }

                if ($LASTEXITCODE -ne 0) {
                    Pop-Location
                    throw "重新构建${Name}插件失败"
                }
            } else {
                Pop-Location
                throw "构建${Name}插件失败"
            }
        }

        Pop-Location
        Write-SuccessLog "${Name}插件构建成功"
    } catch {
        if ((Get-Location).Path -ne $PWD.Path) {
            Pop-Location
        }
        Write-WarningLog "构建${Name}插件失败: $_"
        Write-WarningLog "请检查插件代码中的错误，或者尝试手动进入目录 '$SourceDir' 运行 'go build' 查看详细错误信息"
        # 插件构建失败不中断整个构建过程
    }
}

# 构建各个插件
Build-Plugin -Name "资产管理" -SourceDir "app\assets" -OutputPath "..\..\bin\app\assets\assets.exe"
Build-Plugin -Name "设备管理" -SourceDir "app\device" -OutputPath "..\..\bin\app\device\device.exe"
Build-Plugin -Name "数据防泄漏" -SourceDir "app\dlp" -OutputPath "..\..\bin\app\dlp\dlp.exe"
Build-Plugin -Name "终端管控" -SourceDir "app\control\cmd\control" -OutputPath "..\..\bin\app\control\control.exe"
Build-Plugin -Name "安全审计" -SourceDir "app\audit" -OutputPath "..\..\bin\app\audit\audit.exe"

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

        # 复制构建结果到bin目录
        Write-InfoLog "复制Web前端构建结果到bin目录..."
        Copy-Item -Path dist -Destination ..\bin\web -Recurse -Force

        # 返回上级目录
        Pop-Location

        Write-SuccessLog "Web前端构建成功"
    }
} catch {
    if ((Get-Location).Path -ne $PWD.Path) {
        Pop-Location
    }
    Write-ErrorLog "构建Web前端失败: $_"
    # Web前端构建失败不中断整个构建过程
}

# 复制配置文件
Write-InfoLog "复制配置文件..."
Copy-Item -Path config.yaml -Destination bin\config.yaml

# 复制插件可执行文件到app目录
# Write-InfoLog "复制插件可执行文件到app目录..."
# Copy-PluginToAppDir -PluginName "assets"
# Copy-PluginToAppDir -PluginName "device"
# Copy-PluginToAppDir -PluginName "dlp"
# Copy-PluginToAppDir -PluginName "control"
# Copy-PluginToAppDir -PluginName "audit"

Write-SuccessLog "构建完成！"
Write-Host ""
Write-InfoLog "可以通过以下命令运行程序："
Write-Host "cd bin" -ForegroundColor Yellow
Write-Host ".\agent.exe version" -ForegroundColor Yellow
Write-Host ".\agent.exe plugin list" -ForegroundColor Yellow
Write-Host ".\agent.exe start" -ForegroundColor Yellow
Write-Host ""
Write-WarningLog "注意：插件系统的gRPC接口实现尚未完成，插件加载可能会失败。"
