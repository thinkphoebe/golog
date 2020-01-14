package golog_test

import (
	"fmt"
	"os"
	"testing"

	log "github.com/thinkphoebe/golog"
)

func TestHelloWorld(t *testing.T) {
	log.Infof("Hello world! this is Golog")
}

func TestConsoleWriter(t *testing.T) {
	log.SetLevel(log.LevelDebug)
	log.Debugf("Golog is a lightweight and expandable logger for Golang")
	log.Infof("Golog is compatible with the standard log library")
	log.Warnf("Golog can output to console with nicely color")
	log.Errorf("Golog output to file and rotate by date, hour and size")
	log.Criticalf("Golog can redirect stdout, stderr to log")

	log.Warnf("test Warning log with default color")
	log.GConsoleWriter.SetBrush("\033[34;1m", log.LevelWarn)
	log.Warnf("test Warning log with custom color")
	log.GConsoleWriter.SetColored(false)
	log.Warnf("test Warning log not colored")
}

func TestRotateWriter(t *testing.T) {
	fmtStr := `%(asctime) [%(levelno)][%(filename):%(lineno)] `
	w := log.NewRotateWriter("Rotate.log", log.RotateBySize)
	w.SetRotateSize(1000)
	log.Init(w, log.LevelDebug, fmtStr, false)
}

func TestAddOutput(t *testing.T) {
	log.Infof("write to stderr")
	wFile := log.NewRotateWriter("Multi.log", log.RotateByHour)
	log.AddOutput(wFile)
	log.Infof("both write to both stderr and log file")
	log.Infof("remove file outWriter")
	log.RemoveOutput(log.GConsoleWriter)
	log.Infof("only write to log file")
}

func TestAddRedirect(t *testing.T) {
	fmt.Fprint(os.Stdout, "write to stdout before redirect\n")
	fmt.Fprint(os.Stderr, "write to stderr before redirect\n")

	rstdr := log.AddRedirect(&os.Stderr, log.LevelError, "[stderr]")
	rstdo := log.AddRedirect(&os.Stdout, log.LevelInfo, "[stdout]")

	log.Infof("write to log 1")
	fmt.Fprint(os.Stdout, "write to stdout with recirect\n")
	fmt.Fprint(os.Stderr, "write to stderr with recirect\n")
	log.Infof("write to log 2")

	log.CancelRedirector(rstdr)
	log.CancelRedirector(rstdo)

	fmt.Fprint(os.Stdout, "write to stdout after redirect\n")
	fmt.Fprint(os.Stderr, "write to stderr after redirect\n")
}

func TestNewLogger(t *testing.T) {
	fmtStr := `%(asctime:ts) [%(levelno)][%(filename):%(function):%(lineno)] `
	//logger, err := log.NewLogger(log.NewConsoleWriter(os.Stderr), log.LevelDebug, fmtStr)
	logger, err := log.NewLogger(log.NewRotateWriter("golog.log", log.RotateByHour), log.LevelDebug, fmtStr, false)
	if err != nil {
		return
	}
	logger.Infof("hello world")
	logger.InfoJson(log.Json{"a": 1, "b": "abc", "c": 1.26})
}
