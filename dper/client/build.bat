@echo off
if exist dperClient.exe (  %创建新的dperClient.exe%
    del dperClient.exe
    go build dperClient.go
) else (
    go build dperClient.go
)

if exist daemon.exe (  %daemon.exe%
    del daemon.exe
    go build .\daemonfile\daemon.go
) else (
    go build .\daemonfile\daemon.go
)

if exist daemonClose.exe (  %daemonClose.exe%
    del daemonClose.exe
    go build .\daemonfile\closeScript\daemonClose.go
) else (
    go build .\daemonfile\closeScript\daemonClose.go
)

if exist .\..\..\chain_code_example\example_didSpectrumTrade\example_didSpectrumTrade.exe (
    del .\..\..\chain_code_example\example_didSpectrumTrade\example_didSpectrumTrade.exe
    go build  .\..\..\chain_code_example\example_didSpectrumTrade\example_didSpectrumTrade.go
) else (
    go build .\..\..\chain_code_example\example_didSpectrumTrade\example_didSpectrumTrade.go
)







