# DLP网络过滤器验证脚本
# 用于验证DLP模块是否正确过滤私有网络流量

param(
    [string]$DLPPath = "..\dlp.exe",
    [int]$TestDuration = 30,
    [switch]$Verbose
)

Write-Host "=== DLP网络过滤器验证脚本 ===" -ForegroundColor Green
Write-Host ""

# 检查DLP可执行文件是否存在
if (-not (Test-Path $DLPPath)) {
    Write-Host "错误: 找不到DLP可执行文件: $DLPPath" -ForegroundColor Red
    Write-Host "请先编译DLP模块: go build -o dlp.exe ." -ForegroundColor Yellow
    exit 1
}

Write-Host "1. 检查DLP可执行文件..." -ForegroundColor Cyan
Write-Host "   ✓ 找到DLP可执行文件: $DLPPath" -ForegroundColor Green

# 运行过滤器测试
Write-Host ""
Write-Host "2. 运行过滤器逻辑测试..." -ForegroundColor Cyan
try {
    $testResult = & go run filter_validator.go 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "   ✓ 过滤器逻辑测试通过" -ForegroundColor Green
        if ($Verbose) {
            Write-Host $testResult -ForegroundColor Gray
        }
    } else {
        Write-Host "   ✗ 过滤器逻辑测试失败" -ForegroundColor Red
        Write-Host $testResult -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "   ✗ 无法运行过滤器测试: $_" -ForegroundColor Red
    exit 1
}

# 启动DLP模块进行实际测试
Write-Host ""
Write-Host "3. 启动DLP模块进行实际流量测试..." -ForegroundColor Cyan
Write-Host "   测试持续时间: $TestDuration 秒" -ForegroundColor Gray

# 创建临时日志文件
$logFile = "dlp_test_$(Get-Date -Format 'yyyyMMdd_HHmmss').log"
$dlpProcess = $null

