@echo off
call "C:\Program Files\Microsoft Visual Studio\18\Community\VC\Auxiliary\Build\vcvars32.bat"
echo Compiling tcl2emf.cpp as 32-bit C++...
cl.exe /EHsc /MD tcl2emf.cpp /Fe:tcl2emf.exe gdi32.lib user32.lib winspool.lib
if %ERRORLEVEL% EQU 0 (
    echo Build successful!
    echo Output: tcl2emf.exe
) else (
    echo Build failed with error %ERRORLEVEL%
)
