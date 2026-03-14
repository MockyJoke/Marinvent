@echo off
echo Building georef_tool.exe (32-bit)...

call "C:\Program Files\Microsoft Visual Studio\18\Community\VC\Auxiliary\Build\vcvars32.bat" >nul 2>&1
if %ERRORLEVEL% neq 0 (
    call "C:\Program Files\Microsoft Visual Studio\2022\Community\VC\Auxiliary\Build\vcvars32.bat" >nul 2>&1
)
if %ERRORLEVEL% neq 0 (
    call "C:\Program Files (x86)\Microsoft Visual Studio\2019\Community\VC\Auxiliary\Build\vcvars32.bat" >nul 2>&1
)
if %ERRORLEVEL% neq 0 (
    call "C:\Program Files (x86)\Microsoft Visual Studio\2017\Community\VC\Auxiliary\Build\vcvars32.bat" >nul 2>&1
)

cl /nologo /O2 /DNDEBUG /MD georef_tool.cpp /Fe:georef_tool.exe /link user32.lib gdi32.lib advapi32.lib winspool.lib

if exist georef_tool.exe (
    echo Build successful: georef_tool.exe
    del georef_tool.obj 2>nul
) else (
    echo Build failed!
    exit /b 1
)
