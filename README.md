# Golog [![GoDoc](https://godoc.org/github.com/thinkphoebe/golog?status.svg)](https://godoc.org/github.com/thinkphoebe/golog)
Golog is a lightweight and expandable logger for Golang.

## Features
* Compatible with the standard log package
* Output to console with nicely color
* Write to file and rotate by date, hour and size
* Write to multi-output simultaneously
* Customizable log header
* Redirect stdout, stderr to log
* Build-in five log levels

## Quick start
Golog has a global Logger object initiated with a global ConsoleWriter object.
After import Golog, you could call Debugf(), Infof(), Warnf(), Errorf() and Criticalf() to write logs of different levels with the global logger.
```
go get -u github.com/thinkphoebe/golog
```
```go
package main

import log "github.com/thinkphoebe/golog"

func main() {
    log.Infof("Hello world! this is Golog")
}
```

## Usage

#### Convert codes use the standard log package to Golog
The interface of Golog is compatible with the standard log package. You just need to modify import from
```go
import "log"
```
to
```go
import log "github.com/thinkphoebe/golog"
```

#### Output to console
By default, global Logger output to console with nicely colors.
You can call SetBrush() or SetColored() method of ConsoleWriter() to modify colors or disable log color.
If you use the global logger, you can get its ConsoleWriter object with log.GConsoleWriter.

```go
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
```
![Console colored](https://raw.githubusercontent.com/thinkphoebe/golog/master/console.png)

#### Output to file
Golog can output to file by add RotateWriter. RotateWrite support rotate log by hour, day and file size.
```go
fmtStr := `%(asctime) [%(levelno)][%(filename):%(lineno)] `
w := log.NewRotateWriter("Rotate.log", log.RotateBySize)
w.SetRotateSize(100 * 1000 * 1000)
log.Init(w, log.LevelDebug, fmtStr)
```

#### Write to multi-output
Golog can write to multi-output simultaneously. You can add output by AddOutput() and remove output by RemoveOutput().
```go
log.Infof("write to stderr")
wFile := log.NewRotateWriter("Multi.log", log.RotateByHour)
log.AddOutput(wFile)
log.Infof("both write to both stderr and log file")
log.Infof("remove file output %t", log.RemoveOutput(log.GConsoleWriter))
log.Infof("only write to log file")
```

#### Redirect system stdout and stderr to log
Please attention that AddRedirect() will modify the value of \*\*os.File you passed to it. 
So if you want to output logs to stderr, you must add ConsoleWriter before AddRedirect() called.
The following example use the global logger which GConsoleWriter already added at the package initialization.
```go
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
```
![Console colored](https://raw.githubusercontent.com/thinkphoebe/golog/master/redirect.png)

#### Customize log header
To customize log header, you need to pass a format string to log.NewLogger().
For the global logger, you could call log.Init() to re-initialize.
Format string supports the following keywords write as %(xxx).
* asctime - timestamp
* levelno - log level
* filename - source file name
* lineno - line no log in source file

```go
fmtStr := `%(asctime) [%(levelno)][%(filename):%(lineno)] `
log.Init(log.NewConsoleWriter(os.Stderr), log.LevelDebug, fmtStr)
```

#### New Logger
The most convenient way is to use the global logger. However, you could create new Logger object in some complicate usage.
```go
fmtStr := `%(asctime) [%(levelno)][%(filename):%(lineno)] `
logger, err := log.NewLogger(log.NewRotateWriter("golog.log", log.RotateByHour), log.LevelDebug, fmtStr)
if err != nil {
    // error handling
}
logger.Infof("Hello world, this is [%s]", "golog")
```
    
#### Customize log level
log.SetLevel() accept LevelDebug, LevelInfo, LevelWarn, LevelError and LevelCritical. 
If you set log level to LevelWarn, only Warn, Error and Critical logs will be output.
```go
log.SetLevel(LevelInfo)
```

#### Use you own writer
You can implement your own writers in addition to ConsoleWriter and RotateWriter. 
You need to implement IOutput interface defined in log.go and pass it to log.NewLogger() or log.Init().

#### [View documents from godoc](https://godoc.org/github.com/thinkphoebe/golog)
