package galf

import (
	"fmt"
	"sync"

	"github.com/afex/hystrix-go/hystrix"
)

var circuitConfig map[string]bool
var circuitMutex *sync.RWMutex

func init() {
	circuitConfig = make(map[string]bool)
	circuitMutex = &sync.RWMutex{}
}

// ConfigureCommand applies settings for a circuit
func HystrixConfigureCommand(name string, config hystrix.CommandConfig) {
	circuitMutex.Lock()
	defer circuitMutex.Unlock()
	hystrix.ConfigureCommand(name, config)
	circuitConfig[getHystrixName(name)] = true
}

func existCircuitConfig(name string) bool {
	circuitMutex.RLock()
	_, exists := circuitConfig[getHystrixName(name)]
	circuitMutex.RUnlock()

	return exists
}

func getHystrixName(name string) string {
	return fmt.Sprintf("%s_galf", name)
}
