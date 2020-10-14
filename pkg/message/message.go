package message

import (
	"github.com/fatih/color"
)

func Error(s string) string {
	return color.RedString(s)
}

func Success(s string) string {
	return color.GreenString(s)
}

func Warn(s string) string {
	return color.YellowString(s)
}

func Info(s string) string {
	return color.CyanString(s)
}
