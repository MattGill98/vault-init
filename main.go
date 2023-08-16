package main

import (
	"log"
	"os"
	"time"

	"github.com/mattgill98/vault-init/pkg/client"
)

const (
	DEFAULT_VAULT_ADDR     = "http://127.0.0.1:8200"
	DEFAULT_CHECK_INTERVAL = 1000
)

func main() {
	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		log.Printf("VAULT_ADDR not set, defaulting to %q", DEFAULT_VAULT_ADDR)
		vaultAddr = DEFAULT_VAULT_ADDR
	}

	vaultClientConfig := &client.VaultClientConfig{
		Address: vaultAddr,
	}

	for {
		state, err := client.Health(*vaultClientConfig)
		if err != nil {
			log.Println(err)
			time.Sleep(DEFAULT_CHECK_INTERVAL * time.Millisecond)
			continue
		}

		switch true {
		case state.Active:
			log.Println("Vault is initialized and unsealed.")
		case state.Standby:
			log.Println("Vault is unsealed and in standby mode.")
		case state.Uninitialized:
			log.Println("Vault is not initialized.")
			initialize(*vaultClientConfig)
		case state.Sealed:
			log.Println("Vault is sealed.")
		default:
			log.Printf("Vault is in an unknown state. Status code: %d", state.StatusCode)
		}

		break
	}
}

func initialize(config client.VaultClientConfig) {
	log.Println("Initialising Vault...")

	response, err := client.Initialize(config)
	if err != nil {
		log.Fatalf("Initialization error: %q", err)
	}

	log.Println("Unsealing Vault...")
	for index, key := range response.Keys {
		event, err := client.Unseal(config, key)
		if err != nil {
			log.Printf("Failed to unseal using key [%d]", index)
			break
		}
		log.Printf("Unseal progress: [%d/%d]", event.KeysProvided, event.KeysRequired)
		if !event.Sealed {
			break
		}
	}
}
