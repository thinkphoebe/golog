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

type LogLevel int

type logItem struct {
	level     LogLevel
	filename  string
	function  string
	line      int
	calldepth int
}

type genHeaderFunc func(buf *[]byte, item *logItem)

type headerSession struct {
	name      string // used as key of json output
	isCopy    bool
	strCopy   string
	genHeader genHeaderFunc
}

type outWriter struct {
	writer IOutput
	chIn   chan *outItem
}

type outItem struct {
	msg   []byte
	level LogLevel
}

type cmdItem struct {
	cmd   int         // 0 -> add outWriter, 1 -> remove outWriter
	param interface{} // 0, 1 -> IOutput
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
	outs           []outWriter
	level          LogLevel
	async          bool
	chOut          chan *outItem
	chCmd          chan *cmdItem
	headerSessions []headerSession
}

// Users can redirect an os.File to log by AddRedirect(). AddRedirect() returns a Redirector for CancelRedirect().
type Redirector struct {
	tag     string
	oldAddr **os.File
	old     *os.File
	pipe    *os.File
}

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
const AsyncBuffer = 1000
const OutputBuffer = 10000

var levels = [...]string{int(LevelDebug): "D", int(LevelInfo): "I", int(LevelWarn): "W", int(LevelError): "E", int(LevelCritical): "C"}

