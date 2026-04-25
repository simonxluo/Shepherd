@echo off
REM Shepherd Windows 编译脚本
REM 用法: scripts\build.bat [version]

setlocal enabledelayedexpansion

:: 项目信息
set PROJECT_NAME=shepherd
set BUILD_DIR=build
set CMD_DIR=cmd\shepherd
set VERSION=%1
if "%VERSION%"=="" set VERSION=dev

:: 构建信息
for /f "tokens=1-4 delims=/ " %%a in ('date /t') do (
    set BUILD_DATE=%%a-%%b-%%c
)
for /f "tokens=1-2 delims=: " %%a in ('time /t') do (
    set BUILD_TIME=%%a:%%b
)
set BUILD_DATETIME=%BUILD_DATE%T%BUILD_TIME%Z

:: 设置 Go 代理（如果需要）
if "%GOPROXY%"=="" set GOPROXY=https://goproxy.cn,direct

echo ========================================
echo   Shepherd Build Script (Windows)
echo ========================================
echo.
echo 版本: %VERSION%
echo 构建时间: %BUILD_DATETIME%
echo.

:: 清理旧的构建文件
echo 清理旧的构建文件...
if exist %BUILD_DIR% rmdir /s /q %BUILD_DIR%
mkdir %BUILD_DIR%

:: 设置编译参数
set LDFLAGS=-X main.Version=%VERSION%
set LDFLAGS=%LDFLAGS% -X main.BuildTime=%BUILD_DATETIME%
set LDFLAGS=%LDFLAGS% -s -w

:: 检测架构
set TARGET_ARCH=amd64
if defined PROCESSOR_ARCHITECTURE (
    if "%PROCESSOR_ARCHITECTURE%"=="ARM64" set TARGET_ARCH=arm64
)

set BINARY_NAME=%PROJECT_NAME%-windows-%TARGET_ARCH%.exe

echo 编译目标: windows/%TARGET_ARCH%
echo 输出文件: %BUILD_DIR%\%BINARY_NAME%
echo.

:: 编译
echo 开始编译...
go build -ldflags "%LDFLAGS%" -o "%BUILD_DIR%\%BINARY_NAME%" "%CMD_DIR%\main.go"

if %ERRORLEVEL% EQU 0 (
    echo [√] 编译成功!
) else (
    echo [×] 编译失败
    exit /b 1
)

echo.
echo ========================================
echo   构建完成
echo ========================================
echo 二进制: %BUILD_DIR%\%BINARY_NAME%
echo.

:: 运行测试（可选）
if "%RUN_TESTS%"=="true" (
    echo 运行测试...
    go test ./... -v
)

:: 使用提示
echo 使用方法:
echo   %BUILD_DIR%\%BINARY_NAME% serve                       # 启动服务 (默认 hybrid 模式)
echo   %BUILD_DIR%\%BINARY_NAME% serve --web --build         # 编译 + 前后端一键启动
echo   %BUILD_DIR%\%BINARY_NAME% serve --config config.yaml  # 使用指定配置文件
echo   %BUILD_DIR%\%BINARY_NAME% stop                        # 停止服务
echo   %BUILD_DIR%\%BINARY_NAME% version                     # 查看版本
echo.

endlocal
