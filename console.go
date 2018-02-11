package golog

import "io"

// Write logs to console with colors
type ConsoleWriter struct {
	colored bool
	brush   [int(LevelCritical) + 1][]byte
	dst     io.Writer
}

var defaultBrush = [...][]byte{
	int(LevelDebug):    []byte("\033[32m"),
	int(LevelInfo):     []byte("\033[0m"),
	int(LevelWarn):     []byte("\033[33;1m"),
	int(LevelError):    []byte("\033[31;1m"),
	int(LevelCritical): []byte("\033[35;1m"),
}

var resetBrush = []byte("\033[0m")

// Create a new ConsoleWriter
func NewConsoleWriter(dst io.Writer) *ConsoleWriter {
	w := &ConsoleWriter{
		colored: true,
		brush:   defaultBrush,
		dst:     dst,
	}
	return w
}

// Set display with color or not
func (w *ConsoleWriter) SetColored(colored bool) {
	w.colored = colored
}

// Set colors for specified level of log
func (w *ConsoleWriter) SetBrush(brush string, level LogLevel) {
	w.brush[level] = []byte(brush)
}

func (w *ConsoleWriter) Write(msg []byte, level LogLevel) {
	if w.colored {
		w.dst.Write(w.brush[level])
		w.dst.Write(msg)
		w.dst.Write(resetBrush)
	} else {
		w.dst.Write(msg)
	}
}
