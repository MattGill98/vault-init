package secret

import (
	"testing"

	"github.com/mattgill98/vault-init/pkg/vault"
	"github.com/stretchr/testify/mock"
)

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Printf(format string, args ...interface{}) {
	collectArgs := []interface{}{format}
	for _, arg := range args {
		collectArgs = append(collectArgs, arg)
	}
	m.Called(collectArgs...)
}

func TestPersist(t *testing.T) {
	logger := new(MockLogger)
	logger.On("Printf", mock.Anything, mock.Anything).Return()

	storage := NewMemorySecretStorage(logger)

	state := vault.InitState{
		Keys:      []string{"a", "b", "c"},
		RootToken: "abcdefg",
	}
	storage.Persist(state)

	logger.AssertCalled(t, "Printf", mock.AnythingOfType("string"), state.Keys)
	logger.AssertCalled(t, "Printf", mock.AnythingOfType("string"), state.RootToken)
}
