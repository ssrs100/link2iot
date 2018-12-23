package logs

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

type consoleWriter struct {
	sync.Mutex
	writer   io.Writer
	LogLevel string `json:"logLevel"`
	Level    int
}

func (cw *consoleWriter) println(when time.Time, msg string) {
	cw.Lock()
	h, _, _ := formatTimeHeader(when)
	h = h + "[" + getGID() + "]"
	buf := []byte(h)
	cw.writer.Write(append(append(buf, msg...), '\n'))
	cw.Unlock()
}

func NewConsole() LoggerItf {
	cw := &consoleWriter{
		writer:   os.Stdout,
		LogLevel: "DEBUG",
		Level:    DEBUG,
	}
	return cw
}

func (c *consoleWriter) Init(jsonConfig string) error {
	if len(jsonConfig) == 0 {
		return nil
	}
	err := json.Unmarshal([]byte(jsonConfig), c)
	c.Level = transLogLevel(c.LogLevel)
	return err
}

func (c *consoleWriter) WriteMsg(when time.Time, msg string, level int) error {
	if level < c.Level {
		return nil
	}
	c.println(when, msg)
	return nil
}

func (c *consoleWriter) Destroy() {

}

func (c *consoleWriter) Flush() {

}

func init() {
	Register("console", NewConsole)
}
