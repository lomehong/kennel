@echo off
REM DLP v2.0 - WinDivert 快速安装脚本
REM 用于生产级部署

echo ========================================
echo DLP v2.0 - WinDivert 快速安装脚本
echo ========================================

REM 检查管理员权限
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo 错误: 此脚本需要管理员权限运行
    echo 请右键点击此脚本，选择"以管理员身份运行"
    pause
    exit /b 1
)

echo 检测到管理员权限...

REM 设置变量
set WINDIVERT_VERSION=2.2.2
set INSTALL_DIR=C:\Program Files\WinDivert
set TEMP_DIR=%TEMP%\windivert-install
set DOWNLOAD_URL=https://github.com/basil00/Divert/releases/download/v%WINDIVERT_VERSION%/WinDivert-%WINDIVERT_VERSION%-A.zip

echo 版本: %WINDIVERT_VERSION%
echo 安装目录: %INSTALL_DIR%

REM 检查是否已安装
if exist "%INSTALL_DIR%\WinDivert.dll" (
    echo WinDivert 已安装在 %INSTALL_DIR%
    echo 如需重新安装，请先删除该目录
    pause
    exit /b 0
)

echo 正在创建安装目录...
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"

echo 正在创建临时目录...
if exist "%TEMP_DIR%" rmdir /s /q "%TEMP_DIR%"
mkdir "%TEMP_DIR%"

echo 正在下载 WinDivert %WINDIVERT_VERSION%...
echo 下载地址: %DOWNLOAD_URL%

REM 使用PowerShell下载文件
powershell -Command "& {[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; Invoke-WebRequest -Uri '%DOWNLOAD_URL%' -OutFile '%TEMP_DIR%\windivert.zip' -UseBasicParsing}"

if not exist "%TEMP_DIR%\windivert.zip" (
    echo 错误: 下载失败
    echo 请检查网络连接或手动下载安装
    pause
    exit /b 1
)

echo 下载完成，正在解压...

REM 使用PowerShell解压文件
powershell -Command "& {Expand-Archive -Path '%TEMP_DIR%\windivert.zip' -DestinationPath '%TEMP_DIR%' -Force}"

REM 检测系统架构
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    set ARCH=x64
) else (
    set ARCH=x86
)

echo 检测到系统架构: %ARCH%

set SOURCE_DIR=%TEMP_DIR%\WinDivert-%WINDIVERT_VERSION%-A\%ARCH%

echo 正在安装 WinDivert 文件...

REM 复制主要文件
copy "%SOURCE_DIR%\WinDivert.dll" "%INSTALL_DIR%\" >nul
if %errorLevel% neq 0 (
    echo 错误: 复制 WinDivert.dll 失败
    goto cleanup
)
echo   已复制: WinDivert.dll

copy "%SOURCE_DIR%\WinDivert.sys" "%INSTALL_DIR%\" >nul
if %errorLevel% neq 0 (
    echo 错误: 复制 WinDivert.sys 失败
    goto cleanup
)
echo   已复制: WinDivert.sys

REM 复制其他文件（可选）
if exist "%SOURCE_DIR%\WinDivert.lib" (
    copy "%SOURCE_DIR%\WinDivert.lib" "%INSTALL_DIR%\" >nul
    echo   已复制: WinDivert.lib
)

if exist "%SOURCE_DIR%\WinDivert32.sys" (
    copy "%SOURCE_DIR%\WinDivert32.sys" "%INSTALL_DIR%\" >nul
    echo   已复制: WinDivert32.sys
)

if exist "%SOURCE_DIR%\WinDivert64.sys" (
    copy "%SOURCE_DIR%\WinDivert64.sys" "%INSTALL_DIR%\" >nul
    echo   已复制: WinDivert64.sys
)

REM 复制头文件
if exist "%TEMP_DIR%\WinDivert-%WINDIVERT_VERSION%-A\include\windivert.h" (
    copy "%TEMP_DIR%\WinDivert-%WINDIVERT_VERSION%-A\include\windivert.h" "%INSTALL_DIR%\" >nul
    echo   已复制: windivert.h
)

REM 添加到系统PATH
echo 正在添加到系统PATH...
for /f "tokens=2*" %%a in ('reg query "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v PATH 2^>nul') do set CURRENT_PATH=%%b

echo %CURRENT_PATH% | find /i "%INSTALL_DIR%" >nul
if %errorLevel% neq 0 (
    reg add "HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v PATH /t REG_EXPAND_SZ /d "%CURRENT_PATH%;%INSTALL_DIR%" /f >nul
    echo   已添加到系统PATH
) else (
    echo   PATH中已存在安装目录
)

REM 验证安装
echo 正在验证安装...
if exist "%INSTALL_DIR%\WinDivert.dll" (
    echo   ✓ WinDivert.dll
) else (
    echo   ✗ WinDivert.dll (缺失)
    goto cleanup
)

if exist "%INSTALL_DIR%\WinDivert.sys" (
    echo   ✓ WinDivert.sys
) else (
    echo   ✗ WinDivert.sys (缺失)
    goto cleanup
)

REM 创建配置文件
echo 正在创建配置文件...
(
echo # WinDivert 配置文件
echo # DLP v2.0 生产级部署
echo.
echo [WinDivert]
echo Version=%WINDIVERT_VERSION%
echo InstallPath=%INSTALL_DIR%
echo Architecture=%ARCH%
echo InstallDate=%DATE% %TIME%
echo.
echo [DLP]
echo # DLP相关配置
echo EnableTrafficInterception=true
echo DefaultFilter=outbound and tcp
echo BufferSize=65536
echo WorkerCount=4
) > "%INSTALL_DIR%\windivert.conf"

echo   已创建配置文件: windivert.conf

:cleanup
echo 正在清理临时文件...
if exist "%TEMP_DIR%" rmdir /s /q "%TEMP_DIR%"

echo.
echo ========================================
echo WinDivert 安装完成!
echo ========================================
echo 安装路径: %INSTALL_DIR%
echo 版本: %WINDIVERT_VERSION%
echo 架构: %ARCH%
echo.
echo 注意事项:
echo 1. 请重启命令提示符以使PATH更改生效
echo 2. DLP v2.0 现在可以进行真实的网络流量拦截
echo 3. 运行DLP需要管理员权限
echo.
echo 现在可以启动 DLP v2.0 进行生产级数据泄露防护!
echo ========================================

pause
