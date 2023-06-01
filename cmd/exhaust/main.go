package main

import (
	"exhaust/exhaust"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(exhaust.Analyzer)
}
