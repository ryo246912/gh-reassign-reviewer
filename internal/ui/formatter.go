package ui

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

func PadRight(str string, width int) string {
	w := runewidth.StringWidth(str)
	if w < width {
		return str + strings.Repeat(" ", width-w)
	}
	return str
}
