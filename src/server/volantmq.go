package server

import (
	"common"
	"errors"
	"logs"
	"regexp"
	"sync"
	"time"

	"clients"
	"github.com/VolantMQ/vlapi/mqttp"
	"github.com/VolantMQ/vlapi/plugin"
	"github.com/VolantMQ/vlapi/plugin/persistence"
	"github.com/VolantMQ/vlapi/subscriber"
	"github.com/troian/easygo/netpoll"
	"systree"
	"topics"
	"topics/types"
	"transport"
	"types"
)

var (
	// nolint: megacheck
	nodeNameRegexp = regexp.MustCompile(
		"^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}" +
			"[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
)

var (
	// ErrInvalidNodeName node name does not follow requirements
	ErrInvalidNodeName = errors.New("node name is invalid")

	log = logs.GetLogger()
)

// Config configuration of the MQTT server
type Config struct {
	//MQTT     configuration.MqttConfig
	//Acceptor configuration.AcceptorConfig

	// Configuration of persistence provider
	Persistence persistence.IFace

	// OnDuplicate notify if there is attempt connect client with id that already exists and active
	// If not not set than defaults to mock function
	OnDuplicate func(string, bool)

	// TransportStatus user provided callback to track transport status
	// If not set than defaults to mock function
	TransportStatus func(id string, status string)

	// NodeName
	NodeName string
}

// Server server API
type Server interface {
	// ListenAndServe configures transport according to provided config
	// This is non blocking function. It returns nil if listener started
	// or error if any happened during configuration.
	// Transport status reported over TransportStatus callback in server configuration
	ListenAndServe(interface{}) error

	// Shutdown terminates the server by shutting down all the client connections and closing
	// configured listeners. It does full clean up of the resources and
	Shutdown() error
}

// server is a library implementation of the MQTT server that, as best it can, complies
// with the MQTT 3.1/3.1.1 and 5.0 specs.
type server struct {
	Config
	sessionsMgr *clients.Manager
	topicsMgr   topicsTypes.Provider
	sysTree     systree.Provider
	quit        chan struct{}
	lock        sync.Mutex
	onClose     sync.Once
	ePoll       netpoll.EventPoll
	acceptPool  types.Pool
	transports  struct {
		list map[string]transport.Provider
		wg   sync.WaitGroup
	}
	systree struct {
		publishes []systree.DynamicValue
		timer     *time.Ticker
		wg        sync.WaitGroup
	}
}

var _ vlplugin.Messaging = (*server)(nil)

// NewServer allocate server object
func NewServer(config Config) (Server, error) {
	s := &server{
		Config: config,
	}

	if config.NodeName != "" {
		if !nodeNameRegexp.MatchString(config.NodeName) {
			return nil, ErrInvalidNodeName
		}
	}

	s.quit = make(chan struct{})
	s.transports.list = make(map[string]transport.Provider)

	var err error

	if s.Persistence == nil {
		return nil, errors.New("persistence provider cannot be nil")
	}

	var persisRetained persistence.Retained
	var retains []types.RetainObject

	if s.sysTree, retains, s.systree.publishes, err = systree.NewTree("$SYS/servers/" + s.NodeName); err != nil {
		return nil, err
	}

	persisRetained, _ = s.Persistence.Retained()

	topicsConfig := topicsTypes.NewMemConfig()

	topicsConfig.Stat = s.sysTree.Topics()
	topicsConfig.Persist = persisRetained
	topicsConfig.OverlappingSubscriptions = common.SubsOverlap

	if s.topicsMgr, err = topics.New(topicsConfig); err != nil {
		return nil, err
	}

	if common.Enabled {
		s.sysTree.SetCallbacks(s.topicsMgr)

		for _, o := range retains {
			if err = s.topicsMgr.Retain(o); err != nil {
				return nil, err
			}
		}

		if common.UpdateInterval > 0 {
			s.systree.timer = time.NewTicker(time.Duration(common.UpdateInterval) * time.Second)
			s.systree.wg.Add(1)
			go s.systreeUpdater()
		}
	}

	if s.ePoll, err = netpoll.New(nil); err != nil {
		return nil, err
	}

	s.acceptPool = types.NewPool(common.MaxIncoming, 1, common.PreSpawn)

	mConfig := &clients.Config{
		TopicsMgr:        s.topicsMgr,
		Persist:          s.Persistence,
		Systree:          s.sysTree,
		OnReplaceAttempt: s.OnDuplicate,
		NodeName:         s.NodeName,
	}

	if s.sessionsMgr, err = clients.NewManager(mConfig); err != nil {
		return nil, err
	}

	return s, nil
}

// GetSubscriber ...
func (s *server) GetSubscriber(id string) (vlsubscriber.IFace, error) {
	return s.sessionsMgr.GetSubscriber(id)
}

// ListenAndServe start listener
func (s *server) ListenAndServe(config interface{}) error {
	var l transport.Provider

	var err error

	internalConfig := transport.InternalConfig{
		Handler:    s.sessionsMgr,
		EPoll:      s.ePoll,
		AcceptPool: s.acceptPool,
		Metric:     s.sysTree.Metric(),
	}

	switch c := config.(type) {
	case *transport.ConfigTCP:
		l, err = transport.NewTCP(c, &internalConfig)
	// todo (troian) proper websocket implementation
	// case *transport.ConfigWS:
	// 	l, err = transport.NewWS(c, &internalConfig)
	default:
		return errors.New("invalid listener type")
	}

	if err != nil {
		return err
	}

	defer s.lock.Unlock()
	s.lock.Lock()

	if _, ok := s.transports.list[l.Port()]; ok {
		l.Close() // nolint: errcheck
		return errors.New("already exists")
	}

	s.transports.list[l.Port()] = l
	s.transports.wg.Add(1)
	go func() {
		defer s.transports.wg.Done()

		s.TransportStatus(":"+l.Port(), "started")

		status := "stopped"

		//s.Health.AddReadinessCheck("listener:"+l.Port(), func() error {
		//	if e := l.Ready(); e != nil {
		//		return e
		//	}
		//
		//	return healthcheck.TCPDialCheck(":"+l.Port(), 1*time.Second)()
		//})
		//
		//s.Health.AddLivenessCheck("listener:"+l.Port(), func() error {
		//	if e := l.Alive(); e != nil {
		//		return e
		//	}
		//
		//	return healthcheck.TCPDialCheck(":"+l.Port(), 1*time.Second)()
		//})

		if e := l.Serve(); e != nil {
			status = e.Error()
		}

		s.TransportStatus(":"+l.Port(), status)
	}()

	return nil
}

// Shutdown server
func (s *server) Shutdown() error {
	// By closing the quit channel, we are telling the server to stop accepting new
	// connection.
	s.onClose.Do(func() {
		close(s.quit)

		defer s.lock.Unlock()
		s.lock.Lock()

		// We then close all net.Listener, which will force Accept() to return if it's
		// blocked waiting for new connections.
		for _, l := range s.transports.list {
			if err := l.Close(); err != nil {
				log.Error(err.Error())
			}
		}

		// Wait all of listeners has finished
		s.transports.wg.Wait()

		for port := range s.transports.list {
			delete(s.transports.list, port)
		}

		s.sessionsMgr.Stop() // nolint: errcheck, gas

		// shutdown systree updater
		if s.systree.timer != nil {
			s.systree.timer.Stop()
			s.systree.wg.Wait()
		}

		if err := s.sessionsMgr.Shutdown(); err != nil {
			log.Error("stop session manager, err:%s", err.Error())
		}

		if err := s.topicsMgr.Shutdown(); err != nil {
			log.Error("stop topics manager manager, err:%s", err.Error())
		}

		s.acceptPool.Close()
	})

	return nil
}

func (s *server) systreeUpdater() {
	defer func() {
		s.systree.wg.Done()
	}()

	select {
	case <-s.systree.timer.C:
		for _, val := range s.systree.publishes {
			p := val.Publish()
			pkt := mqttp.NewPublish(mqttp.ProtocolV311)

			pkt.SetPayload(p.Payload())
			pkt.SetTopic(p.Topic())  // nolint: errcheck
			pkt.SetQoS(p.QoS())      // nolint: errcheck
			s.topicsMgr.Publish(pkt) // nolint: errcheck
		}
	case <-s.quit:
		return
	}
}
