package main

import (
	"flag"
	"log"
	"os"

	"github.com/AbrahamBass/gorf/internal/hotreload"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Uso: gorf [run]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCmd := flag.NewFlagSet("run", flag.ExitOnError)
		mainFile := runCmd.String("main", "main.go", "Archivo principal")
		runCmd.Parse(os.Args[2:])
		hotreload.HotReload(*mainFile)
	default:
		log.Fatal("Comando no reconocido")
	}
}
