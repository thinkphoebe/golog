package golog

import (
	"fmt"
	"os"
	"syscall"
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
	fp         *os.File
}

// Create a new RotateWriter
func NewRotateWriter(file string, mode RotateMode) *RotateWriter {
	w := &RotateWriter{file: file, rotateMode: mode, rotateSize: defaultRotateSize}
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
	var suffix string

	if w.rotateMode == RotateByDay {
		suffix = time.Now().Format(format_time_day)
	} else if w.rotateMode == RotateByHour {
		suffix = time.Now().Format(format_time_hour)
	} else if w.rotateMode == RotateBySize {
		if w.suffix == "" {
			fi, err := os.Stat(w.file)
			if err == nil {
				stat := fi.Sys().(*syscall.Stat_t)
				ctime := time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec))
				w.suffix = ctime.Format(format_time_size)
				w.writedSize = fi.Size()
			}
		}
		suffix = w.suffix
		if w.writedSize >= w.rotateSize {
			suffix = time.Now().Format(format_time_size)
		}
	} else {
		return nil
	}

	if w.fp == nil || suffix != w.suffix {
		return w.doRotate(suffix)
	}
	return nil
}

func (w *RotateWriter) doRotate(suffix string) error {
	if w.fp != nil {
		w.fp.Close()
	}

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

	f, err := os.OpenFile(w.file, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "RotateWriter open log file error [%v]\n", err)
		return err
	}
	w.fp = f
	w.suffix = suffix
	return nil
}
