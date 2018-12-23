package auth

import (
	"errors"
	"fmt"
)

// Manager auth
type Manager struct {
	p         []IFace
	anonymous bool
}

var providers = make(map[string]IFace)

// Register auth provider
func Register(name string, i IFace) error {
	if name == "" && i == nil {
		return errors.New("invalid args")
	}

	if _, dup := providers[name]; dup {
		return errors.New("already exists")
	}

	providers[name] = i

	return nil
}

// UnRegister authenticator
func UnRegister(name string) {
	delete(providers, name)
}

// NewManager new auth manager
func NewManager(p []string, allowAnonymous bool) (*Manager, error) {
	m := Manager{
		anonymous: allowAnonymous,
	}

	for _, pa := range p {
		pvd, ok := providers[pa]
		if !ok {
			return nil, fmt.Errorf("session: unknown provider %q", pa)
		}

		m.p = append(m.p, pvd)
	}

	return &m, nil
}

// AllowAnonymous allow anonymous connections
func (m *Manager) AllowAnonymous() error {
	if m.anonymous {
		return StatusAllow
	}

	return StatusDeny
}

// Password authentication
func (m *Manager) Password(clientID, user, password string) error {
	if user == "" && m.anonymous {
		return StatusAllow
	} else {
		for _, p := range m.p {
			if status := p.Password(clientID, user, password); status == StatusAllow {
				return status
			}
		}
	}

	return StatusDeny
}

// Password authentication
func (m *Manager) FetchUser(user string) *User {
	for _, p := range m.p {
		if user := p.GetUser(user); user != nil {
			return user
		}
	}
	return nil
}

// ACL check permissions
func (m *Manager) ACL(clientID, user, topic string, access AccessType) error {
	for _, p := range m.p {
		if status := p.ACL(clientID, user, topic, access); status == StatusAllow {
			return status
		}
	}

	return StatusDeny
}
