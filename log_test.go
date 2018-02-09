package golog

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestConsole(t *testing.T) {
	SetLevel(LevelDebug)
	Debugf("Golog is a lightweight and expandable logger for Golang")
	Infof("Golog is compatible with the standard log library")
	Warnf("Golog can output to console with nicely color")
	Errorf("Golog output to file and rotate by date, hour and size")
	Criticalf("Golog can redirect stdout, stderr to log")

	fmtStr := "%(asctime) [%(levelno)][%(filename):%(lineno)] "
	w := NewConsoleWriter(os.Stderr)
	Init(w, LevelDebug, fmtStr)
	Warnf("test Warning log with default color")
	w.SetBrush("\033[34;1m", LevelWarn)
	Warnf("test Warning log with custom color")
	w.SetColored(false)
	Warnf("test Warning log not colored")
}

func TestFormatHeader(t *testing.T) {
	fmtStr := `%(asctime) [%(levelno)][%(filename):%(lineno)] `
	l, err := NewLogger(NewConsoleWriter(os.Stderr), LevelDebug, fmtStr)
	if err != nil {
		t.Errorf("NewLogger got err [%v]", err)
	}

	l.Outputf(LevelInfo, CallDepth, "this is a test [%d]", 666)
}

func TestAddRemoveOutput(t *testing.T) {
	w := NewRotateWriter("TestAddRemoveOutput.log", RotateByHour)
	AddOutput(w)
	Infof("both write to both stderr and log file")
	Infof("remove file output %t", RemoveOutput(w))
	Infof("only write to stderr")
}

func TestRedirect(t *testing.T) {
	w := NewRotateWriter("TestRedirect.log", RotateByHour)
	AddOutput(w)

	fmt.Fprint(os.Stdout, "write to stdout before redirect\n")
	fmt.Fprint(os.Stderr, "write to stderr before redirect\n")

	rstdr := AddRedirect(&os.Stderr, LevelError, "[stderr]")
	rstdo := AddRedirect(&os.Stdout, LevelInfo, "[stdout]")

	Infof("write to log file 1")
	fmt.Fprint(os.Stdout, "write to stdout with recirect\n")
	fmt.Fprint(os.Stderr, "write to stderr with recirect\n")
	Infof("write to log file 2")

	CancelRedirector(rstdr)
	CancelRedirector(rstdo)

	fmt.Fprint(os.Stdout, "write to stdout after redirect\n")
	fmt.Fprint(os.Stderr, "write to stderr after redirect\n")

	time.Sleep(time.Second)
}

func TestRotate(t *testing.T) {
	fmtStr := "%(asctime) [%(levelno)][%(filename):%(lineno)] "
	w := NewRotateWriter("TestRotate.log", RotateBySize)
	w.SetRotateSize(100 * 1000 * 1000)
	Init(w, LevelDebug, fmtStr)
}
