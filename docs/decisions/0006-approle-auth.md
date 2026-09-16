# 0006 — Use typed AppRole authentication with external credential rotation

## Decision

Add AppRole as a third mutually exclusive `OpenBaoConnection` authentication
method. The API accepts a mount path plus same-namespace Secret references for
the role ID and Secret ID. The operator does not configure the OpenBao auth
method, create roles, or issue Secret IDs.

The typed client obtains a token through
`POST /v1/auth/<mount>/login`, renews renewable tokens through the existing
token renewal path, and rereads both credential references when a fresh login
is required. AppRole clients are cached across reconciliations for lease
handling; static token clients remain uncached so token Secret rotation is
observed immediately.

## Rationale

AppRole provides a practical machine-auth option for installations where
Kubernetes Auth is unavailable, while keeping the connection boundary explicit
and avoiding arbitrary OpenBao request payloads. Separate credential references
allow role IDs and Secret IDs to have independent rotation and access policy.

The operator must not silently own AppRole credential lifecycle. OpenBao
administrators or another credential process remain responsible for Secret ID
creation, expiry, use limits, and rotation. The operator only consumes the
current Kubernetes Secret values and never publishes them in status or logs.

## Verification boundary

- HTTP contract tests cover login body, lease renewal, unauthorized relogin,
  validation, namespace headers, and credential source rereads.
- Controller tests cover dynamic-client caching and configuration replacement.
- The focused Kind workflow covers a real AppRole-authenticated connection;
  controller state branches remain unit and HTTP test responsibilities.
