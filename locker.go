package indigo

import (
	"fmt"
	"sync"
)

type Locker interface {
	Add(key string, r *Rule)
	Get(key string) *Rule
	Remove(key string)
	Lock()
	Unlock()
	RLock()
	RUnlock()
	ContainsRule(id string) bool
	ReplaceRule(id string, r *Rule) error
	Keys() []string
	RuleCount() int
}

type DefaultLocker struct {
	mu   sync.RWMutex
	root *Rule
}

func NewLocker() *DefaultLocker {
	return &DefaultLocker{
		root: &Rule{
			ID:    "ROOT",
			Rules: map[string]*Rule{},
		},
	}
}

func (l *DefaultLocker) Keys() []string {
	ks := make([]string, len(l.root.Rules))
	for k := range l.root.Rules {
		ks = append(ks, k)
	}
	return ks
}

func (l *DefaultLocker) Add(key string, r *Rule) {
	l.root.Rules[key] = r
}

func (l *DefaultLocker) Get(key string) *Rule {
	return l.root.Rules[key]
}

func (l *DefaultLocker) Remove(key string) {
	delete(l.root.Rules, key)
}

func (l *DefaultLocker) ContainsRule(id string) bool {
	if f := find(l.root, id); f != nil {
		return true
	}
	return false
}

func (l *DefaultLocker) ReplaceRule(id string, n *Rule) error {
	p := find(l.root, id)
	if p == nil {
		return fmt.Errorf("rule with id '%s' not found", id)
	}
	delete(p.Rules, id)
	p.Rules[n.ID] = n
	return nil
}

func (l *DefaultLocker) Lock() {
	l.mu.Lock()
}

func (l *DefaultLocker) Unlock() {
	l.mu.Unlock()
}

func (l *DefaultLocker) RLock() {
	l.mu.RLock()
}

func (l *DefaultLocker) RUnlock() {
	l.mu.RUnlock()
}

func (l *DefaultLocker) RuleCount() int {
	return count(l.root) - 1 // do not count the root
}

func count(r *Rule) int {
	c := 1 // count r
	for k := range r.Rules {
		c = c + count(r.Rules[k])
	}
	return c
}

// find searches rescursively for the parent of the rule with the id,
// starting in the provided rule. The provided rule cannot be the
// rule.
func find(p *Rule, id string) *Rule {
	if p == nil {
		return nil
	}
	for k := range p.Rules {
		c := p.Rules[k]
		if c.ID == id {
			return p
		}
		if f := find(c, id); f != nil {
			return f
		}
	}
	return nil
}
