package logger

import "github.com/fatih/color"

func Underline(message interface{}) string {
	return color.New(color.Underline).Sprint(message)
}

func Bold(message interface{}) string {
	return color.New(color.Bold).Sprint(message)
}

func Red(message interface{}) string {
	return color.New(color.FgRed).Sprint(message)
}

func Green(message interface{}) string {
	return color.New(color.FgGreen).Sprint(message)
}

func Italic(message string) string {
	return color.New(color.Italic).Sprint(message)
}
