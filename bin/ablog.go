package main

import (
	"github.com/johnnylee/ablog"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		ablog.RootPrefix = os.Args[1]
	}

	ablog.Main()
}