func NewLogger(out IOutput, level LogLevel, fmtStr string, async bool) (*Logger, error) {
	l := &Logger{
		level: level,
		async: async,
	}
	err := l.setHeaderFormat(fmtStr)
	if err != nil {
		l = nil
	} else {
		if async {
			l.chOut = make(chan *outItem, AsyncBuffer)
			l.chCmd = make(chan *cmdItem, 100)
			go l.copyRoutine()
		}

		l.AddOutput(out)
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

// copy logs from chOut to chIn of each outWriter
func (l *Logger) copyRoutine() {
	for {
		select {
		case item := <-l.chOut:
			for _, out := range l.outs {
				out.chIn <- item
			}
		case cmd, ok := <-l.chCmd:
			if !ok {
				break
			}
			if cmd.cmd == 0 {
				l.addOutput(cmd.param.(IOutput))
			} else if cmd.cmd == 1 {
				l.removeOutput(cmd.param.(IOutput))
			}
		}
	}
}

func (l *Logger) outputRoutine(out *outWriter) {
	for {
		item, ok := <-out.chIn
		if !ok {
			break
		}
		if len(out.chIn) > OutputBuffer*3/5 && item.level <= LevelDebug ||
			len(out.chIn) > OutputBuffer*4/5 && item.level <= LevelInfo {
			continue
		}
		out.writer.Write(item.msg, item.level)
	}
}

func (l *Logger) addOutput(w IOutput) {
	out := outWriter{writer: w}
	if l.async {
		out.chIn = make(chan *outItem, OutputBuffer)
		go l.outputRoutine(&out)
	}
	l.outs = append(l.outs, out)
}

// Add an outWriter to write. You can add more than one outWriter.
func (l *Logger) AddOutput(w IOutput) {
	if l.async {
		l.chCmd <- &cmdItem{cmd: 0, param: w}
	} else {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.addOutput(w)
	}
}

func (l *Logger) removeOutput(w IOutput) {
	for i, v := range l.outs {
		if v.writer == w {
			l.outs = append(l.outs[:i], l.outs[i+1:]...)
			if l.async {
				close(v.chIn)
			}
		}
	}
}

func (l *Logger) RemoveOutput(w IOutput) {
	if l.async {
		l.chCmd <- &cmdItem{cmd: 1, param: w}
	} else {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.removeOutput(w)
	}
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

// An async logger should be Close() to avoid resource leak.
// Before Close() any redirect should be canceled.
func (l *Logger) Close() {
	if l.async {
		close(l.chCmd)
		close(l.chOut)
		for _, v := range l.outs {
			close(v.chIn)
		}
	}
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

	appendInt := func(b []byte, x int, width int) []byte {
		var buf [20]byte
		i := len(buf)
		for x >= 10 {
			i--
			q := x / 10
			buf[i] = byte('0' + x - q*10)
			x = q
		}
		i--
		buf[i] = byte('0' + x)
		for w := len(buf) - i; w < width; w++ {
			b = append(b, '0')
		}
		return append(b, buf[i:]...)
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
					//*buf = time.Now().AppendFormat(*buf, "2006-01-02 15:04:05.999")
					now := time.Now()
					year, mon, day := now.Date()
					hour, min, sec := now.Clock()
					nsec := now.Nanosecond()
					*buf = appendInt(*buf, year, 4)
					*buf = append(*buf, []byte("-")...)
					*buf = appendInt(*buf, int(mon), 2)
					*buf = append(*buf, []byte("-")...)
					*buf = appendInt(*buf, day, 2)
					*buf = append(*buf, []byte(" ")...)
					*buf = appendInt(*buf, hour, 2)
					*buf = append(*buf, []byte(":")...)
					*buf = appendInt(*buf, min, 2)
					*buf = append(*buf, []byte(":")...)
					*buf = appendInt(*buf, sec, 2)
					*buf = append(*buf, []byte(".")...)
					*buf = appendInt(*buf, nsec/1000000, 3)
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

func (l *Logger) write(msg []byte, level LogLevel) {
	if l.async {
		l.chOut <- &outItem{msg: msg, level: level}
	} else {
		l.mu.Lock()
		defer l.mu.Unlock()
		for _, w := range l.outs {
			w.writer.Write(msg, level)
		}
	}
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
	l.write(buf, level)
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

func (l *Logger) Logf(level LogLevel, format string, a ...interface{}) {
	l.Outputf(level, NormalDepth+1, format, a...)
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

func (l *Logger) Criticalf(format string, a ...interface{}) {
	l.Outputf(LevelCritical, NormalDepth+1, format, a...)
}

func (l *Logger) OutputJson(level LogLevel, calldepth int, items Json) {
	if level < l.level {
		return
	}

	var buf []byte

	item := logItem{
		level:     level,
		calldepth: calldepth,
	}
	for index, s := range l.headerSessions {
		if s.isCopy {
			// ATTENTION if first header session is string const, it will be added to header of json string
			if index == 0 {
				buf = append(buf, s.strCopy...)
			}
		} else {
			var value []byte
			s.genHeader(&value, &item)
			items[s.name] = string(value)
		}
	}

	bufJson, err := json.Marshal(items)
	if err != nil {
		return
	}
	buf = append(buf, bufJson...)
	buf = append(buf, '\n')
	l.write(buf, level)
}

func (l *Logger) LogJson(level LogLevel, items Json) {
	l.OutputJson(level, NormalDepth+1, items)
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
var std, _ = NewLogger(GConsoleWriter, LevelInfo, "%(asctime) [%(levelno)][%(filename):%(function):%(lineno)] ", false)

// You can use this method to modify settings of the global logger on program start.
// Since no lock callers should ensure no multi-goroutines access.
func Init(out IOutput, level LogLevel, fmtStr string, async bool) error {
	l, err := NewLogger(out, level, fmtStr, async)
	if err == nil {
		std = l
	}
	return err
}

func Level() LogLevel         { return std.Level() }
func SetLevel(level LogLevel) { std.SetLevel(level) }

func AddOutput(w IOutput)    { std.AddOutput(w) }
func RemoveOutput(w IOutput) { std.RemoveOutput(w) }

func AddRedirect(file **os.File, level LogLevel, tag string) *Redirector {
	return std.AddRedirect(file, level, tag)
}
func CancelRedirector(r *Redirector) { std.CancelRedirect(r) }

func Output(level LogLevel, calldepth int, a ...interface{}) { std.Output(level, calldepth+1, a...) }
func Outputf(level LogLevel, calldepth int, format string, a ...interface{}) {
	std.Outputf(level, calldepth+1, format, a...)
}

func Logf(level LogLevel, format string, a ...interface{}) {
	std.Outputf(level, NormalDepth+1, format, a...)
}
func Debugf(format string, a ...interface{}) { std.Outputf(LevelDebug, NormalDepth+1, format, a...) }
func Infof(format string, a ...interface{})  { std.Outputf(LevelInfo, NormalDepth+1, format, a...) }
func Warnf(format string, a ...interface{})  { std.Outputf(LevelWarn, NormalDepth+1, format, a...) }
func Errorf(format string, a ...interface{}) { std.Outputf(LevelError, NormalDepth+1, format, a...) }
func Criticalf(format string, a ...interface{}) {
	std.Outputf(LevelCritical, NormalDepth+1, format, a...)
}
