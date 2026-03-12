package log

import (
	"fmt"
	"time"
)

func Printf(format string, args ...any) {
	fmt.Printf("["+time.Now().Format("15:04:05")+"] "+format, args...)
}

func Println(args ...any) {
	fmt.Print("[" + time.Now().Format("15:04:05") + "] ")
	fmt.Println(args...)
}
