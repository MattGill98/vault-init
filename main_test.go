package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mattgill98/vault-init/pkg/mocking"
	"github.com/mattgill98/vault-init/pkg/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDelayFn struct {
	mock.Mock
}

func (m *MockDelayFn) func1(d time.Duration) {
	m.Called(d)
}

func TestWaitForVault_VaultDown(t *testing.T) {
	mockVault := new(mocking.VaultMock)
	vaultClient = mockVault
	mockDelay := new(MockDelayFn)

	mockDelay.On("func1", 1*time.Second).Once().Return()
	mockVault.On("HealthCheck").Once().Return(vault.HealthState{}, fmt.Errorf("Failed to call vault"))
	mockVault.On("HealthCheck").Once().Return(vault.HealthState{}, nil)

	WaitForVault(mockDelay.func1)
	mockDelay.AssertNumberOfCalls(t, "func1", 1)
	mockVault.AssertNumberOfCalls(t, "HealthCheck", 2)
}

func TestWaitForVault_VaultUp(t *testing.T) {
	mockVault := new(mocking.VaultMock)
	vaultClient = mockVault

	mockVault.On("HealthCheck").Once().Return(vault.HealthState{Active: true}, nil)
	mockVault.On("HealthCheck").Once().Return(vault.HealthState{Standby: true}, nil)
	mockVault.On("HealthCheck").Once().Return(vault.HealthState{Uninitialized: true}, nil)
	mockVault.On("HealthCheck").Once().Return(vault.HealthState{Sealed: true}, nil)
	mockVault.On("HealthCheck").Once().Return(vault.HealthState{StatusCode: 418}, nil)

	statusFn := func() vault.HealthState { return WaitForVault(func(d time.Duration) {}) }
	assert.Equal(t, vault.HealthState{Active: true}, statusFn())
	assert.Equal(t, vault.HealthState{Standby: true}, statusFn())
	assert.Equal(t, vault.HealthState{Uninitialized: true}, statusFn())
	assert.Equal(t, vault.HealthState{Sealed: true}, statusFn())
	assert.Equal(t, vault.HealthState{StatusCode: 418}, statusFn())
	mockVault.AssertCalled(t, "HealthCheck")
}

func TestGetVaultAddress_EmptyString(t *testing.T) {
	os.Setenv("VAULT_ADDR", "")
	assert.Equal(t, DEFAULT_VAULT_ADDR, GetVaultAddress(), "Expected the default vault address")
}

func TestGetVaultAddress_ValidString(t *testing.T) {
	vaultAddr := "http://127.0.0.1/myvault"
	os.Setenv("VAULT_ADDR", vaultAddr)
	assert.Equal(t, vaultAddr, GetVaultAddress(), "Expected the default vault address")
}
