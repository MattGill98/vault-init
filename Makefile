.DEFAULT_GOAL := help

vault-container: ## Run a vault container to test against
	docker run --cap-add=IPC_LOCK --name vault-demo --rm -p 8200:8200 -v ./example/config/:/vault/config/ hashicorp/vault server

help: ## Show this help display
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
