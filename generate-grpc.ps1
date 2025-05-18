# 生成gRPC代码的脚本

# 设置错误处理
$ErrorActionPreference = "Stop"

Write-Host "开始生成gRPC代码..." -ForegroundColor Green

# 检查protoc是否已安装
try {
    $protoc = Get-Command protoc -ErrorAction Stop
    Write-Host "找到protoc: $($protoc.Path)" -ForegroundColor Green
} catch {
    Write-Host "未找到protoc，请安装Protocol Buffers编译器" -ForegroundColor Red
    Write-Host "下载地址: https://github.com/protocolbuffers/protobuf/releases" -ForegroundColor Yellow
    exit 1
}

# 检查Go插件是否已安装
try {
    $protocGenGo = Get-Command protoc-gen-go -ErrorAction Stop
    Write-Host "找到protoc-gen-go: $($protocGenGo.Path)" -ForegroundColor Green
} catch {
    Write-Host "未找到protoc-gen-go，正在安装..." -ForegroundColor Yellow
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    if ($LASTEXITCODE -ne 0) {
        Write-Host "安装protoc-gen-go失败" -ForegroundColor Red
        exit 1
    }
    Write-Host "protoc-gen-go安装成功" -ForegroundColor Green
}

# 检查gRPC插件是否已安装
try {
    $protocGenGoGRPC = Get-Command protoc-gen-go-grpc -ErrorAction Stop
    Write-Host "找到protoc-gen-go-grpc: $($protocGenGoGRPC.Path)" -ForegroundColor Green
} catch {
    Write-Host "未找到protoc-gen-go-grpc，正在安装..." -ForegroundColor Yellow
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
    if ($LASTEXITCODE -ne 0) {
        Write-Host "安装protoc-gen-go-grpc失败" -ForegroundColor Red
        exit 1
    }
    Write-Host "protoc-gen-go-grpc安装成功" -ForegroundColor Green
}

# 创建输出目录
$outputDir = "pkg\plugin\proto\gen"
New-Item -ItemType Directory -Path $outputDir -Force | Out-Null
Write-Host "创建输出目录: $outputDir" -ForegroundColor Cyan

# 生成gRPC代码
$protoFile = "pkg\plugin\proto\module.proto"
Write-Host "正在生成gRPC代码: $protoFile" -ForegroundColor Cyan

# 执行protoc命令
try {
    # 生成Go代码
    protoc --go_out=. --go_opt=paths=source_relative $protoFile
    if ($LASTEXITCODE -ne 0) {
        throw "生成Go代码失败"
    }
    
    # 生成gRPC代码
    protoc --go-grpc_out=. --go-grpc_opt=paths=source_relative $protoFile
    if ($LASTEXITCODE -ne 0) {
        throw "生成gRPC代码失败"
    }
    
    Write-Host "gRPC代码生成成功" -ForegroundColor Green
} catch {
    Write-Host "生成gRPC代码失败: $_" -ForegroundColor Red
    exit 1
}

# 移动生成的文件到gen目录
try {
    $generatedFiles = Get-ChildItem -Path "pkg\plugin\proto" -Filter "*.pb.go"
    foreach ($file in $generatedFiles) {
        Move-Item -Path $file.FullName -Destination "$outputDir\$($file.Name)" -Force
        Write-Host "移动文件: $($file.Name) -> $outputDir\$($file.Name)" -ForegroundColor Cyan
    }
    Write-Host "文件移动成功" -ForegroundColor Green
} catch {
    Write-Host "移动文件失败: $_" -ForegroundColor Red
    exit 1
}

Write-Host "gRPC代码生成完成！" -ForegroundColor Green
