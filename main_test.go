package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/mattgill98/vault-init/pkg/mocking"
	"github.com/mattgill98/vault-init/pkg/secret"
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

func TestGetStorage_Error(t *testing.T) {
	createKubernetesStorage = func() (secret.KeyStorage, error) { return nil, fmt.Errorf("Mock error") }

	storage, err := GetStorage()
	assert.Nil(t, storage)
	assert.Equal(t, err.Error(), "Mock error")
}

func TestGetStorage_InMemory(t *testing.T) {
	createKubernetesStorage = func() (secret.KeyStorage, error) { return nil, secret.ErrNotInCluster }
	mockInMemoryStorage := new(mocking.KeyStorageMock)
	createInMemoryStorage = func() secret.KeyStorage { return mockInMemoryStorage }

	storage, err := GetStorage()
	assert.Equal(t, mockInMemoryStorage, storage)
	assert.Nil(t, err)
}

func TestGetStorage_Kubernetes(t *testing.T) {
	mockKubernetesStorage := new(mocking.KeyStorageMock)
	createKubernetesStorage = func() (secret.KeyStorage, error) { return mockKubernetesStorage, nil }

	storage, err := GetStorage()
	assert.Equal(t, mockKubernetesStorage, storage)
	assert.Nil(t, err)
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

func TestInitializeVault_Success(t *testing.T) {
	mockVault := new(mocking.VaultMock)
	vaultClient = mockVault

	mockState := vault.InitState{Keys: []string{"a"}, RootToken: "b"}
	mockVault.On("Initialize").Once().Return(mockState, nil)

	state, err := InitializeVault()
	assert.Nil(t, err)
	assert.Equal(t, mockState.Keys, state.Keys)
	assert.Equal(t, mockState.RootToken, state.RootToken)

	mockVault.AssertCalled(t, "Initialize")
}

func TestInitializeVault_InitializationError(t *testing.T) {
	mockVault := new(mocking.VaultMock)
	vaultClient = mockVault
	mockVault.On("Initialize").Once().Return(vault.InitState{}, fmt.Errorf("Mock error"))

	state, err := InitializeVault()
	assert.Nil(t, state)
	assert.Contains(t, err.Error(), "Mock error", "Initialization error")

	mockVault.AssertCalled(t, "Initialize")
}

func TestUnsealVault_FetchError(t *testing.T) {
	mockKeyStorage := new(mocking.KeyStorageMock)
	keyStorage = mockKeyStorage
	mockKeyStorage.On("Fetch").Return(&vault.InitState{}, fmt.Errorf("Mock error"))

	ok, err := UnsealVault()
	assert.False(t, ok)
	assert.Contains(t, err.Error(), "Mock error", "Failed to fetch keys")
}

func TestUnsealVault_Success(t *testing.T) {
	mockKeyStorage := new(mocking.KeyStorageMock)
	keyStorage = mockKeyStorage
	mockKeys := []string{"a", "b", "c", "d"}
	mockKeyStorage.On("Fetch").Return(&vault.InitState{Keys: mockKeys}, nil)

	mockVault := new(mocking.VaultMock)
	vaultClient = mockVault
	mockVault.On("Unseal", mock.Anything).Times(2).Return(vault.UnsealState{Sealed: true}, nil)
	mockVault.On("Unseal", mock.Anything).Once().Return(vault.UnsealState{Sealed: false}, nil)

	ok, err := UnsealVault()
	assert.True(t, ok)
	assert.Nil(t, err)
	mockVault.AssertNumberOfCalls(t, "Unseal", 3)
}

func TestUnsealVaultFromState_TooManyFailures(t *testing.T) {
	mockVault := new(mocking.VaultMock)
	vaultClient = mockVault
	mockVault.On("Unseal", mock.Anything).Times(2).Return(vault.UnsealState{Sealed: true}, nil)
	mockVault.On("Unseal", mock.Anything).Once().Return(vault.UnsealState{Sealed: false}, nil)

	ok, err := UnsealVaultFromState(vault.InitState{Keys: []string{"a", "b", "c"}})
	assert.True(t, ok)
	assert.Nil(t, err)
	mockVault.AssertNumberOfCalls(t, "Unseal", 3)
	mockVault.AssertCalled(t, "Unseal", "a")
	mockVault.AssertCalled(t, "Unseal", "b")
	mockVault.AssertCalled(t, "Unseal", "c")
}

func TestUnsealVaultFromState_SingleError(t *testing.T) {
	mockVault := new(mocking.VaultMock)
	vaultClient = mockVault
	mockVault.On("Unseal", mock.Anything).Once().Return(vault.UnsealState{Sealed: true}, nil)
	mockVault.On("Unseal", mock.Anything).Once().Return(vault.UnsealState{}, fmt.Errorf("Mock error"))
	mockVault.On("Unseal", mock.Anything).Once().Return(vault.UnsealState{Sealed: false}, nil)

	ok, err := UnsealVaultFromState(vault.InitState{Keys: []string{"a", "b", "c"}})
	assert.True(t, ok)
	assert.Nil(t, err)
	mockVault.AssertNumberOfCalls(t, "Unseal", 3)
	mockVault.AssertCalled(t, "Unseal", "a")
	mockVault.AssertCalled(t, "Unseal", "b")
	mockVault.AssertCalled(t, "Unseal", "c")
}

func TestUnsealVaultFromState_MultipleErrors(t *testing.T) {
	mockVault := new(mocking.VaultMock)
	vaultClient = mockVault
	mockVault.On("Unseal", mock.Anything).Once().Return(vault.UnsealState{Sealed: true}, nil)
	mockVault.On("Unseal", mock.Anything).Times(2).Return(vault.UnsealState{}, fmt.Errorf("Mock error"))

	ok, err := UnsealVaultFromState(vault.InitState{Keys: []string{"a", "b", "c"}})
	assert.False(t, ok)
	assert.Equal(t, "Too many unseal failures", err.Error())
	mockVault.AssertNumberOfCalls(t, "Unseal", 3)
	mockVault.AssertCalled(t, "Unseal", "a")
	mockVault.AssertCalled(t, "Unseal", "b")
	mockVault.AssertCalled(t, "Unseal", "c")
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
