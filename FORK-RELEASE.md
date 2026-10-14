# Fork-only provider release

Temporary Under Armour build of the provider from the PR #64 branch
(`feat/webforward`): the `ultradns_webforward` resource and data source, and
the AAAA and web forward refresh-drift fixes. It uses the released UltraDNS Go
SDK pinned in `go.mod`.

This branch (`fork/release`) is `feat/webforward` plus the files below. It
exists only in `underarmour/terraform-provider-ultradns` and is never sent
upstream. Delete it once upstream releases PR #64.

| Path | Purpose |
| --- | --- |
| `.goreleaser.fork.yml` | `linux_amd64` and `darwin_arm64` ZIPs plus `SHA256SUMS`, GitHub prerelease |
| `.github/workflows/fork-release.yml` | Runs on `v*-webforward.*` tags in this fork only |
| `FORK-RELEASE.md` | This file |

The upstream workflows and `.goreleaser.yml` are unchanged.

## Updating

Merge the latest `feat/webforward` into `fork/release`, then tag the next build.

## Publishing

Versions follow the next upstream patch plus a `-webforward.N` suffix.

```sh
git tag v1.11.2-webforward.1
git push origin v1.11.2-webforward.1
```

The workflow runs the unit tests, then publishes a prerelease with:

- `terraform-provider-ultradns_1.11.2-webforward.1_linux_amd64.zip`
- `terraform-provider-ultradns_1.11.2-webforward.1_darwin_arm64.zip`
- `terraform-provider-ultradns_1.11.2-webforward.1_SHA256SUMS`

Artifacts are not GPG-signed. Consumers verify them with the SHA256 manifest.

## Consuming

Prerelease versions must be pinned exactly:

```hcl
terraform {
  required_providers {
    ultradns = {
      source  = "registry.opentofu.org/underarmour/ultradns"
      version = "1.11.2-webforward.1"
    }
  }
}
```

GitHub Actions step, run before `tofu init`:

```yaml
- name: Install fork UltraDNS provider
  env:
    V: 1.11.2-webforward.1
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

Local use on macOS follows the same pattern with the `darwin_arm64` ZIP. Record
hashes for both platforms from a mirror containing both ZIPs:

```sh
tofu providers lock -fs-mirror=<mirror-dir> \
  -platform=linux_amd64 -platform=darwin_arm64 \
  registry.opentofu.org/underarmour/ultradns
```

Existing state that records `registry.opentofu.org/ultradns/ultradns` needs one
migration per state:

```sh
tofu state replace-provider \
  registry.opentofu.org/ultradns/ultradns \
  registry.opentofu.org/underarmour/ultradns
```

## Removal

1. Delete the `fork/release` branch and its `v*-webforward.*` releases.
2. In consumers, restore `source = "ultradns/ultradns"` with the upstream
   version, run the reverse `replace-provider`, and remove the mirror step.
