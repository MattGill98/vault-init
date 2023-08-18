package secret

import (
	"fmt"
	"log"

	"github.com/mattgill98/vault-init/pkg/vault"
)

type Logger interface {
	Printf(format string, args ...interface{})
}
type BuiltinLogger struct {
	logger log.Logger
}

func (l *BuiltinLogger) Printf(format string, args ...interface{}) {
	l.logger.Printf(format, args...)
}

func NewBuiltinLogger() Logger {
	return &BuiltinLogger{
		logger: *log.Default(),
	}
}

type memorySecretStorage struct {
	logger      Logger
	storedState *vault.InitState
}

func NewMemorySecretStorage(logger Logger) KeyStorage {
	return &memorySecretStorage{
		logger: logger,
	}
}

func (memory *memorySecretStorage) Persist(state vault.InitState) (bool, error) {
	memory.storedState = &state
	if memory.logger != nil {
		memory.logger.Printf("Root key: %v", state.RootToken)
		memory.logger.Printf("Seal Keys: %v", state.Keys)
	}
	return true, nil
}

func (memory *memorySecretStorage) Fetch() (*vault.InitState, error) {
	state := memory.storedState
	if state == nil {
		return nil, fmt.Errorf("State not found")
	}
	return state, nil
}