try {
    # 启动DLP进程
    Write-Host "   启动DLP进程..." -ForegroundColor Gray
    $dlpProcess = Start-Process -FilePath $DLPPath -RedirectStandardOutput $logFile -RedirectStandardError "$logFile.err" -PassThru -NoNewWindow
    
    if (-not $dlpProcess) {
        throw "无法启动DLP进程"
    }
    
    Write-Host "   ✓ DLP进程已启动 (PID: $($dlpProcess.Id))" -ForegroundColor Green
    
    # 等待DLP初始化
    Start-Sleep -Seconds 5
    
    # 检查进程是否仍在运行
    if ($dlpProcess.HasExited) {
        $errorContent = Get-Content "$logFile.err" -ErrorAction SilentlyContinue
        throw "DLP进程启动后立即退出。错误信息: $errorContent"
    }
    
    Write-Host "   ✓ DLP进程运行正常" -ForegroundColor Green
    
    # 生成测试流量
    Write-Host ""
    Write-Host "4. 生成测试网络流量..." -ForegroundColor Cyan
    
    $testTargets = @(
        @{Name="本地回环"; URL="http://127.0.0.1:8080"; ShouldFilter=$true},
        @{Name="私有网络"; URL="http://192.168.1.1"; ShouldFilter=$true},
        @{Name="Google DNS"; URL="http://8.8.8.8"; ShouldFilter=$false},
        @{Name="Cloudflare DNS"; URL="http://1.1.1.1"; ShouldFilter=$false}
    )
    
    foreach ($target in $testTargets) {
        Write-Host "   测试访问: $($target.Name) ($($target.URL))" -ForegroundColor Gray
        try {
            # 使用短超时时间，我们不关心是否成功连接，只关心是否产生流量
            $null = Invoke-WebRequest -Uri $target.URL -TimeoutSec 2 -ErrorAction SilentlyContinue
        } catch {
            # 忽略连接错误，我们只是想产生网络流量
        }
        Start-Sleep -Seconds 1
    }
    
    Write-Host "   ✓ 测试流量生成完成" -ForegroundColor Green
    
    # 等待更多流量和日志生成
    Write-Host ""
    Write-Host "5. 等待日志生成和分析..." -ForegroundColor Cyan
    Start-Sleep -Seconds ($TestDuration - 10)
    
    # 分析日志
    Write-Host ""
    Write-Host "6. 分析DLP日志..." -ForegroundColor Cyan
    
    if (Test-Path $logFile) {
        $logContent = Get-Content $logFile -ErrorAction SilentlyContinue
        
        # 检查是否有私有网络流量被记录
        $privateIPs = @("127\.", "10\.", "172\.1[6-9]\.", "172\.2[0-9]\.", "172\.3[0-1]\.", "192\.168\.")
        $foundPrivateTraffic = $false
        $publicTrafficCount = 0
        
        foreach ($line in $logContent) {
            foreach ($pattern in $privateIPs) {
                if ($line -match $pattern) {
                    $foundPrivateTraffic = $true
                    Write-Host "   ⚠ 发现私有网络流量记录: $line" -ForegroundColor Yellow
                }
            }
            
            # 检查公网流量
            if ($line -match "8\.8\.8\.8|1\.1\.1\.1") {
                $publicTrafficCount++
            }
        }
        
        Write-Host ""
        Write-Host "7. 验证结果:" -ForegroundColor Cyan
        
        if (-not $foundPrivateTraffic) {
            Write-Host "   ✓ 未发现私有网络流量记录 - 过滤器工作正常" -ForegroundColor Green
        } else {
            Write-Host "   ✗ 发现私有网络流量记录 - 过滤器可能存在问题" -ForegroundColor Red
        }
        
        if ($publicTrafficCount -gt 0) {
            Write-Host "   ✓ 检测到公网流量记录 ($publicTrafficCount 条) - DLP功能正常" -ForegroundColor Green
        } else {
            Write-Host "   ⚠ 未检测到公网流量记录 - 可能需要检查DLP配置" -ForegroundColor Yellow
        }
        
        if ($Verbose) {
            Write-Host ""
            Write-Host "完整日志内容:" -ForegroundColor Gray
            Write-Host $logContent -ForegroundColor Gray
        }
        
    } else {
        Write-Host "   ⚠ 未找到DLP日志文件，可能DLP未正常启动" -ForegroundColor Yellow
    }
    
} catch {
    Write-Host "   ✗ 测试过程中发生错误: $_" -ForegroundColor Red
} finally {
    # 清理：停止DLP进程
    if ($dlpProcess -and -not $dlpProcess.HasExited) {
        Write-Host ""
        Write-Host "8. 清理测试环境..." -ForegroundColor Cyan
        try {
            $dlpProcess.Kill()
            $dlpProcess.WaitForExit(5000)
            Write-Host "   ✓ DLP进程已停止" -ForegroundColor Green
        } catch {
            Write-Host "   ⚠ 停止DLP进程时出错: $_" -ForegroundColor Yellow
        }
    }
    
    # 清理临时文件
    if (Test-Path $logFile) {
        Write-Host "   日志文件保存为: $logFile" -ForegroundColor Gray
    }
    if (Test-Path "$logFile.err") {
        $errorContent = Get-Content "$logFile.err" -ErrorAction SilentlyContinue
        if ($errorContent) {
            Write-Host "   错误日志: $errorContent" -ForegroundColor Red
        }
        Remove-Item "$logFile.err" -ErrorAction SilentlyContinue
    }
}

Write-Host ""
Write-Host "=== 验证完成 ===" -ForegroundColor Green
Write-Host ""
Write-Host "使用说明:" -ForegroundColor Cyan
Write-Host "- 如果过滤器工作正常，DLP日志中不应出现127.x.x.x、10.x.x.x、172.16-31.x.x、192.168.x.x等私有地址"
Write-Host "- 只有发往公网IP的流量应该被记录和审计"
Write-Host "- 如果发现问题，请检查WinDivert过滤器配置和应用层过滤逻辑"
Write-Host ""
Write-Host "手动验证方法:" -ForegroundColor Cyan
Write-Host "1. 启动DLP: .\dlp.exe"
Write-Host "2. 访问本地服务: curl http://127.0.0.1:8080 (不应在日志中出现)"
Write-Host "3. 访问私有网络: curl http://192.168.1.1 (不应在日志中出现)"
Write-Host "4. 访问公网服务: curl http://8.8.8.8 (应该在日志中出现)"
