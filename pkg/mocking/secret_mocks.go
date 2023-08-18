package mocking

import (
	"github.com/mattgill98/vault-init/pkg/vault"
	"github.com/stretchr/testify/mock"
)

type KeyStorageMock struct {
	mock.Mock
}

func (m *KeyStorageMock) Persist(state vault.InitState) (bool, error) {
	args := m.Called(state)
	return args.Bool(0), args.Error(1)
}

func (m *KeyStorageMock) Fetch() (*vault.InitState, error) {
	args := m.Called()
	return args.Get(0).(*vault.InitState), args.Error(1)
}
