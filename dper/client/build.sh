#! /bin/sh

if [ ! -f "dperClient" ]; then
	go build dperClient.go
	else
	rm -f dperClient
	go build dperClient.go
fi

if [ ! -f "daemon" ]; then
	go build ./daemonfile/daemon.go
	else
	rm -f daemon
	go build ./daemonfile/daemon.go
fi

if [ ! -f "daemonClose" ]; then
	go build ./daemonfile/closeScript/daemonClose.go
	else
	rm -f daemonClose
	go build ./daemonfile/closeScript/daemonClose.go
fi

if [ ! -f "example_didSpectrumTrade"]; then
	go build ./../../chain_code_example/example_didSpectrumTrade/example_didSpectrumTrade.go
	else
	rm -f example_didSpectrumTrade
	go build ./../../chain_code_example/example_didSpectrumTrade/example_didSpectrumTrade.go
fi


