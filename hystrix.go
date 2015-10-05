package galf

import (
	"errors"
	"fmt"
	"sync"

	"github.com/afex/hystrix-go/hystrix"
)

var hystrixConfigs map[string]bool
var hystrixMutex *sync.RWMutex

func init() {
	hystrixConfigs = make(map[string]bool)
	hystrixMutex = &sync.RWMutex{}
}

type HystrixConfig struct {
	Name       string
	configName string
}

func (hc *HystrixConfig) valid() error {
	if hc.configName != "" {
		return nil
	}

	if !existHystrixConfig(hc.Name) {
		msg := fmt.Sprintf("Hystrix config name not found: %s", hc.Name)
		return errors.New(msg)
	}

	hc.configName = getHystrixConfigName(hc.Name)
	return nil
}

// ConfigureCommand applies settings for a circuit
func HystrixConfigureCommand(name string, config hystrix.CommandConfig) {
	hystrixMutex.Lock()
	defer hystrixMutex.Unlock()
	hystrix.ConfigureCommand(name, config)
	hystrixConfigs[getHystrixConfigName(name)] = true
}

func getHystrixConfigName(name string) string {
	return fmt.Sprintf("%s_galf", name)
}

func existHystrixConfig(name string) bool {
	hystrixMutex.RLock()
	defer hystrixMutex.RUnlock()
	_, exists := hystrixConfigs[getHystrixConfigName(name)]
	return exists
}
