@echo off
REM ShareSerial Client Launcher
REM Simple script to start the client

echo ========================================
echo ShareSerial Windows Client Launcher
echo ========================================
echo

REM Default settings
set SERVER=192.168.246.17:7700
set PORT=8888

echo Server: %SERVER%
echo Local Port: %PORT%
echo.
echo Press any key to start client...
pause >nul

echo.
echo Starting ShareSerial Client...
echo.

shareserial-client-windows.exe --server %SERVER% --local-port %PORT%

echo.
echo Client stopped.
pause