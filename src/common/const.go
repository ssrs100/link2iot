package common

import "github.com/VolantMQ/vlapi/mqttp"

var (
	VERSION = []string{"v3.1.1"}

	// options
	ConnectTimeout = 2
	OfflineQoS0 = true
	SessionDups = true
	RetainAvailable = true
	SubsOverlap = false
	SubsID = false
	SubsShared = false
	SubsWildcard = true
	ReceiveMax = 65535
	MaxPacketSize uint32 = 268435455
	MaxTopicAlias uint16 = 65535
	MaxQoS mqttp.QosType = 2

	// keepAlive:
	Period = 60
	Force = false

	// systree:
	Enabled = true
	UpdateInterval = 10

	// acceptor:
	MaxIncoming = 1000
	PreSpawn = 100
)
