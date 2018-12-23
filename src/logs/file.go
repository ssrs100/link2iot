package logs

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	LOG_PATTERN string = "2006-01-02-15-04-05"

	MB = 1024 * 1024

	ONE_DAY_SECONDS = 60 * 60 * 24
)

type fileLogWriter struct {
	//写日志文件用来控制顺序原子性
	sync.Mutex

	//写入的文件
	Filename   string `json:"filename"`
	fileWriter *os.File

	//超过最大行翻转日志文件
	MaxLines         int `json:"maxlines"`
	maxLinesCurLines int

	//超过最大容量翻转日志文件
	MaxSize        int `json:"maxsize"`
	maxSizeCurSize int

	//按时间翻转
	Daily         bool  `json:"daily"`
	MaxDays       int64 `json:"maxdays"`
	dailyOpenDate int

	Rotate bool `json:"rotate"`

	//string类型与外部配置文件对接
	LogLevel string `json:"logLevel"`

	//int类型，用来比较日志级别
	Level int

	//生成的日志文件权限
	Perm os.FileMode `json:"perm"`

	//日志文件名和后缀
	fileNameOnly string

	//日志总大小限制：mb
	MaxTotalSize int64 `json:"maxTotalSize"`
}

// newFileWriter 返回Logger 的一个接口实例
func newFileWriter() LoggerItf {
	w := &fileLogWriter{
		Filename: "",
		MaxLines: 1000000,
		//256 MB
		MaxSize:      256,
		MaxTotalSize: 1024,
		Daily:        true,
		MaxDays:      7,
		Rotate:       true,
		Level:        DEBUG,
		Perm:         0660,
	}
	return w
}

// 初始化文件日志实例
// 参数形式:
//	{
//	"filename":"test.log",
//	"maxLines":10000,
//	"maxsize":256,
//	"daily":true,
//	"maxDays":15,
//	"rotate":true,
//  	"perm":0600
//	}
func (w *fileLogWriter) Init(jsonConfig string) error {
	err := json.Unmarshal([]byte(jsonConfig), w)
	if err != nil {
		return err
	}
	if len(w.Filename) == 0 {
		return errors.New("jsonconfig must have filename")
	}
	w.Level = transLogLevel(w.LogLevel)
	suffix := filepath.Ext(w.Filename)

	w.fileNameOnly = strings.TrimSuffix(filepath.Base(w.Filename), suffix)

	//转换成字节数
	w.MaxSize = w.MaxSize * MB
	w.MaxTotalSize = w.MaxTotalSize * MB
	err = w.startLogger()
	return err
}

// start file logger. create log file and set to locker-inside file writer.
func (w *fileLogWriter) startLogger() error {
	file, err := w.createLogFile()
	if err != nil {
		return err
	}
	if w.fileWriter != nil {
		w.fileWriter.Close()
	}
	w.fileWriter = file
	return w.initFd()
}

func (w *fileLogWriter) needRotate(size int, day int) bool {
	return (w.MaxLines > 0 && w.maxLinesCurLines >= w.MaxLines) ||
		(w.MaxSize > 0 && w.maxSizeCurSize+size >= w.MaxSize) ||
		(w.Daily && day != w.dailyOpenDate)

}

// 将日志写入文件
func (w *fileLogWriter) WriteMsg(when time.Time, msg string, level int) error {
	if level < w.Level {
		return nil
	}
	h, d, errTime := formatTimeHeader(when)
	if errTime != nil {
		return errTime
	}

	if w.Level == DEBUG {
		msg = h + "[" + getGID() + "]" + msg + "\n"
	} else {
		msg = h + msg + "\n"
	}

	if w.Rotate {
		if w.needRotate(len(msg), d) {
			w.Lock()
			if w.needRotate(len(msg), d) {
				if err := w.doRotate(when); err != nil {
					fmt.Fprintf(os.Stderr, "FileLogWriter(%q): %s\n", w.Filename, err)
				}
			}
			w.Unlock()
		}
	}

	w.Lock()
	_, err := w.fileWriter.Write([]byte(msg))
	if err == nil {
		w.maxLinesCurLines++
		w.maxSizeCurSize += len(msg)
	}
	w.Unlock()
	return err
}

func (w *fileLogWriter) createLogFile() (*os.File, error) {
	fd, err := os.OpenFile(w.Filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, w.Perm)
	return fd, err
}

