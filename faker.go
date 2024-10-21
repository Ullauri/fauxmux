package fauxmux

import "sync"

// FakeDataFunc is a function that generates fake data for a given type
type FakeDataFunc func(v interface{}) error

type Config struct {
	FakeDataFunc FakeDataFunc
}

func (c *Config) FakeData(v interface{}) error {
	return c.FakeDataFunc(v)
}

var config Config
var mutex sync.Mutex

// Setup sets the configuration for the fauxmux package
func Setup(c Config) {
	mutex.Lock()
	defer mutex.Unlock()
	config = c
}
