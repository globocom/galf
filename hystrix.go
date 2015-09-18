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
		// log.WithFields(log.Fields{
		// 	"HystrixConfigName": hc.Name,
		// }).Fatal("Não foi configurado o galf hytrix com essse circuit name")
		return errors.New("dsdd")
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
	_, exists := hystrixConfigs[getHystrixConfigName(name)]
	hystrixMutex.RUnlock()

	return exists
}
