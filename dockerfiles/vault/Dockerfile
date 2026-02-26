# Vault in production mode with Raft storage (see vault/vault.hcl).
# After first start, init and unseal Vault (see library/vault_restart.md).
FROM hashicorp/vault:1.18

RUN mkdir -p /vault/config
COPY vault/vault.hcl /vault/config/vault.hcl

ENTRYPOINT ["vault"]
CMD ["server", "-config=/vault/config/vault.hcl"]
