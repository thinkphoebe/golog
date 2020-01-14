package golog

import (
	"fmt"
	"os"
	"time"
)

type RotateMode int

const (
	RotateNone RotateMode = iota
	RotateByHour
	RotateByDay
	RotateBySize
)

const (
	format_time_day  = "2006-01-02"
	format_time_hour = "2006-01-02-15"
	format_time_size = "2006-01-02.150405.999999"
)

const defaultRotateSize = 100 * 1000 * 1000 //100M

// Write logs to file, support rotate by day, hour, size
type RotateWriter struct {
	file       string
	rotateMode RotateMode
	rotateSize int64
	writedSize int64
	suffix     string
	rotateFlag int
	fp         *os.File
}

// Create a new RotateWriter
func NewRotateWriter(file string, mode RotateMode) *RotateWriter {
	w := &RotateWriter{file: file, rotateMode: mode, rotateSize: defaultRotateSize, rotateFlag: -1}
	err := w.rotate()
	if err != nil {
		return nil
	}
	return w
}

// No lock, callers lock if necessary
func (w *RotateWriter) Write(msg []byte, level LogLevel) {
	err := w.rotate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "RotateWriter rotate error [%v]\n", err)
		return
	}
	w.fp.Write(msg)
	w.writedSize += int64(len(msg))
}

// Used by RotateBySize
func (w *RotateWriter) SetRotateSize(size int64) {
	w.rotateSize = size
}

func (w *RotateWriter) rotate() error {
	suffix := ""
	rotate := false
	t := time.Now()

	// on first write
	if w.fp == nil {
		rotate = true
		info, err := os.Stat(w.file)
		if err == nil {
			if w.rotateMode == RotateByDay && info.ModTime().Day() != t.Day() {
				w.suffix = info.ModTime().Format(format_time_day)
			} else if w.rotateMode == RotateByHour && info.ModTime().Hour() != t.Hour() {
				w.suffix = info.ModTime().Format(format_time_hour)
			} else if w.rotateMode == RotateBySize {
				w.writedSize = info.Size()
			}
		}
	}

	if w.rotateMode == RotateByDay && w.rotateFlag != t.Day() {
		rotate = true
		w.rotateFlag = t.Day()
		suffix = t.Format(format_time_day)
	} else if w.rotateMode == RotateByHour && w.rotateFlag != t.Hour() {
		rotate = true
		w.rotateFlag = t.Hour()
		suffix = t.Format(format_time_hour)
	} else if w.rotateMode == RotateBySize && w.writedSize > w.rotateSize {
		rotate = true
		w.writedSize = 0
		// ATTENTION use current time as rotated file name
		w.suffix = t.Format(format_time_size)
	}

	if rotate {
		return w.doRotate(suffix)
	}
	return nil
}

func (w *RotateWriter) doRotate(suffix string) error {
	if w.suffix != "" {
		info, err := os.Stat(w.file)
		if err == nil && !info.IsDir() {
			lastFileName := w.file + "." + w.suffix
			err := os.Rename(w.file, lastFileName)
			if err != nil {
				return err
			}
		}
	}

	f, err := os.OpenFile(w.file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RotateWriter open log file error [%v]\n", err)
		return err
	}

	if w.fp != nil {
		w.fp.Close()
	}
	w.fp = f
	w.suffix = suffix
	return nil
}
