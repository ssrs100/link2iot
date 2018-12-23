package logs

import (
	"fmt"
	"conf"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

//常用的日志级别定义
const (
	DEBUG = iota
	INFO
	WARN
	ERROR
	FATAL
)

type logMsg struct {
	level int
	msg   string
	time  time.Time
}

//logger基本的数据结构
type Logger struct {
	level               int
	lock                sync.Mutex
	msgChan             chan *logMsg
	signalChan          chan string
	outputs             []*nameLogger
	wg                  sync.WaitGroup
	enableFuncCallDepth bool
	loggerFuncCallDepth int
	asynchronous        bool
	//logger 实例
	loggerInstance *Logger
}

//定义LOGGER基本的接口
type LoggerItf interface {
	Init(config string) error
	WriteMsg(t time.Time, msg string, level int) error
	Destroy()
	Flush()
}

type loggerType func() LoggerItf

type nameLogger struct {
	LoggerItf
	name string
}

//注册的日志插件容器
var adapters = make(map[string]loggerType)

var logMsgPool *sync.Pool

//logger 实例
var loggerInstance *Logger

//实例初始化锁
var instanceLock sync.Mutex

//外部实现的日志插件通过该接口注册进来
func Register(name string, log loggerType) {
	if log == nil {
		panic("logs:Register is nil, name:" + name)
	}

	if _, ok := adapters[name]; ok {
		fmt.Println("logs:Register is dumplicated, name:", name)
		return
	}
	adapters[name] = log
}

//创建LOGGER实例，默认为DEBUG级别
func newLogger() *Logger {
	logger := Logger{}
	basedir := os.Getenv("APP_BASE_DIR")
	logFile := filepath.Join(basedir, "conf", "log4g.json")
	config := conf.LoadFile(logFile)
	pattern := config.GetString("pattern")
	param := config.GetJson()
	if err := logger.setLogger(pattern, param); err != nil {
		fmt.Println("set log failed. err:", err)
	}
	logger.EnableFuncCallDepth(true)
	logger.SetLogFuncCallDepth(2)

	logger.asynchronous = false
	logger.level = DEBUG
	logger.msgChan = make(chan *logMsg, 10000)
	logger.signalChan = make(chan string, 1)
	return &logger
}

func GetLogger() *Logger {
	if loggerInstance == nil {
		instanceLock.Lock()
		if loggerInstance == nil {
			loggerInstance = newLogger()
		}
		instanceLock.Unlock()

	}
	return loggerInstance
}

func (log *Logger) setLogger(logType string, config string) error {
	log.lock.Lock()
	defer log.lock.Unlock()

	for _, l := range log.outputs {
		if l.name == logType {
			return fmt.Errorf("logs:dumplicate log type is being set, logType %q", logType)
		}
	}

	logFun, ok := adapters[logType]
	if !ok {
		return fmt.Errorf("logs: unkown logType %q", logType)
	}

	logInstance := logFun()
	err := logInstance.Init(config)
	if err != nil {
		return fmt.Errorf("logs: SetLogger failed. error: %q", err.Error())
	}
	log.outputs = append(log.outputs, &nameLogger{name: logType, LoggerItf: logInstance})
	return nil
}

// 开启异步日志功能
func (log *Logger) Async() *Logger {
	log.asynchronous = true
	logMsgPool = &sync.Pool{
		New: func() interface{} {
			return &logMsg{}
		},
	}
	log.wg.Add(1)
	go log.startLogger()
	return log
}

func (log *Logger) DelLogger(adapterName string) error {
	log.lock.Lock()
	defer log.lock.Unlock()
	outputs := []*nameLogger{}
	for _, lg := range log.outputs {
		if lg.name == adapterName {
			lg.Destroy()
		} else {
			outputs = append(outputs, lg)
		}
	}
	if len(outputs) == len(log.outputs) {
		return fmt.Errorf("logs: unknown adaptername %q (forgotten Register?)", adapterName)
	}
	log.outputs = outputs
	return nil
}

func (log *Logger) writeToLoggers(t time.Time, msg string, level int) {
	for _, l := range log.outputs {
		err := l.WriteMsg(t, msg, level)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to WriteMsg to adapter:%v,error:%v\n", l.name, err)
		}
	}
}

