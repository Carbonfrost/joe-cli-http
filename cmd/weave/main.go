package main

import (
	"os"

	"github.com/Carbonfrost/joe-cli-http/internal/cmd/weave"
)

func main() {
	weave.Run(os.Args)
}
