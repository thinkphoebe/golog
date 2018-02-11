package golog

import (
	"fmt"
	"os"
)

// Provide compatible interface for the standard log package
func (l *Logger) Print(v ...interface{}) { l.Output(LevelInfo, NormalDepth+1, fmt.Sprint(v...)) }

// Provide compatible interface for the standard log package
func (l *Logger) Println(v ...interface{}) { l.Output(LevelInfo, NormalDepth+1, fmt.Sprintln(v...)) }

// Provide compatible interface for the standard log package
func (l *Logger) Printf(format string, v ...interface{}) {
	l.Output(LevelInfo, NormalDepth+1, fmt.Sprintf(format, v...))
}

// Provide compatible interface for the standard log package
func (l *Logger) Fatal(v ...interface{}) {
	l.Output(LevelCritical, NormalDepth+1, fmt.Sprint(v...))
	os.Exit(1)
}

// Provide compatible interface for the standard log package
func (l *Logger) Fatalln(v ...interface{}) {
	l.Output(LevelCritical, NormalDepth+1, fmt.Sprintln(v...))
	os.Exit(1)
}

// Provide compatible interface for the standard log package
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Output(LevelCritical, NormalDepth+1, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Provide compatible interface for the standard log package
func (l *Logger) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.Output(LevelCritical, NormalDepth+1, s)
	panic(s)
}

// Provide compatible interface for the standard log package
func (l *Logger) Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	l.Output(LevelCritical, NormalDepth+1, s)
	panic(s)
}

// Provide compatible interface for the standard log package
func (l *Logger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	l.Output(LevelCritical, NormalDepth+1, s)
	panic(s)
}

// Provide compatible interface for the standard log package
func Print(v ...interface{}) { std.Output(LevelInfo, NormalDepth+1, fmt.Sprint(v...)) }

// Provide compatible interface for the standard log package
func Println(v ...interface{}) { std.Output(LevelInfo, NormalDepth+1, fmt.Sprintln(v...)) }

// Provide compatible interface for the standard log package
func Printf(format string, v ...interface{}) {
	std.Output(LevelInfo, NormalDepth+1, fmt.Sprintf(format, v...))
}

// Provide compatible interface for the standard log package
func Fatal(v ...interface{}) {
	std.Output(LevelCritical, NormalDepth+1, fmt.Sprint(v...))
	os.Exit(1)
}

// Provide compatible interface for the standard log package
func Fatalln(v ...interface{}) {
	std.Output(LevelCritical, NormalDepth+1, fmt.Sprintln(v...))
	os.Exit(1)
}

// Provide compatible interface for the standard log package
func Fatalf(format string, v ...interface{}) {
	std.Output(LevelCritical, NormalDepth+1, fmt.Sprintf(format, v...))
	os.Exit(1)
}

// Provide compatible interface for the standard log package
func Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	std.Output(LevelCritical, NormalDepth+1, s)
	panic(s)
}

// Provide compatible interface for the standard log package
func Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	std.Output(LevelCritical, NormalDepth+1, s)
	panic(s)
}

// Provide compatible interface for the standard log package
func Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	std.Output(LevelCritical, NormalDepth+1, s)
	panic(s)
}