func (log *Logger) writeMsg(logLevel int, msg string) error {
	when := time.Now()
	if log.enableFuncCallDepth {
		_, file, line, ok := runtime.Caller(log.loggerFuncCallDepth)
		if !ok {
			file = "???"
			line = 0
		}
		_, filename := path.Split(file)
		msg = "[" + filename + ":" + strconv.FormatInt(int64(line), 10) + "]" + msg
	}
	if log.asynchronous {
		lm := logMsgPool.Get().(*logMsg)
		lm.level = logLevel
		lm.msg = msg
		lm.time = when
		log.msgChan <- lm
	} else {
		log.writeToLoggers(when, msg, logLevel)
	}
	return nil
}

func (log *Logger) SetLevel(l int) {
	log.level = l
}

func (log *Logger) SetLogFuncCallDepth(d int) {
	log.loggerFuncCallDepth = d
}

func (log *Logger) GetLogFuncCallDepth() int {
	return log.loggerFuncCallDepth
}

func (log *Logger) EnableFuncCallDepth(b bool) {
	log.enableFuncCallDepth = b
}

func (log *Logger) startLogger() {
	end := false
	for {
		select {
		case msg := <-log.msgChan:
			log.writeToLoggers(msg.time, msg.msg, msg.level)
			logMsgPool.Put(msg)
		case sig := <-log.signalChan:
			log.flush()
			if sig == "close" {
				for _, l := range log.outputs {
					l.Destroy()
				}
				log.outputs = nil
				end = true
			}
			log.wg.Done()
		}
		if end {
			break
		}
	}
}

func (log *Logger) flush() {
	for {
		if len(log.msgChan) > 0 {
			msg := <-log.msgChan
			log.writeToLoggers(msg.time, msg.msg, msg.level)
			logMsgPool.Put(msg)
			continue
		}
		break
	}
	for _, l := range log.outputs {
		l.Flush()
	}
}

func (log *Logger) Close() {
	if log.asynchronous {
		log.signalChan <- "close"
		log.wg.Wait()
	} else {
		log.flush()
		for _, l := range log.outputs {
			l.Destroy()
		}
		log.outputs = nil
	}
	close(log.msgChan)
	close(log.signalChan)
	loggerInstance = nil
}

func (log *Logger) Flush() {
	if log.asynchronous {
		log.signalChan <- "flush"
		log.wg.Wait()
		log.wg.Add(1)
		return
	}
	log.flush()
}

func (log *Logger) Debug(format string, v ...interface{}) {
	if DEBUG < log.level {
		return
	}
	msg := fmt.Sprintf("[DEBUG] "+format, v...)
	log.writeMsg(DEBUG, msg)
}

func (log *Logger) Info(format string, v ...interface{}) {
	if INFO < log.level {
		return
	}
	msg := fmt.Sprintf("[INFO] "+format, v...)
	log.writeMsg(INFO, msg)
}

func (log *Logger) Warn(format string, v ...interface{}) {
	if WARN < log.level {
		return
	}
	msg := fmt.Sprintf("[WARN] "+format, v...)
	log.writeMsg(WARN, msg)
}

func (log *Logger) Error(format string, v ...interface{}) {
	if ERROR < log.level {
		return
	}
	msg := fmt.Sprintf("[ERROR] "+format, v...)
	log.writeMsg(ERROR, msg)
}

func (log *Logger) Fatal(format string, v ...interface{}) {
	if FATAL < log.level {
		return
	}
	msg := fmt.Sprintf("[FATAL] "+format, v...)
	log.writeMsg(FATAL, msg)
}
