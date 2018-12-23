package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)


type simpleAuth struct {
	sync.RWMutex
	creds map[string]*User
}

var auth *simpleAuth

func NewSimpleAuth() *simpleAuth {
	auth = &simpleAuth{
		creds: make(map[string]*User),
	}
	return auth
}

func GetAuth() *simpleAuth {
	return auth
}

func (a *simpleAuth) AddUser(userMap map[string]string) {
	u := userMap["name"]
	p := userMap["password"]
	id := userMap["project_id"]
	if len(u) == 0 || len(p) == 0 || len(id) == 0 {
		log.Error("Add user failed, params:%v", userMap)
		return
	}
	algo := sha256.New()
	algo.Write([]byte(p))
	hash := hex.EncodeToString(algo.Sum(nil))

	user := &User{
		Name:         u,
		PasswordHash: hash,
		ProjectId:    id,
	}

	a.Lock()
	defer a.Unlock()
	a.creds[u] = user
}

func (a *simpleAuth) DelUser(u string) {
	a.Lock()
	defer a.Unlock()
	delete(a.creds, u)
}

func (a *simpleAuth) GetUser(u string) *User{
	a.RLock()
	defer a.RUnlock()
	return a.creds[u]
}

// nolint: golint
func (a *simpleAuth) Password(clientID, user, password string) error {
	log.Debug("connect user:%s, password:%s", user, password)
	log.Debug("a.creds:%v", a.creds)
	a.RLock()
	u, ok := a.creds[user]
	a.RUnlock()
	if ok {
		algo := sha256.New()
		algo.Write([]byte(password))
		loginHash := hex.EncodeToString(algo.Sum(nil))
		log.Debug("loginHash:%s, hash:%s", loginHash, u.PasswordHash)
		if loginHash == u.PasswordHash {
			return StatusAllow
		}
	}
	return StatusDeny
}

// nolint: golint
func (a *simpleAuth) ACL(clientID, user, topic string, access AccessType) error {
	return StatusAllow
}

// nolint: golint
func (a *simpleAuth) Shutdown() error {
	a.Lock()
	defer a.Unlock()
	a.creds = nil
	return nil
}
