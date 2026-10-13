# Fork-only Valimail Monitor release

Temporary Under Armour build of the provider with
`primary_create_info.valimail_monitor` on `ultradns_zone`. It exists only in
`underarmour/terraform-provider-ultradns` and is not intended for upstream.
Remove it once upstream supports Valimail Monitor.

## What is fork-only

| Path | Purpose |
| --- | --- |
| `internal/forkzone/` | Adapter adding `valimailMonitor` on top of the pinned, unmodified SDK |
| `.goreleaser.fork.yml` | `linux_amd64` and `darwin_arm64` ZIPs plus `SHA256SUMS`, GitHub prerelease |
| `.github/workflows/fork-release.yml` | Runs on `v*-valimail.*` tags in this fork only |
| `.golangci.yml` | Excludes staticcheck SA1019 (deprecated `TestCase.Providers`) so CI lint passes |
| `FORK-RELEASE.md` | This file |

The upstream workflows and `.goreleaser.yml` are unchanged.

## Publishing

Versions follow the next upstream patch plus a `-valimail.N` suffix.

```sh
git tag v1.11.2-valimail.1
git push origin v1.11.2-valimail.1
```

The workflow runs the unit tests, then publishes a prerelease with:

- `terraform-provider-ultradns_1.11.2-valimail.1_linux_amd64.zip`
- `terraform-provider-ultradns_1.11.2-valimail.1_darwin_arm64.zip`
- `terraform-provider-ultradns_1.11.2-valimail.1_SHA256SUMS`

Artifacts are not GPG-signed. Consumers verify them with the SHA256 manifest.

## Consuming

Provider requirement. Prerelease versions must be pinned exactly:

```hcl
terraform {
  required_providers {
    ultradns = {
      source  = "registry.opentofu.org/underarmour/ultradns"
      version = "1.11.2-valimail.1"
    }
  }
}
```

GitHub Actions step, run before `tofu init`:

```yaml
- name: Install fork UltraDNS provider
  env:
    V: 1.11.2-valimail.1
  run: |
    set -euo pipefail
    M="$RUNNER_TEMP/tofu-mirror/registry.opentofu.org/underarmour/ultradns"
    mkdir -p "$M" && cd "$M"
    base="https://github.com/underarmour/terraform-provider-ultradns/releases/download/v$V"
    curl -fsSLO "$base/terraform-provider-ultradns_${V}_linux_amd64.zip"
    curl -fsSLO "$base/terraform-provider-ultradns_${V}_SHA256SUMS"
    sha256sum --ignore-missing -c "terraform-provider-ultradns_${V}_SHA256SUMS"
    rm "terraform-provider-ultradns_${V}_SHA256SUMS"
    cat > "$RUNNER_TEMP/tofurc" <<EOF
    provider_installation {
      filesystem_mirror {
        path    = "$RUNNER_TEMP/tofu-mirror"
        include = ["registry.opentofu.org/underarmour/ultradns"]
      }
      direct {
        exclude = ["registry.opentofu.org/underarmour/ultradns"]
      }
    }
    EOF
    echo "TF_CLI_CONFIG_FILE=$RUNNER_TEMP/tofurc" >> "$GITHUB_ENV"
```

Local use on macOS follows the same pattern with the `darwin_arm64` ZIP. The
lock file should carry hashes for both platforms. Generate them from a mirror
that contains both ZIPs:

```sh
tofu providers lock -fs-mirror=<mirror-dir> \
  -platform=linux_amd64 -platform=darwin_arm64 \
  registry.opentofu.org/underarmour/ultradns
```

### Switching existing state

Existing state records `registry.opentofu.org/ultradns/ultradns`. After
changing `source`, run once per state:

```sh
tofu state replace-provider \
  registry.opentofu.org/ultradns/ultradns \
  registry.opentofu.org/underarmour/ultradns
```

Run the reverse when returning to upstream.

### Behavior

- `valimail_monitor` defaults to `false`. A primary zone with Valimail Monitor
  enabled outside OpenTofu is disabled on apply unless the config sets `true`.
- Changes apply in place through PATCH. Any primary-zone PUT update is followed
  by a PATCH that sets the configured Valimail value again.
- Secondary and alias zones never send the field.

## Removal

1. Delete the fork-only paths listed above.
2. Point the `zone` imports in `internal/service` and `internal/zone` back at
   `github.com/ultradns/ultradns-go-sdk/pkg/zone`.
3. Remove `valimail_monitor` from `internal/zone`, or adopt the upstream field.
4. In consumers, restore `source = "ultradns/ultradns"`, run the reverse
   `replace-provider`, and remove the mirror step.
