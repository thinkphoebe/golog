package golog

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type logItem struct {
	level     LogLevel
	filename  string
	line      int
	calldepth int
}

type genHeaderFunc func(buf *[]byte, item *logItem)

type headerSession struct {
	isCopy    bool
	strCopy   string
	genHeader genHeaderFunc
}

type IOutput interface {
	Write(msg []byte, level LogLevel)
}

type Logger struct {
	mu             sync.Mutex
	outs           []IOutput
	level          LogLevel
	headerSessions []headerSession
}

type Redirector struct {
	tag     string
	oldAddr **os.File
	old     *os.File
	pipe    *os.File
}

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelCritical
)
const CallDepth = 2

var levels = [...]string{int(LevelDebug): "D", int(LevelInfo): "I", int(LevelWarn): "W", int(LevelError): "E", int(LevelCritical): "C"}

func NewLogger(out IOutput, level LogLevel, fmtStr string) (*Logger, error) {
	l := &Logger{
		level: level,
		outs:  []IOutput{out},
	}
	err := l.setHeaderFormat(fmtStr)
	if err != nil {
		l = nil
	}
	return l, err
}

func SetLevelName(level LogLevel, name string) {
	levels[level] = name
}

func LevelName(level LogLevel) string {
	return levels[level]
}

func (l *Logger) Level() LogLevel {
	return l.level
}

//SetLevel is not locked
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) AddOutput(w IOutput) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.outs = append(l.outs, w)
}

func (l *Logger) RemoveOutput(w IOutput) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, v := range l.outs {
		if v == w {
			l.outs = append(l.outs[:i], l.outs[i+1:]...)
			return true
		}
	}
	return false
}

func (l *Logger) AddRedirect(file **os.File, level LogLevel, tag string) *Redirector {
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil
	}
	old := *file
	*file = pw

	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			l.Outputf(level, CallDepth, "%s %s", tag, scanner.Text())
		}
		l.Warnf("read redirector pipe %s complete", tag)
	}()

	r := Redirector{
		tag:     tag,
		oldAddr: file,
		old:     old,
		pipe:    pw,
	}
	return &r
}

func (l *Logger) CancelRedirect(r *Redirector) {
	l.Warnf("close redirector pipe %s", r.tag)
	r.pipe.Close()
	*r.oldAddr = r.old
}

// Please ATTENTION that header format is not designed to be modify after logger created.
// So users can only set header format on NewLogger() called. No lock when Logger.headerSessions used.
func (l *Logger) setHeaderFormat(fmtStr string) error {
	initCaller := func(item *logItem) {
		var ok bool
		_, item.filename, item.line, ok = runtime.Caller(item.calldepth + 1)
		if !ok {
			item.filename = "???"
			item.line = 0
		}
	}

	reg, err := regexp.Compile(`%\([\w]+\)`)
	if err != nil {
		return err
	}

	matches := reg.FindAllStringIndex(fmtStr, -1)
	sessions := make([]headerSession, 0, len(matches))
	beg := 0
	end := 0
	for _, match := range matches {
		end = match[0]
		if end > beg {
			sessions = append(sessions, headerSession{
				isCopy:  true,
				strCopy: fmtStr[beg:end],
			})
		}

		kw := fmtStr[match[0]:match[1]]
		switch kw {
		case "%(asctime)":
			sessions = append(sessions, headerSession{
				isCopy: false,
				genHeader: func(buf *[]byte, item *logItem) {
					*buf = time.Now().AppendFormat(*buf, "2006-01-02 15:04:05.999")
				},
			})
		case "%(filename)":
			sessions = append(sessions, headerSession{
				isCopy: false,
				genHeader: func(buf *[]byte, item *logItem) {
					if len(item.filename) == 0 {
						initCaller(item)
					}
					i := strings.LastIndexByte(item.filename, '/')
					if i >= 0 {
						item.filename = item.filename[i+1:]
					}
					*buf = append(*buf, item.filename...)
				},
			})
		case "%(lineno)":
			sessions = append(sessions, headerSession{
				isCopy: false,
				genHeader: func(buf *[]byte, item *logItem) {
					if item.line < 0 {
						initCaller(item)
					}
					*buf = append(*buf, strconv.Itoa(item.line)...)
				},
			})
		case "%(levelno)":
			sessions = append(sessions, headerSession{
				isCopy: false,
				genHeader: func(buf *[]byte, item *logItem) {
					*buf = append(*buf, levels[item.level]...)
				},
			})
		default:
			err = fmt.Errorf("unknown keyword [%s]", kw)
			return err
		}

		beg = match[1]
	}

	if beg < len(fmtStr) {
		sessions = append(sessions, headerSession{
			isCopy:  true,
			strCopy: fmtStr[beg:],
		})
	}
	l.headerSessions = sessions
	return nil
}

