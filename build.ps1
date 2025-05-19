# AppFramework 构建脚本
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

Write-SuccessLog "开始构建 AppFramework..."

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
go get -v ./...

# 构建主程序
Write-InfoLog "构建主程序..."
try {
    go build -o bin\agent.exe cmd\agent\main.go
    if ($LASTEXITCODE -ne 0) {
        throw "构建主程序失败"
    }
    Write-SuccessLog "主程序构建成功"
} catch {
    Write-ErrorLog "构建主程序失败: $_"
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
        # 切换到插件目录，然后构建
        Push-Location $SourceDir

        # 如果指定了主文件，则使用它进行构建
        if ($MainFile -ne "") {
            go build -o (Join-Path $PWD.Path $OutputPath) $MainFile
        } else {
            go build -o (Join-Path $PWD.Path $OutputPath)
        }

        if ($LASTEXITCODE -ne 0) {
            Pop-Location
            throw "构建${Name}插件失败"
        }
        Pop-Location
        Write-SuccessLog "${Name}插件构建成功"
    } catch {
        if ((Get-Location).Path -ne $PWD.Path) {
            Pop-Location
        }
        Write-WarningLog "构建${Name}插件失败: $_"
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

Write-SuccessLog "构建完成！"
Write-Host ""
Write-InfoLog "可以通过以下命令运行程序："
Write-Host "cd bin" -ForegroundColor Yellow
Write-Host ".\agent.exe version" -ForegroundColor Yellow
Write-Host ".\agent.exe plugin list" -ForegroundColor Yellow
Write-Host ".\agent.exe start" -ForegroundColor Yellow
Write-Host ""
Write-WarningLog "注意：插件系统的gRPC接口实现尚未完成，插件加载可能会失败。"
