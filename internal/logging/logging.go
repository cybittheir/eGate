package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type dailyFileWriter struct {
	dir     string
	silent  bool
	curDate string
	file    *os.File
	console io.Writer
}

func newDailyFileWriter(dir string, silent bool) (*dailyFileWriter, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("не удалось создать папку %s: %w", dir, err)
	}
	w := &dailyFileWriter{
		dir:     dir,
		silent:  silent,
		console: os.Stdout,
	}
	if err := w.rotateIfNeeded(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *dailyFileWriter) rotateIfNeeded() error {
	today := time.Now().Format("20060102")
	if w.file != nil && today == w.curDate {
		return nil
	}
	if w.file != nil {
		_ = w.file.Close()
	}
	path := filepath.Join(w.dir, today+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("не удалось открыть лог‑файл %s: %w", path, err)
	}
	w.file = f
	w.curDate = today
	return nil
}

func (w *dailyFileWriter) Write(p []byte) (int, error) {
	if err := w.rotateIfNeeded(); err != nil {
		return 0, err
	}
	if w.silent {
		return w.file.Write(p)
	}
	return io.MultiWriter(w.file, w.console).Write(p)
}

func (w *dailyFileWriter) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Setup настраивает стандартный log в соответствии с флагами.
func Setup(saveToFile, silent bool) (io.Closer, error) {
	if !saveToFile && silent {
		log.SetOutput(io.Discard)
		return nil, nil
	}
	if !saveToFile && !silent {
		log.SetOutput(os.Stdout)
		return nil, nil
	}

	writer, err := newDailyFileWriter("Logs", silent)
	if err != nil {
		return nil, err
	}
	log.SetOutput(writer)
	return writer, nil
}
