disable_mlock = true
ui = true
cluster_addr  = "https://127.0.0.1:8201"
api_addr      = "https://127.0.0.1:8200"

listener "tcp" {
  tls_disable = 1
  address = "[::]:8200"
  cluster_address = "[::]:8201"
}

storage "file" {
  path = "/vault/data"
}
