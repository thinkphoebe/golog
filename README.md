# Golog
Golog is a lightweight and expandable logger for Golang.

## Features
* Compatible with the standard log library
* Output to console with nicely color
* Write to file and rotate by date, hour and size
* Both output to console and file
* Customizable log header
* Redirect stdout, stderr to log
* Build-in 5 log levels

## Quick start
```
go get -u github.com/thinkphoebe/golog
```
```go
package main

import log "github.com/thinkphoebe/golog"

func main() {
    log.Info("Hello world!")
}
```

## Usage
#### New Logger
The most convenient way is to use the global Logger object. However, you could create new Logger object in some complicate usage.
```go
fmtStr := "%(asctime) [%(levelno)][%(filename):%(lineno)] "
logger, err := log.NewLogger(log.NewRotateWriter("golog.log", log.RotateByHour), log.LevelDebug, fmtStr)
if err != nil {
    // error handling
}
logger.Infof("Hello world, this is [%s]", "golog")
```

#### Output to console
By default, global Logger output to console with nicely colors. Users can call log.Init() to set custom colors or disable colors.

```go
fmtStr := "%(asctime) [%(levelno)][%(filename):%(lineno)] "
w := log.NewConsoleWriter(os.Stderr)
log.Init(w, log.LevelDebug, fmtStr)
log.Warnf("test Warning log with default color")
w.SetBrush("\033[34;1m", log.LevelWarn)
log.Warnf("test Warning log with custom color")
w.SetColored(false)
log.Warnf("test Warning log not colored")
```
![Console colored](https://raw.githubusercontent.com/thinkphoebe/golog/master/console.png)

#### Output to file
Golog can output to file by add RotateWriter. RotateWrite support RotateByNone, RotateByHour, RotateByDay, RotateBySize.
```go
fmtStr := "%(asctime) [%(levelno)][%(filename):%(lineno)] "
w := log.NewRotateWriter("TestRotate.log", log.RotateBySize)
w.SetRotateSize(100 * 1000 * 1000)
log.Init(w, log.LevelDebug, fmtStr)
```

#### Write to multi-output
Golog can write to multi-output simultaneously. You can add output by AddOutput() and remove output by RemoveOutput().
```go
fmtStr := "%(asctime) [%(levelno)][%(filename):%(lineno)] "
wStderr := log.NewConsoleWriter(os.Stderr)
wFile := log.NewRotateWriter("TestMulti.log", RotateByHour)
log.Init(wStderr, log.LevelDebug, fmtStr)
Infof("write to stderr")
log.AddOutput(wFile)
Infof("both write to both stderr and log file")
Infof("remove file output %t", log.RemoveOutput(wStderr))
Infof("only write to log file")
```

#### Customize log header and log level
```go
fmtStr := `%(asctime) [%(levelno)][%(filename):%(lineno)] `
log.Init(log.NewConsoleWriter(os.Stderr), log.LevelDebug, fmtStr)
```

#### Redirect system stdout and stderr to log
Please note that AddRedirect() will modify the value of **os.File you passed to it. So if you want to output logs to stderr, you must add ConsoleWriter before AddRedirect().
```go
fmtStr := "%(asctime) [%(levelno)][%(filename):%(lineno)] "
w := log.NewConsoleWriter(os.Stderr)
log.Init(w, log.LevelDebug, fmtStr)

fmt.Fprint(os.Stdout, "write to stdout before redirect\n")
fmt.Fprint(os.Stderr, "write to stderr before redirect\n")

rstdr := log.AddRedirect(&os.Stderr, log.LevelError, "[stderr]")
rstdo := log.AddRedirect(&os.Stdout, log.LevelInfo, "[stdout]")

Infof("write to log file 1")
fmt.Fprint(os.Stdout, "write to stdout with recirect\n")
fmt.Fprint(os.Stderr, "write to stderr with recirect\n")
Infof("write to log file 2")

log.CancelRedirector(rstdr)
log.CancelRedirector(rstdo)

fmt.Fprint(os.Stdout, "write to stdout after redirect\n")
fmt.Fprint(os.Stderr, "write to stderr after redirect\n")
```
![Console colored](https://raw.githubusercontent.com/thinkphoebe/golog/master/redirect.png)
    
#### Use you own writer
You can implement your own writers in addition to ConsoleWriter and RotateWriter. Just implement IOutput interface defined in log.go and pass it to log.NewLogger() or log.Init().
```go
type IOutput interface {
	Write(msg []byte, level LogLevel)
}
```

#### Migrate from standard log library
To migrate from standard log module, programs only need to modify from
```go
import "log"
```
to
```go
import log "github.com/thinkphoebe/golog"
```