# 生成gRPC代码的脚本

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

Write-SuccessLog "开始生成gRPC代码..."

# 检查protoc是否已安装
try {
    $protoc = Get-Command protoc -ErrorAction Stop
    Write-SuccessLog "找到protoc: $($protoc.Path)"
} catch {
    Write-ErrorLog "未找到protoc，请安装Protocol Buffers编译器"
    Write-WarningLog "下载地址: https://github.com/protocolbuffers/protobuf/releases"
    exit 1
}

# 检查Go插件是否已安装
try {
    $protocGenGo = Get-Command protoc-gen-go -ErrorAction Stop
    Write-SuccessLog "找到protoc-gen-go: $($protocGenGo.Path)"
} catch {
    Write-WarningLog "未找到protoc-gen-go，正在安装..."

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
            Write-InfoLog "尝试安装protoc-gen-go (尝试 $($retryCount + 1)/$maxRetries)..."
            # 修复：移除 -timeout 参数，它不是 go install 命令的有效参数
            go install -v google.golang.org/protobuf/cmd/protoc-gen-go@latest
            $success = $true
            Write-SuccessLog "protoc-gen-go安装成功"
        } catch {
            $retryCount++
            if ($retryCount -lt $maxRetries) {
                Write-WarningLog "安装失败，将在5秒后重试: $_"
                Start-Sleep -Seconds 5
            } else {
                Write-ErrorLog "安装protoc-gen-go失败，已达到最大重试次数: $_"
                $env:GOPROXY = $originalGoproxy
                exit 1
            }
        }
    }

    # 恢复原始GOPROXY环境变量
    $env:GOPROXY = $originalGoproxy
}

# 检查gRPC插件是否已安装
try {
    $protocGenGoGRPC = Get-Command protoc-gen-go-grpc -ErrorAction Stop
    Write-SuccessLog "找到protoc-gen-go-grpc: $($protocGenGoGRPC.Path)"
} catch {
    Write-WarningLog "未找到protoc-gen-go-grpc，正在安装..."

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
            Write-InfoLog "尝试安装protoc-gen-go-grpc (尝试 $($retryCount + 1)/$maxRetries)..."
            # 修复：移除 -timeout 参数，它不是 go install 命令的有效参数
            go install -v google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
            $success = $true
            Write-SuccessLog "protoc-gen-go-grpc安装成功"
        } catch {
            $retryCount++
            if ($retryCount -lt $maxRetries) {
                Write-WarningLog "安装失败，将在5秒后重试: $_"
                Start-Sleep -Seconds 5
            } else {
                Write-ErrorLog "安装protoc-gen-go-grpc失败，已达到最大重试次数: $_"
                $env:GOPROXY = $originalGoproxy
                exit 1
            }
        }
    }

    # 恢复原始GOPROXY环境变量
    $env:GOPROXY = $originalGoproxy
}

# 创建输出目录
$outputDir = "pkg\plugin\proto\gen"
New-Item -ItemType Directory -Path $outputDir -Force | Out-Null
Write-InfoLog "创建输出目录: $outputDir"

# 检查proto文件是否存在
$protoFile = "pkg\plugin\proto\module.proto"
if (-not (Test-Path $protoFile)) {
    Write-ErrorLog "Proto文件不存在: $protoFile"
    exit 1
}

Write-InfoLog "正在生成gRPC代码: $protoFile"

# 执行protoc命令
try {
    # 生成Go代码
    Write-InfoLog "生成Go代码..."
    $goOutput = protoc --go_out=. --go_opt=paths=source_relative $protoFile 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorLog "生成Go代码失败，错误输出:"
        Write-Host $goOutput -ForegroundColor Red
        throw "生成Go代码失败"
    }

    # 生成gRPC代码
    Write-InfoLog "生成gRPC代码..."
    $grpcOutput = protoc --go-grpc_out=. --go-grpc_opt=paths=source_relative $protoFile 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-ErrorLog "生成gRPC代码失败，错误输出:"
        Write-Host $grpcOutput -ForegroundColor Red
        throw "生成gRPC代码失败"
    }

    Write-SuccessLog "gRPC代码生成成功"
} catch {
    Write-ErrorLog "生成gRPC代码失败: $_"
    Write-ErrorLog "请检查proto文件是否正确，或者尝试手动运行protoc命令查看详细错误信息"
    exit 1
}

# 移动生成的文件到gen目录
try {
    $generatedFiles = Get-ChildItem -Path "pkg\plugin\proto" -Filter "*.pb.go"

    if ($generatedFiles.Count -eq 0) {
        Write-WarningLog "未找到生成的文件，请检查protoc命令是否正确执行"
    } else {
        Write-InfoLog "找到 $($generatedFiles.Count) 个生成的文件，正在移动到 $outputDir"

        foreach ($file in $generatedFiles) {
            # 检查目标文件是否存在
            $targetPath = "$outputDir\$($file.Name)"
            if (Test-Path $targetPath) {
                Write-InfoLog "目标文件已存在，将被覆盖: $targetPath"
            }

            # 移动文件
            Move-Item -Path $file.FullName -Destination $targetPath -Force
            Write-InfoLog "移动文件: $($file.Name) -> $targetPath"

            # 验证文件是否成功移动
            if (Test-Path $targetPath) {
                Write-InfoLog "文件成功移动到: $targetPath"
            } else {
                throw "文件移动失败: $($file.Name)"
            }
        }

        Write-SuccessLog "文件移动成功"
    }
} catch {
    Write-ErrorLog "移动文件失败: $_"
    Write-ErrorLog "请检查文件权限和目录结构"
    exit 1
}

# 检查生成的文件
$finalFiles = Get-ChildItem -Path $outputDir -Filter "*.pb.go"
if ($finalFiles.Count -gt 0) {
    Write-SuccessLog "gRPC代码生成完成！共生成 $($finalFiles.Count) 个文件:"
    foreach ($file in $finalFiles) {
        Write-InfoLog "- $($file.Name)"
    }
} else {
    Write-WarningLog "gRPC代码生成可能不完整，输出目录中没有找到.pb.go文件"
}
