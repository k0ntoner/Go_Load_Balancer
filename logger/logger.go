package logger

import (
	"log"
	"os"
)

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
)

func New(component string, color string) *log.Logger {
	prefix := color + "[" + component + "] " + ColorReset
	return log.New(os.Stdout, prefix, log.LstdFlags|log.Lmicroseconds)
}
