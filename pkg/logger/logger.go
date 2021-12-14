package logger

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	red    = color.New(color.FgRed)
	yellow = color.New(color.FgYellow)
	blue   = color.New(color.FgBlue)
	cyan   = color.New(color.FgCyan)
	green  = color.New(color.FgGreen)
)

func Warn(message string, attributes ...interface{}) {
	yellow.PrintFunc()("[WARNING] | ")
	fmt.Println(fmt.Sprintf(message, attributes...))
}

func Error(message string, attributes ...interface{}) {
	red.PrintFunc()("[ERROR]   | ")
	fmt.Println(fmt.Sprintf(message, attributes...))
}

func Fatal(message string, attributes ...interface{}) {
	red.PrintFunc()("[FATAL]   | ")
	fmt.Println(fmt.Sprintf(message, attributes...))
	os.Exit(1)
}

func Success(message string, attributes ...interface{}) {
	green.PrintFunc()("[SUCCESS] | ")
	fmt.Println(fmt.Sprintf(message, attributes...))
}

func Info(message string, attributes ...interface{}) {
	blue.PrintFunc()("[INFO]    | ")
	fmt.Println(fmt.Sprintf(message, attributes...))
}

func InfoScan(message string) string {
	blue.PrintFunc()("[INFO]    | ")
	fmt.Print(message)
	var input string
	fmt.Scanln(&input)
	return input
}
