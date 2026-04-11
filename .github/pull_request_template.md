## Summary

- what changed
- why it changed
- user, operator, or contributor impact

## Validation

- [ ] `go mod tidy`
- [ ] `go build ./...`
- [ ] `go test ./... -v -race -count=1` or the equivalent platform-constrained helper path
- [ ] `go vet ./...`
- [ ] `golangci-lint run ./...`
- [ ] `make build`
- [ ] `cd web && npm run build` (if dashboard or API surfaces changed)
- [ ] `make release-gate` (required for release, packaging, tenant, migration, plugin, or docs changes)

## Compatibility And Operations

- API, state, or plugin compatibility impact:
- install, upgrade, or rollback impact:
- tenant isolation or security considerations:
- documentation or example updates included:

## Notes

- follow-up work
- known limitations
