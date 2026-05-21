@echo off
REM Shepherd 运行脚本 (Windows)
REM 使用 Cobra CLI: shepherd serve [--config path] [--web] [--build] [--host addr] [--port num]
REM 节点角色通过配置文件 node.role 字段设置 (hybrid/master/client)

setlocal enabledelayedexpansion

set "SCRIPT_DIR=%~dp0"
set "PROJECT_DIR=%SCRIPT_DIR%.."
set "BUILD_DIR=%PROJECT_DIR%\build"
set "BINARY_NAME=shepherd.exe"

set "INFO=[INFO]"
set "SUCCESS=[SUCCESS]"
set "WARNING=[WARNING]"
set "ERROR=[ERROR]"

:show_help
echo   Shepherd 运行脚本 (Windows)
echo.
echo 用法: %~nx0 [选项]
echo.
echo 选项:
echo     -h, --help         显示此帮助信息
echo     -b, --build        运行前先编译
echo     -v, --version      显示版本信息
echo     --config PATH      指定配置文件路径
echo     --web              启动前端开发服务器
echo     --host ADDR        监听地址
echo     --port PORT        监听端口
echo.
echo 节点角色通过配置文件 node.role 字段设置:
echo     hybrid             混合模式 (默认)
echo     master             Master 模式 - 管理多个 Client 节点
echo     client             Client 模式 - 作为工作节点
echo.
echo 示例:
echo     REM 使用默认配置启动
echo     %~nx0
echo.
echo     REM 编译后启动，同时启动前端
echo     %~nx0 --build --web
echo.
echo     REM 使用自定义配置文件
echo     %~nx0 --config config\node\server.config.yaml
echo.
goto :eof

:check_binary
if not exist "%BUILD_DIR%\%BINARY_NAME%" (
    echo %WARNING% 二进制文件不存在: %BUILD_DIR%\%BINARY_NAME%
    set /p BUILD_NOW="是否现在编译? (y/N): "
    if /i "!BUILD_NOW!"=="y" (
        cd /d "%SCRIPT_DIR%"
        call build.bat
        cd /d "%PROJECT_DIR%"
    ) else (
        echo %ERROR% 无法继续，请先编译项目
        exit /b 1
    )
)
goto :eof

:show_version
if exist "%BUILD_DIR%\%BINARY_NAME%" (
    "%BUILD_DIR%\%BINARY_NAME%" version
) else (
    echo %ERROR% 二进制文件不存在，请先编译
    exit /b 1
)
exit /b 0

:main
set "BUILD_FIRST=0"
set "SERVE_ARGS="

:parse_args
if "%~1"=="" goto :args_done
if /i "%~1"=="-h" goto :show_help
if /i "%~1"=="--help" goto :show_help
if /i "%~1"=="-b" (
    set "BUILD_FIRST=1"
    shift
    goto :parse_args
)
if /i "%~1"=="-v" goto :show_version
if /i "%~1"=="--version" goto :show_version
if /i "%~1"=="--config" (
    set "SERVE_ARGS=!SERVE_ARGS! --config %~2"
    shift /2
    goto :parse_args
)
if /i "%~1"=="--web" (
    set "SERVE_ARGS=!SERVE_ARGS! --web"
    shift
    goto :parse_args
)
if /i "%~1"=="--host" (
    set "SERVE_ARGS=!SERVE_ARGS! --host %~2"
    shift /2
    goto :parse_args
)
if /i "%~1"=="--port" (
    set "SERVE_ARGS=!SERVE_ARGS! --port %~2"
    shift /2
    goto :parse_args
)
echo %ERROR% 未知参数: %~1
goto :show_help

:args_done

if "%BUILD_FIRST%"=="1" (
    echo %INFO% 编译项目...
    cd /d "%SCRIPT_DIR%"
    call build.bat
    cd /d "%PROJECT_DIR%"
    echo %SUCCESS% 编译完成
)

call :check_binary

echo.
echo.
echo   Shepherd
echo.
echo.

cd /d "%PROJECT_DIR%"
"%BUILD_DIR%\%BINARY_NAME%" serve %SERVE_ARGS%

goto :eof

call :main %*
