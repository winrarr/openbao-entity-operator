# Backlog

These are real outcomes that are intentionally not part of the first vertical slice.

## Entity aliases

Add an `OpenBaoEntityAlias` resource that binds an auth-method mount accessor and alias name to a stable `OpenBaoEntity` ID, with safe orphan/delete behavior and status for the alias ID.

## Groups and membership

Add OpenBao-native group lifecycle and membership resources with explicit references, deterministic membership reconciliation, and protection against deleting groups that were only claimed for membership management.

## Installation chart

Add a Helm chart when the supported installation surface needs chart values or release packaging beyond the generated Kustomize bundle.
