package galf

import (
	"fmt"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/afex/hystrix-go/hystrix"
)

var hystrixConfigs map[string]bool
var hystrixMutex *sync.RWMutex

func init() {
	hystrixConfigs = make(map[string]bool)
	hystrixMutex = &sync.RWMutex{}
}

type HystrixConfig struct {
	Name        string
	nameHystrix string
}

func (hc *HystrixConfig) useHystrix() bool {
	if hc.nameHystrix != "" {
		return true
	}

	if !existHystrixConfig(hc.Name) {
		log.WithFields(log.Fields{
			"HystrixConfigName": hc.Name,
		}).Fatal("Não foi configurado o galf hytrix com essse circuit name")
	}
	hc.nameHystrix = getHystrixConfigName(hc.Name)
	return true
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
