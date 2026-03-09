## Summary

<!-- Briefly describe what this PR changes and why. -->

## Checklist

- [ ] `make lint` passes locally
- [ ] `make test` passes locally
- [ ] If proto files changed, `make proto-generate` was run and the updated `gen/` files are included in this PR
- [ ] If proto files changed, `make proto-breaking` was checked for unintended breaking changes
- [ ] Architecture boundaries are respected (domain logic in Go, UI is presentation-only, no telemetry added)
- [ ] No files under `gen/` were edited by hand

## Related issues

<!-- Link any related issues, e.g. Closes #123 -->
