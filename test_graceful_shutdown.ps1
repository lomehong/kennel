# 测试优雅终止功能的脚本

# 设置错误处理
$ErrorActionPreference = "Stop"

Write-Host "开始测试优雅终止功能..." -ForegroundColor Green

# 构建应用程序
Write-Host "构建应用程序..." -ForegroundColor Cyan
go build -o agent.exe .\cmd\agent\main.go

# 构建测试插件
Write-Host "构建测试插件..." -ForegroundColor Cyan
go build -o test_plugin.exe .\test\test_plugin\main.go

# 创建插件目录
if (-not (Test-Path "plugins\test")) {
    New-Item -Path "plugins\test" -ItemType Directory -Force | Out-Null
}

# 复制测试插件到插件目录
Copy-Item -Path "test_plugin.exe" -Destination "plugins\test\" -Force

# 启动应用程序
Write-Host "启动应用程序..." -ForegroundColor Cyan
$process = Start-Process -FilePath ".\agent.exe" -ArgumentList "start" -PassThru -NoNewWindow

# 等待应用程序启动
Write-Host "等待应用程序启动..." -ForegroundColor Cyan
Start-Sleep -Seconds 3

# 发送 SIGINT 信号（Ctrl+C）
Write-Host "发送 SIGINT 信号（模拟 Ctrl+C）..." -ForegroundColor Yellow
[Console]::TreatControlCAsInput = $true
$process.CloseMainWindow()

# 等待应用程序优雅终止
Write-Host "等待应用程序优雅终止..." -ForegroundColor Cyan
$process.WaitForExit(10000)  # 等待最多10秒

# 检查应用程序是否已终止
if ($process.HasExited) {
    Write-Host "应用程序已成功优雅终止！" -ForegroundColor Green
    Write-Host "退出代码: $($process.ExitCode)" -ForegroundColor Cyan
} else {
    Write-Host "应用程序未能在预期时间内终止，强制终止..." -ForegroundColor Red
    $process.Kill()
}

# 清理
Write-Host "清理测试文件..." -ForegroundColor Cyan
Remove-Item -Path "agent.exe" -Force -ErrorAction SilentlyContinue
Remove-Item -Path "test_plugin.exe" -Force -ErrorAction SilentlyContinue
Remove-Item -Path "plugins\test\test_plugin.exe" -Force -ErrorAction SilentlyContinue

Write-Host "测试完成！" -ForegroundColor Green
