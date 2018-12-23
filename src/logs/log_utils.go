package logs

import (
	"runtime"
	"strconv"
	"strings"
	"time"
)

func formatTimeHeader(when time.Time) (string, int, error) {
	//time format: 2016-04-27 10:18:51.453993
	str := when.Format("2006-01-02 15:04:05")
	_, _, d := when.Date()
	us := when.Nanosecond() / 1000
	return ("[" + str + "." + strconv.Itoa(us) + "]"), d, nil
}

//转换日志级别，由字符串到数字转换
func transLogLevel(level string) int {
	var ret int = DEBUG
	switch level {
	case "DEBUG":
		ret = DEBUG
	case "INFO":
		ret = INFO
	case "WARN":
		ret = WARN
	case "ERROR":
		ret = ERROR
	case "FATAL":
		ret = FATAL
	default:
		ret = DEBUG
	}
	return ret
}

//获取当前的协程id。官方不提供go id，这里通过堆栈信息获取，仅DEBUG日志使用
func getGID() string {
	buf := make([]byte, 64)
	n := runtime.Stack(buf[:], false)
	gid := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine"))[0]
	return gid
}
