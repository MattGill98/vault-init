package secret

import "github.com/mattgill98/vault-init/pkg/vault"

type KeyStorage interface {
	Persist(state vault.InitState)
	Fetch() vault.InitState
}
