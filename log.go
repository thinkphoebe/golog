// Golog is a lightweight and expandable logger for Golang.
package golog

import (
	"bufio"
	"encoding/json"
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
	function  string
	line      int
	calldepth int
}

type genHeaderFunc func(buf *[]byte, item *logItem)

type headerSession struct {
	name      string
	isCopy    bool
	strCopy   string
	genHeader genHeaderFunc
}

type Json map[string]interface{}

// Output interface of Logger. Users can implement this interface to output to other destinations such as udp.
type IOutput interface {
	Write(msg []byte, level LogLevel)
}

// A Logger represents an active logging object that generates lines of
// output to an IOutput. A Logger can be used simultaneously from
// multiple goroutines; it guarantees to serialize access to the Writer.
type Logger struct {
	mu             sync.Mutex
	outs           []IOutput
	level          LogLevel
	headerSessions []headerSession
}

// Users can redirect an os.File to log by AddRedirect(). AddRedirect() returns a Redirector for CancelRedirect().
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

// Parameter calldepth is used to recover the PC for file name and line no print.
// In general use, you should set calldepth to NormalDepth on call Output() or Outputf().
const NormalDepth = 2

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

// By default, log level is printed as 'D', 'I', 'W', 'E' and 'C', you could modify them by SetLevelTag().
func SetLevelTag(level LogLevel, name string) {
	levels[level] = name
}

func LevelTag(level LogLevel) string {
	return levels[level]
}

func TagLevel(name string) LogLevel {
	for i, v := range levels {
		if v == name {
			return LogLevel(i)
		}
	}
	return LevelDebug
}

func (l *Logger) Level() LogLevel {
	return l.level
}

// If you set log level to LevelWarn, only Warn, Error and Critical logs will be output.
func (l *Logger) SetLevel(level LogLevel) {
	//SetLevel is not locked
	l.level = level
}

// Add an output to write. You can add more than one output.
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

// Redirect an os.File to log, such as os.stderr.
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
			l.Outputf(level, NormalDepth, "%s %s", tag, scanner.Text())
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

// Cancel an os.File redirect added by AddRedirect.
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
		var pc uintptr
		pc, item.filename, item.line, ok = runtime.Caller(item.calldepth + 1)
		if ok {
			var f = runtime.FuncForPC(pc)
			if f != nil {
				item.function = f.Name()
			} else {
				item.function = "???"
			}
		} else {
			item.filename = "???"
			item.line = 0
		}

		i := strings.LastIndexByte(item.filename, '/')
		if i >= 0 {
			item.filename = item.filename[i+1:]
		}

		i = strings.LastIndexByte(item.function, '/')
		if i >= 0 {
			item.function = item.function[i+1:]
		}
		i = strings.IndexByte(item.function, '.')
		if i >= 0 {
			item.function = item.function[i+1:]
		}
	}

	reg, err := regexp.Compile(`%\([\w\:]+\)`)
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

		kw := fmtStr[match[0]+2 : match[1]-1]
		secs := strings.Split(kw, ":")
		name := secs[0]
		if len(secs) > 1 {
			name = secs[1]
		}

		switch secs[0] {
		case "asctime":
			sessions = append(sessions, headerSession{
				name:   name,
				isCopy: false,
				genHeader: func(buf *[]byte, item *logItem) {
					*buf = time.Now().AppendFormat(*buf, "2006-01-02 15:04:05.999")
				},
			})
		case "filename":
			sessions = append(sessions, headerSession{
				name:   name,
				isCopy: false,
				genHeader: func(buf *[]byte, item *logItem) {
					if len(item.filename) == 0 {
						initCaller(item)
					}
					*buf = append(*buf, item.filename...)
				},
			})
		case "function":
			sessions = append(sessions, headerSession{
				name:   name,
				isCopy: false,
				genHeader: func(buf *[]byte, item *logItem) {
					if len(item.function) == 0 {
						initCaller(item)
					}
					*buf = append(*buf, item.function...)
				},
			})
		case "lineno":
			sessions = append(sessions, headerSession{
				name:   name,
				isCopy: false,
				genHeader: func(buf *[]byte, item *logItem) {
					if item.line < 0 {
						initCaller(item)
					}
					*buf = append(*buf, strconv.Itoa(item.line)...)
				},
			})
		case "levelno":
			sessions = append(sessions, headerSession{
				name:   name,
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
		calldepth: calldepth,
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
		l.output(level, calldepth+1, fmt.Sprint(a...))
	}
}

func (l *Logger) Outputf(level LogLevel, calldepth int, format string, a ...interface{}) {
	if level >= l.level {
		l.output(level, calldepth+1, fmt.Sprintf(format, a...))
	}
}

func (l *Logger) Debugf(format string, a ...interface{}) {
	l.Outputf(LevelDebug, NormalDepth+1, format, a...)
}

func (l *Logger) Infof(format string, a ...interface{}) {
	l.Outputf(LevelInfo, NormalDepth+1, format, a...)
}

func (l *Logger) Warnf(format string, a ...interface{}) {
	l.Outputf(LevelWarn, NormalDepth+1, format, a...)
}

func (l *Logger) Errorf(format string, a ...interface{}) {
	l.Outputf(LevelError, NormalDepth+1, format, a...)
}

func (l *Logger) Critical(format string, a ...interface{}) {
	l.Outputf(LevelCritical, NormalDepth+1, format, a...)
}

func (l *Logger) OutputJson(level LogLevel, calldepth int, items Json) {
	if level < l.level {
		return
	}

	item := logItem{
		level:     level,
		calldepth: calldepth,
	}
	for _, s := range l.headerSessions {
		if !s.isCopy {
			var value []byte
			s.genHeader(&value, &item)
			items[s.name] = string(value)
		}
	}

	buf, err := json.Marshal(items)
	if err != nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	for _, w := range l.outs {
		w.Write(buf, level)
		w.Write([]byte("\n"), level)
	}
}

func (l *Logger) DebugJson(items Json) {
	l.OutputJson(LevelDebug, NormalDepth+1, items)
}
func (l *Logger) InfoJson(items Json) {
	l.OutputJson(LevelInfo, NormalDepth+1, items)
}
func (l *Logger) WarnJson(items Json) {
	l.OutputJson(LevelWarn, NormalDepth+1, items)
}
func (l *Logger) ErrorJson(items Json) {
	l.OutputJson(LevelError, NormalDepth+1, items)
}
func (l *Logger) CriticalJson(items Json) {
	l.OutputJson(LevelDebug, NormalDepth+1, items)
}

// ================ the following functions write to the global logger ================

// ConsoleWriter object used by the global logger.
var GConsoleWriter = NewConsoleWriter(os.Stderr)
var std, _ = NewLogger(GConsoleWriter, LevelInfo, "%(asctime) [%(levelno)][%(filename):%(function):%(lineno)] ")

// You can use this method to modify settings of the global logger on program start.
// Since no lock callers should ensure no multi-goroutines access.
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

func Debugf(format string, a ...interface{}) { std.Outputf(LevelDebug, NormalDepth+1, format, a...) }
func Infof(format string, a ...interface{})  { std.Outputf(LevelInfo, NormalDepth+1, format, a...) }
func Warnf(format string, a ...interface{})  { std.Outputf(LevelWarn, NormalDepth+1, format, a...) }
func Errorf(format string, a ...interface{}) { std.Outputf(LevelError, NormalDepth+1, format, a...) }
func Criticalf(format string, a ...interface{}) {
	std.Outputf(LevelCritical, NormalDepth+1, format, a...)
}
