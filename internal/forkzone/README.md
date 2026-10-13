# Fork-only Valimail Monitor zone adapter

This package exists only in the Under Armour fork. It adds the
`valimailMonitor` zone API field, which the released UltraDNS Go SDK
does not expose.

It wraps `github.com/ultradns/ultradns-go-sdk/pkg/zone` at the version
pinned in `go.mod`. The SDK itself is unmodified; no SDK fork, release,
`replace` directive, or Go workspace is involved.

- SDK types are reused through aliases and embedding. `PrimaryZone` and
  `Response` add a nullable `ValimailMonitor *bool`; non-nil `false` is
  serialized explicitly.
- `UpdateZone` (PUT), `DeleteZone`, `ListZone`, `MigrateZoneAccount`, and
  non-Valimail creates delegate to the SDK service.
- `ReadZone`, primary `CreateZone`, and `PartialUpdateZone` (PATCH) call
  the SDK client directly with the extended types. Primary creation waits on
  asynchronous tasks with the same SDK task service and retry limits.

## Removal

When upstream supports Valimail Monitor, delete this directory and point the
`zone` imports back at `github.com/ultradns/ultradns-go-sdk/pkg/zone`.
