package banner

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
)

const dateFormat = "2006/01/02 15:04:05"

func Printf(format string, args ...interface{}) {
	format = strings.TrimRight(format, "\n")
	Println(fmt.Sprintf(format, args...))
}

func Println(args ...interface{}) {
	dateStr := time.Now().In(time.Local).Format(dateFormat)
	color.Green("%s", dateStr)
	color.Green("%s %s", dateStr, fmt.Sprint(args...))
}
