package mocking

import (
	"github.com/mattgill98/vault-init/pkg/vault"
	"github.com/stretchr/testify/mock"
)

type VaultMock struct {
	mock.Mock
}

func (m *VaultMock) HealthCheck() (vault.HealthState, error) {
	args := m.Called()
	return args.Get(0).(vault.HealthState), args.Error(1)
}
func (m *VaultMock) Initialize() (vault.InitResponse, error) {
	args := m.Called()
	return args.Get(0).(vault.InitResponse), args.Error(1)
}
func (m *VaultMock) Unseal(key string) (vault.UnsealState, error) {
	args := m.Called()
	return args.Get(0).(vault.UnsealState), args.Error(1)
}