func (w *fileLogWriter) initFd() error {
	fd := w.fileWriter
	fInfo, err := fd.Stat()
	if err != nil {
		return fmt.Errorf("get stat err: %s\n", err)
	}
	w.maxSizeCurSize = int(fInfo.Size())
	w.dailyOpenDate = time.Now().Day()
	w.maxLinesCurLines = 0
	if fInfo.Size() > 0 {
		count, err := w.lines()
		if err != nil {
			return err
		}
		w.maxLinesCurLines = count
	}
	return nil
}

func (w *fileLogWriter) lines() (int, error) {
	fd, err := os.Open(w.Filename)
	if err != nil {
		return 0, err
	}
	defer fd.Close()

	//一次最多读32K
	buf := make([]byte, 32768)
	count := 0
	lineSep := []byte{'\n'}

	for {
		c, err := fd.Read(buf)
		if err != nil && err != io.EOF {
			return count, err
		}

		count += bytes.Count(buf[:c], lineSep)

		if err == io.EOF {
			break
		}
	}

	return count, nil
}

// 老文件重新命名，重新启动日志文件
func (w *fileLogWriter) doRotate(logTime time.Time) error {
	_, err := os.Lstat(w.Filename)
	if err != nil {
		return err
	}
	// 设置文件名
	zipName := fmt.Sprintf("%s.%s.zip", w.fileNameOnly, logTime.Format(LOG_PATTERN))
	logName := fmt.Sprintf("%s.%s.log", w.fileNameOnly, logTime.Format(LOG_PATTERN))
	// 在改名前将文件关闭
	w.fileWriter.Close()

	dir := filepath.Dir(w.Filename)
	if dir != "." {
		zipName = filepath.Join(dir, zipName)
		logName = filepath.Join(dir, logName)
	}
	renameErr := os.Rename(w.Filename, logName)
	// 重新启动日志文件
	startLoggerErr := w.startLogger()
	go w.compressAndClean(logName, zipName)

	if startLoggerErr != nil {
		return fmt.Errorf("Rotate StartLogger: %s\n", startLoggerErr)
	}
	if renameErr != nil {
		return fmt.Errorf("Rotate: %s\n", renameErr)
	}
	return nil

}

//压缩和清理日志
func (w *fileLogWriter) compressAndClean(logName, zipName string) {
	w.compressFile(logName, zipName)
	w.deleteOldLog()
}

func (w *fileLogWriter) compressFile(source, target string) error {
	reader, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() {
		reader.Close()
		//关闭后删除原文件
		os.Remove(source)
	}()

	filename := filepath.Base(source)
	writer, err := os.Create(target)
	if err != nil {
		return err
	}
	defer writer.Close()

	archiver := gzip.NewWriter(writer)
	archiver.Name = filename
	defer archiver.Close()

	_, err = io.Copy(archiver, reader)
	return err
}

func (w *fileLogWriter) deleteOldLog() {
	dir := filepath.Dir(w.Filename)
	var totalSize int64
	fileTimeMap := make(map[string]os.FileInfo)
	timeKeys := make([]string, 0)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) (returnErr error) {
		defer func() {
			//如果出现异常，捕获后忽略继续执行
			if r := recover(); r != nil {
				fmt.Fprintf(os.Stderr, "Unable to delete old log '%s', error: %v\n", path, r)
			}
		}()

		//目录忽略
		if info.IsDir() {
			return
		}
		//非配置的日志文件忽略
		if !strings.HasPrefix(filepath.Base(path), w.fileNameOnly) {
			return
		}
		//超过最大保存天数，删除该日志文件
		if info.ModTime().Unix() < (time.Now().Unix() - ONE_DAY_SECONDS*w.MaxDays) {
			os.Remove(path)
			return
		}
		timeKey := info.ModTime().Format(LOG_PATTERN)
		fileTimeMap[timeKey] = info
		timeKeys = append(timeKeys, timeKey)
		totalSize += info.Size()
		return
	})

	//总大小未超过配置的日志最大空间大小
	if totalSize-w.MaxTotalSize <= 0 {
		return
	}
	sort.Strings(timeKeys)
	var delSize int64
	for _, v := range timeKeys {
		if delSize > totalSize-w.MaxTotalSize {
			break
		}
		delSize += fileTimeMap[v].Size()
		os.Remove(filepath.Join(dir, fileTimeMap[v].Name()))
	}
}

// 实现关闭文件接口
func (w *fileLogWriter) Destroy() {
	w.fileWriter.Close()
}

// 将数据刷新到磁盘
func (w *fileLogWriter) Flush() {
	w.fileWriter.Sync()
}

func init() {
	Register("file", newFileWriter)
}
