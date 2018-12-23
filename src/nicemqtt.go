package main

import (
	_ "github.com/go-sql-driver/mysql"

	"auth"
	"conf"
	"crypto/tls"
	"fmt"
	"github.com/VolantMQ/vlapi/plugin/persistence/mem"
	"io/ioutil"
	"logs"
	"orm"
	"os"
	"os/signal"
	"path/filepath"
	"server"
	"syscall"
	"transport"
	"utils"
)

var (
	log           = logs.GetLogger()
	server_config = "nicemqtt.json"
	config        *conf.Config

	basedir string
)

type User struct {
	Id      string `orm:"size(64);pk"`
	Name    string `orm:"size(128)"`
	Passwd  string `orm:"size(128)"`
	Email   string `orm:"size(128);null"`
	Mobile  string `orm:"size(128);null"`
	Address string `orm:"size(512);null"`
}


func initDB() error {
	// register model
	orm.RegisterModel(new(User))

	dsn := fmt.Sprintf("blue:blue@123@tcp(%s:%d)/blue?charset=utf8", config.GetString("db_host"),
		config.GetIntWithDefault("db_port", 3306))
	// set default database
	if err := orm.RegisterDataBase("default", "mysql", dsn, 30); err != nil {
		return err
	}

	// create table
	if err := orm.RunSyncdb("default", false, true); err != nil {
		return err
	}
	return nil
}

func getUsers() []User {
	var users []User
	// 获取 QueryBuilder 对象. 需要指定数据库驱动参数。
	// 第二个返回值是错误对象，在这里略过
	qb, err := orm.NewQueryBuilder("mysql")
	if err != nil {
		log.Error("build sql error:%s", err.Error())
		return users
	}

	// 构建查询对象
	qb = qb.Select("*").From("user")

	// 导出 SQL 语句
	sql := qb.String()
	log.Debug(sql)
	// 执行 SQL 语句
	o := orm.NewOrm()
	o.Raw(sql).QueryRows(&users)

	return users
}

func registerAuth() *auth.Manager {
	if err := initDB(); err != nil {
		log.Error("initDB fail:%s", err.Error())
		return nil
	}
	sAuth := auth.NewSimpleAuth()
	userList := getUsers()
	log.Debug("user size:%d", len(userList))
	for _, user := range userList {
		userMap := make(map[string] string)
		userMap["name"] = user.Name
		userMap["password"] = user.Passwd
		userMap["project_id"] = user.Id
		sAuth.AddUser(userMap)
	}
	_ = auth.Register("internal", sAuth)
	def, _ := auth.NewManager([]string{"internal"}, false)
	return def
}

func loadMqtt(defaultAuth *auth.Manager) *transport.ConfigTCP {

	tCfg := &transport.Config{
		Host:        config.GetString("host"),
		Port:        config.GetString("port"),
		AuthManager: defaultAuth,
	}

	tcpConfig := transport.NewConfigTCP(tCfg)

	if config.GetBoolWithDefault("ssl_enable", false) {
		cerPath := filepath.Join(basedir, "conf", "ssl.crt")
		certPEMBlock, err := ioutil.ReadFile(cerPath)
		if err != nil {
			return nil
		}
		keyPath := filepath.Join(basedir, "conf", "ssl.key")
		keyPEMBlock, err := ioutil.ReadFile(keyPath)
		if err != nil {
			return nil
		}
		x509, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
		if err != nil {
			return nil
		}
		c := &tls.Config{}
		c.Certificates = append(c.Certificates, x509)
		tcpConfig.TLS = c
	}

	return tcpConfig

}

func main() {
	defer func() {
		log.Info("service stopped")

		if r := recover(); r != nil {
			log.Error("%v", r)
		}
	}()
	basedir = utils.GetAppBaseDir()
	if len(basedir) == 0 {
		log.Error("Evironment APP_BASE_DIR(app installed root path) should be set.")
		os.Exit(1)
	}
	//
	//获取配置信息
	appConfig := filepath.Join(basedir, "conf", server_config)
	config = conf.LoadFile(appConfig)
	if config == nil {
		errStr := fmt.Sprintf("Can not load %s.", server_config)
		log.Error(errStr)
		os.Exit(1)
	}

	// 注册鉴权
	auth := registerAuth()

	tcpConfig := loadMqtt(auth)

	persist, _ := persistenceMem.Load(nil, nil)

	serverConfig := server.Config{
		Persistence: persist,
		TransportStatus: func(id string, status string) {
			log.Info("listener id: %s status: %s", id, status)
		},
		OnDuplicate: func(s string, b bool) {
			log.Info("Session duplicate: clientId: %s, allowed: %v", s, b)
		},
	}
	srv, err := server.NewServer(serverConfig)
	if err != nil {
		log.Error("server create err:%s", err.Error())
		return
	}

	log.Info("MQTT server created")

	// http server start
	host := config.GetString("host")
	go server.StartHTTPServer(host, "8080")

	log.Info("MQTT starting listeners")
	if err = srv.ListenAndServe(tcpConfig); err != nil {
		log.Error("listen and serve err:%s", err.Error())
		return
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	log.Info("service received signal: %s", sig.String())

	if err = srv.Shutdown(); err != nil {
		log.Error("shutdown server err:%s", err.Error())
	}

	log.Info("service stopped.")
}
