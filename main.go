package main

import (
	"log"
	"os"
	"time"

	"github.com/mattgill98/vault-init/pkg/vault"
)

const (
	DEFAULT_VAULT_ADDR = "http://127.0.0.1:8200"
)

var (
	vaultClient vault.Vault
)

func main() {
	address := GetVaultAddress()
	vaultClient = vault.NewVaultClient(address)

	vaultState := WaitForVault(func(d time.Duration) {
		time.Sleep(d)
	})

	if vaultState.Uninitialized {
		InitializeVault()
	}
}

func WaitForVault(delay func(d time.Duration)) vault.HealthState {
	for {
		state, err := vaultClient.HealthCheck()
		if err != nil {
			log.Println(err)
			delay(1 * time.Second)
			continue
		}

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

		return state
	}
}

func InitializeVault() {
	log.Println("Initialising Vault...")

	response, err := vaultClient.Initialize()
	if err != nil {
		log.Printf("Initialization error: %q", err)
		return
	}

	log.Println("Unsealing Vault...")
	for index, key := range response.Keys {
		event, err := vaultClient.Unseal(key)
		if err != nil {
			log.Printf("Failed to unseal using key [%d]", index)
			continue
		}
		log.Printf("Unseal progress: [%d/%d]", event.KeysProvided, event.KeysRequired)
		if !event.Sealed {
			break
		}
	}
}

func GetVaultAddress() string {
	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr != "" {
		return vaultAddr
	}
	log.Printf("VAULT_ADDR not set, defaulting to %q", DEFAULT_VAULT_ADDR)
	return DEFAULT_VAULT_ADDR
}