func (l *Logger) output(level LogLevel, calldepth int, s string) {
	var buf []byte

	item := logItem{
		level:     level,
		calldepth: calldepth + 1,
	}
	for _, s := range l.headerSessions {
		if s.isCopy {
			buf = append(buf, s.strCopy...)
		} else {
			s.genHeader(&buf, &item)
		}
	}

	buf = append(buf, s...)
	if len(s) == 0 || s[len(s)-1] != '\n' {
		buf = append(buf, '\n')
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	for _, w := range l.outs {
		w.Write(buf, level)
	}
}

func (l *Logger) Output(level LogLevel, calldepth int, a ...interface{}) {
	if level >= l.level {
		l.output(level, calldepth, fmt.Sprint(a...))
	}
}

func (l *Logger) Outputf(level LogLevel, calldepth int, format string, a ...interface{}) {
	if level >= l.level {
		l.output(level, calldepth, fmt.Sprintf(format, a...))
	}
}

func (l *Logger) Debugf(format string, a ...interface{}) {
	l.Outputf(LevelDebug, CallDepth+1, format, a...)
}

func (l *Logger) Infof(format string, a ...interface{}) {
	l.Outputf(LevelInfo, CallDepth+1, format, a...)
}

func (l *Logger) Warnf(format string, a ...interface{}) {
	l.Outputf(LevelWarn, CallDepth+1, format, a...)
}

func (l *Logger) Errorf(format string, a ...interface{}) {
	l.Outputf(LevelError, CallDepth+1, format, a...)
}

func (l *Logger) Critical(format string, a ...interface{}) {
	l.Outputf(LevelCritical, CallDepth+1, format, a...)
}

// ================ the following functions write to the default logger ================

var std, _ = NewLogger(NewConsoleWriter(os.Stderr), LevelInfo, "%(asctime) [%(levelno)][%(filename):%(lineno)] ")

// This function is designed to modify the settings of default logger on program start.
// Since no lock when modifying variable "std", callers should ensure no multi-goroutines access.
func Init(out IOutput, level LogLevel, fmtStr string) error {
	l, err := NewLogger(out, level, fmtStr)
	if err == nil {
		std = l
	}
	return err
}

func Level() LogLevel         { return std.Level() }
func SetLevel(level LogLevel) { std.SetLevel(level) }

func AddOutput(w IOutput)         { std.AddOutput(w) }
func RemoveOutput(w IOutput) bool { return std.RemoveOutput(w) }

func AddRedirect(file **os.File, level LogLevel, tag string) *Redirector {
	return std.AddRedirect(file, level, tag)
}
func CancelRedirector(r *Redirector) { std.CancelRedirect(r) }

func Output(level LogLevel, calldepth int, a ...interface{}) { std.Output(level, calldepth+1, a...) }
func Outputf(level LogLevel, calldepth int, format string, a ...interface{}) {
	std.Outputf(level, calldepth+1, format, a...)
}

func Debugf(format string, a ...interface{})    { std.Outputf(LevelDebug, CallDepth+1, format, a...) }
func Infof(format string, a ...interface{})     { std.Outputf(LevelInfo, CallDepth+1, format, a...) }
func Warnf(format string, a ...interface{})     { std.Outputf(LevelWarn, CallDepth+1, format, a...) }
func Errorf(format string, a ...interface{})    { std.Outputf(LevelError, CallDepth+1, format, a...) }
func Criticalf(format string, a ...interface{}) { std.Outputf(LevelCritical, CallDepth+1, format, a...) }
