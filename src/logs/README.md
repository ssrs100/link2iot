# logs
====

日志模块说明

1.依赖环境变量，使用前需配置包安装的根路径环境变量，如

```
APP_BASE_DIR=F:\naga

```

2.包的外部使用方法

```
log := logs.GetLogger()

log.Debug("xx")

```

3. 配置文件根路径的conf目录下，可修改日志级别和路径等

```
//记录日志的方式，支持console,file两种
  "pattern":"file",
//指定日志的路径
  "filename":"F:\\test\\logs\\test.log",
//指定日志级别，支持DEBUG/INFO/WARN/ERROR/FATAL级别
  "logLevel":"DEBUG",
//日志滚动参数
  "rotate":true,  //是否滚动
  "maxsize":256,  //单个日志文件最大大小（M）
  "maxdays":7,    //最大保留天数
  "maxlines":1000000,  //单个文件最大日志行数
  "maxTotalSize":10240, //日志占用的最大空间（M）
  "daily": true   //是否每天产生一个日志文件
```
