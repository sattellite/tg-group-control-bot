package memo

import (
	"sync"

	"github.com/pkg/errors"
)

// Item contains value
type Item struct {
	Value interface{}
}

// Memo contains all keys
type Memo struct {
	Items map[interface{}]Item
	mutex sync.RWMutex
}

// New returns memo struct
func New() *Memo {
	var m Memo
	m.Items = make(map[interface{}]Item)
	return &m
}

// Set adding key-value pair to memory
func (m *Memo) Set(key, value interface{}) {
	m.mutex.Lock()
	m.Items[key] = Item{
		Value: value,
	}
	m.mutex.Unlock()
}

// Get returning value from storage
func (m *Memo) Get(key interface{}) (interface{}, error) {
	m.mutex.Lock()
	defer func() {
		m.mutex.Unlock()
	}()

	// Check for key exist
	if item, exist := m.Items[key]; exist {
		return item.Value, nil
	}
	return nil, errors.New("notexist")

}
