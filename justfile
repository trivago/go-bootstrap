set shell := ["/usr/bin/env", "bash", "-euo", "pipefail", "-c"]

# ------------------------------------------------------------------------------

_default:
  @just -l

test:
  go test -v ./...

init-nix:
    #!/usr/bin/env bash
    set -euo pipefail

    cat <<-EOF > .envrc
    if ! has nix_direnv_version || ! nix_direnv_version 3.0.4; then
        source_url "https://raw.githubusercontent.com/nix-community/nix-direnv/3.0.4/direnvrc" "sha256-DzlYZ33mWF/Gs8DDeyjr8mnVmQGx7ASYqA5WlxwvBG4="
    fi
    use flake
    EOF

    direnv allow
