package main

import (
	"dpchain/p2p/whisper/cli"
	"os"
)

func main() {
	defer os.Exit(0)
	cmd := cli.NewCommandLine()
	cmd.Run()
}
