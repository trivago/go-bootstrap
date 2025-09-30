set shell := ["/usr/bin/env", "bash", "-euo", "pipefail", "-c"]

# ------------------------------------------------------------------------------

_default:
  @just -l

# Run all unittests
test:
  @go test -v ./...

# Create .envrc file to autmatically load required install via direnv and nix
init-nix:
  @hack/init-nix.sh
