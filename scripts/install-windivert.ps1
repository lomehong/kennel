# WinDivert 安装脚本
# 用于DLP v2.0生产级部署

param(
    [string]$Version = "2.2.2",
    [string]$InstallPath = "C:\Program Files\WinDivert",
    [switch]$Force
)

# 检查管理员权限
if (-NOT ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Error "此脚本需要管理员权限运行"
    exit 1
}

Write-Host "DLP v2.0 - WinDivert 安装脚本" -ForegroundColor Green
Write-Host "版本: $Version" -ForegroundColor Yellow
Write-Host "安装路径: $InstallPath" -ForegroundColor Yellow

# 检查是否已安装
if (Test-Path "$InstallPath\WinDivert.dll" -and -not $Force) {
    Write-Host "WinDivert 已安装在 $InstallPath" -ForegroundColor Green
    Write-Host "使用 -Force 参数强制重新安装" -ForegroundColor Yellow
    exit 0
}

# 创建临时目录
$TempDir = "$env:TEMP\WinDivert-$Version"
if (Test-Path $TempDir) {
    Remove-Item $TempDir -Recurse -Force
}
New-Item -ItemType Directory -Path $TempDir -Force | Out-Null

try {
    # 下载WinDivert
    $DownloadUrl = "https://github.com/basil00/Divert/releases/download/v$Version/WinDivert-$Version-A.zip"
    $ZipFile = "$TempDir\WinDivert-$Version.zip"
    
    Write-Host "正在下载 WinDivert $Version..." -ForegroundColor Yellow
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $ZipFile -UseBasicParsing
    
    # 解压文件
    Write-Host "正在解压文件..." -ForegroundColor Yellow
    Expand-Archive -Path $ZipFile -DestinationPath $TempDir -Force
    
    # 创建安装目录
    if (-not (Test-Path $InstallPath)) {
        New-Item -ItemType Directory -Path $InstallPath -Force | Out-Null
    }
    
    # 检测系统架构
    $Architecture = if ([Environment]::Is64BitOperatingSystem) { "x64" } else { "x86" }
    $SourcePath = "$TempDir\WinDivert-$Version-A\$Architecture"
    
    Write-Host "检测到系统架构: $Architecture" -ForegroundColor Yellow
    
    # 复制文件
    Write-Host "正在安装 WinDivert 文件..." -ForegroundColor Yellow
    
    $FilesToCopy = @(
        "WinDivert.dll",
        "WinDivert.lib", 
        "WinDivert.sys",
        "WinDivert32.sys",
        "WinDivert64.sys"
    )
    
    foreach ($File in $FilesToCopy) {
        $SourceFile = "$SourcePath\$File"
        $DestFile = "$InstallPath\$File"
        
        if (Test-Path $SourceFile) {
            Copy-Item $SourceFile $DestFile -Force
            Write-Host "  已复制: $File" -ForegroundColor Green
        } else {
            Write-Warning "  文件不存在: $File"
        }
    }
    
    # 复制头文件
    $HeaderSource = "$TempDir\WinDivert-$Version-A\include\windivert.h"
    $HeaderDest = "$InstallPath\windivert.h"
    if (Test-Path $HeaderSource) {
        Copy-Item $HeaderSource $HeaderDest -Force
        Write-Host "  已复制: windivert.h" -ForegroundColor Green
    }
    
    # 添加到系统PATH
    $CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
    if ($CurrentPath -notlike "*$InstallPath*") {
        Write-Host "正在添加到系统PATH..." -ForegroundColor Yellow
        $NewPath = "$CurrentPath;$InstallPath"
        [Environment]::SetEnvironmentVariable("PATH", $NewPath, "Machine")
        Write-Host "  已添加到系统PATH" -ForegroundColor Green
    }
    
    # 验证安装
    Write-Host "正在验证安装..." -ForegroundColor Yellow
    
    $RequiredFiles = @("WinDivert.dll", "WinDivert.sys")
    $AllFilesExist = $true
    
    foreach ($File in $RequiredFiles) {
        $FilePath = "$InstallPath\$File"
        if (Test-Path $FilePath) {
            $FileInfo = Get-Item $FilePath
            Write-Host "  ✓ $File ($($FileInfo.Length) bytes)" -ForegroundColor Green
        } else {
            Write-Host "  ✗ $File (缺失)" -ForegroundColor Red
            $AllFilesExist = $false
        }
    }
    
    if ($AllFilesExist) {
        Write-Host "`nWinDivert 安装成功!" -ForegroundColor Green
        Write-Host "安装路径: $InstallPath" -ForegroundColor Yellow
        Write-Host "请重启命令提示符以使PATH更改生效" -ForegroundColor Yellow
        
        # 创建配置文件
        $ConfigFile = "$InstallPath\windivert.conf"
        $ConfigContent = @"
# WinDivert 配置文件
# DLP v2.0 生产级部署

[WinDivert]
Version=$Version
InstallPath=$InstallPath
Architecture=$Architecture
InstallDate=$(Get-Date -Format "yyyy-MM-dd HH:mm:ss")

[DLP]
# DLP相关配置
EnableTrafficInterception=true
DefaultFilter=outbound and tcp
BufferSize=65536
WorkerCount=4
"@
        Set-Content -Path $ConfigFile -Value $ConfigContent -Encoding UTF8
        Write-Host "已创建配置文件: $ConfigFile" -ForegroundColor Green
        
    } else {
        Write-Error "WinDivert 安装失败，某些文件缺失"
        exit 1
    }
    
} catch {
    Write-Error "安装过程中发生错误: $($_.Exception.Message)"
    exit 1
} finally {
    # 清理临时文件
    if (Test-Path $TempDir) {
        Remove-Item $TempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Write-Host "`n安装完成!" -ForegroundColor Green
Write-Host "现在可以运行 DLP v2.0 进行真实的网络流量拦截" -ForegroundColor Yellow
