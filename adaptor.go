// This file provide a compatible interface to standard log module.
// To migrate from standard log module, programs only need to modify from
//     import "log"
// to
//     import log "github.com/thinkphoebe/golog"
package golog

import (
	"fmt"
	"os"
)

func (l *Logger) Print(v ...interface{})   { l.Output(LevelInfo, CallDepth+1, fmt.Sprint(v...)) }
func (l *Logger) Println(v ...interface{}) { l.Output(LevelInfo, CallDepth+1, fmt.Sprintln(v...)) }

func (l *Logger) Printf(format string, v ...interface{}) {
	l.Output(LevelInfo, CallDepth+1, fmt.Sprintf(format, v...))
}

func (l *Logger) Fatal(v ...interface{}) {
	l.Output(LevelCritical, CallDepth+1, fmt.Sprint(v...))
	os.Exit(1)
}

func (l *Logger) Fatalln(v ...interface{}) {
	l.Output(LevelCritical, CallDepth+1, fmt.Sprintln(v...))
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Output(LevelCritical, CallDepth+1, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *Logger) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.Output(LevelCritical, CallDepth+1, s)
	panic(s)
}

func (l *Logger) Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	l.Output(LevelCritical, CallDepth+1, s)
	panic(s)
}

func (l *Logger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	l.Output(LevelCritical, CallDepth+1, s)
	panic(s)
}

// ================ the following functions write to the standard logger ================

func Print(v ...interface{})   { std.Output(LevelInfo, CallDepth+1, fmt.Sprint(v...)) }
func Println(v ...interface{}) { std.Output(LevelInfo, CallDepth+1, fmt.Sprintln(v...)) }

func Printf(format string, v ...interface{}) {
	std.Output(LevelInfo, CallDepth+1, fmt.Sprintf(format, v...))
}

func Fatal(v ...interface{}) {
	std.Output(LevelCritical, CallDepth+1, fmt.Sprint(v...))
	os.Exit(1)
}

func Fatalln(v ...interface{}) {
	std.Output(LevelCritical, CallDepth+1, fmt.Sprintln(v...))
	os.Exit(1)
}

func Fatalf(format string, v ...interface{}) {
	std.Output(LevelCritical, CallDepth+1, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	std.Output(LevelCritical, CallDepth+1, s)
	panic(s)
}

func Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	std.Output(LevelCritical, CallDepth+1, s)
	panic(s)
}

func Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	std.Output(LevelCritical, CallDepth+1, s)
	panic(s)
}
