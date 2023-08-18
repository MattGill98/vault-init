package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mattgill98/vault-init/pkg/secret"
	"github.com/mattgill98/vault-init/pkg/vault"
)

const (
	DEFAULT_VAULT_ADDR = "http://127.0.0.1:8200"
)

var (
	address                 = GetVaultAddress()
	debugLogging            = GetDebugLogging()
	vaultClient             vault.Vault
	keyStorage              secret.KeyStorage
	createKubernetesStorage = func() (secret.KeyStorage, error) { return secret.NewKubernetesSecretStorage("vault-keys", "default") }
	createInMemoryStorage   = func() secret.KeyStorage { return secret.NewMemorySecretStorage(log.Default()) }
)

func main() {
	vaultClient = vault.NewVaultClient(address)

	storage, err := GetStorage()
	if err != nil {
		panic(err.Error())
	}
	keyStorage = storage

	for {
		ok, err := run()
		if !ok {
			panic(err.Error())
		}
		time.Sleep(5 * time.Second)
	}
}

func run() (bool, error) {
	vaultState := WaitForVault(func(d time.Duration) {
		time.Sleep(d)
	})

	if vaultState.Uninitialized {
		state, err := InitializeVault()
		if err != nil {
			return false, err
		}
		ok, err := SaveState(*state)
		if !ok {
			return false, err
		}
		ok, err = UnsealVaultFromState(*state)
		if !ok {
			return false, err
		}
	}

	if vaultState.Sealed {
		UnsealVault()
	}

	return true, nil
}

func GetStorage() (secret.KeyStorage, error) {
	kubeStorage, err := createKubernetesStorage()
	if kubeStorage != nil {
		return kubeStorage, nil
	}
	if err == secret.ErrNotInCluster {
		log.Println("No Kubernetes environment detected")
		return createInMemoryStorage(), nil
	}
	return nil, err
}

func WaitForVault(delay func(d time.Duration)) vault.HealthState {
	for {
		state, err := vaultClient.HealthCheck()
		if err != nil {
			log.Println(err)
			delay(1 * time.Second)
			continue
		}

		if debugLogging == true {
			switch true {
			case state.Active:
				log.Println("Vault is initialized and unsealed.")
			case state.Standby:
				log.Println("Vault is unsealed and in standby mode.")
			case state.Uninitialized:
				log.Println("Vault is not initialized.")
			case state.Sealed:
				log.Println("Vault is sealed.")
			default:
				log.Printf("Vault is in an unknown state. Status code: %d", state.StatusCode)
			}
		}

		return state
	}
}

func InitializeVault() (*vault.InitState, error) {
	log.Println("Initialising Vault...")

	state, err := vaultClient.Initialize()
	if err != nil {
		return nil, fmt.Errorf("Initialization error: %w", err)
	}
	return &state, nil
}

func SaveState(state vault.InitState) (bool, error) {
	log.Println("Storing Vault keys...")
	return keyStorage.Persist(state)
}

func UnsealVault() (bool, error) {
	state, err := keyStorage.Fetch()
	if err != nil {
		return false, fmt.Errorf("Failed to fetch keys: %w", err)
	}
	return UnsealVaultFromState(*state)
}

func UnsealVaultFromState(state vault.InitState) (bool, error) {
	log.Println("Unsealing Vault...")
	for index, key := range state.Keys {
		event, err := vaultClient.Unseal(key)
		if err != nil {
			log.Printf("Failed to unseal using key [%d]", index)
			continue
		}
		log.Printf("Unseal progress: [%d/%d]", event.KeysProvided, event.KeysRequired)
		if !event.Sealed {
			return true, nil
		}
	}
	return false, fmt.Errorf("Too many unseal failures")
}

func GetVaultAddress() string {
	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr != "" {
		return vaultAddr
	}
	log.Printf("VAULT_ADDR not set, defaulting to %q", DEFAULT_VAULT_ADDR)
	return DEFAULT_VAULT_ADDR
}

func GetDebugLogging() bool {
	value := os.Getenv("DEBUG")
	return strings.EqualFold(value, "true")
}
