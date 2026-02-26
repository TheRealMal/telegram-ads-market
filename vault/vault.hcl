storage "raft" {
  path = "/vault/data"
  node_id = "vault_node_1"
}
listener "tcp" {
  address = "0.0.0.0:8200"
  cluster_address = "0.0.0.0:8201"
  tls_disable = "true" # Disable TLS for initial setup/testing
}
api_addr = "http://0.0.0.0:8200"
cluster_addr = "http://0.0.0.0:8201"
ui = true
