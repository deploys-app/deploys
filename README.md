# deploys

The command-line interface for [deploys.app](https://deploys.app) — manage
projects, deployments, domains, routes, registries, GitHub integration, billing,
and access from your terminal or CI.

It is a thin client over the deploys.app API: every command maps to an API call,
prints the result, and exits non-zero on error.

## Install

**Go:**

```bash
go install github.com/deploys-app/deploys@latest
```

**Binaries:** download the archive for your OS/arch from the
[GitHub releases](https://github.com/deploys-app/deploys/releases), or from the
public bucket at `https://dl.deploys.app/deploys/` — `latest/` for the newest
release or `<tag>/` for a specific one (e.g.
`https://dl.deploys.app/deploys/latest/`).

**Container:**

```bash
docker run --rm -e DEPLOYS_TOKEN registry.deploys.app/public/deploys-cli <command>...
```

Image tags: `latest` is the newest tagged release, `<tag>` (e.g. `v1.2.3`) pins a
specific release, and `nightly` tracks the newest `master` build.

## Authentication

The CLI resolves credentials in this order:

1. **Service-account key (HTTP basic):** set both `DEPLOYS_AUTH_USER` and
   `DEPLOYS_AUTH_PASS`. The user is a service-account email
   (`<sid>@<project>.serviceaccount.deploys.app`) and the pass is its key secret.
   Best for CI.
2. **Bearer token:** set `DEPLOYS_TOKEN` to an API token.
3. **Application Default Credentials:** if neither of the above is set, the CLI
   falls back to Google ADC (e.g. `gcloud auth application-default login`) and
   uses the resulting access token.

Point the CLI at a non-production API with `DEPLOYS_ENDPOINT`.

## Usage

```
deploys <command> <subcommand> [flags]
```

Run `deploys` with no arguments for the top-level help, or any command with `-h`
to see its flags.

### Output formats

Every command accepts `-output` (default `table`):

```bash
deploys deployment list -project acme -location gke.cluster-rcf2          # table
deploys deployment list -project acme -location gke.cluster-rcf2 -oyaml   # yaml
deploys deployment list -project acme -location gke.cluster-rcf2 -ojson   # json
```

`-oyaml`, `-ojson`, and `-otable` are shorthands for `-output yaml|json|table`.

### Conventions

Unless noted otherwise, commands take `-project` and (for location-scoped
resources) `-location`; `-name` identifies the resource. The reference below
lists the flags specific to each command — `-project`, `-location`, `-name`, and
`-output` are omitted from the per-command notes where they follow this pattern.

## Examples

Deploy a web service:

```bash
deploys deployment deploy \
  -project acme -location gke.cluster-rcf2 \
  -name web -image registry.deploys.app/acme/web:latest -port 8080
```

A deploy is a **merge** — fields you don't pass keep the previous revision's
value, so you can ship config changes without re-specifying everything:

```bash
# add one env var and bump the memory limit; image/port/etc. are preserved
deploys deployment deploy -project acme -location gke.cluster-rcf2 -name web \
  -image registry.deploys.app/acme/web:v2 \
  -addEnv LOG_LEVEL=debug -memLimit 512Mi

# clear a previously-set TTL (explicit zero is sent; omitting -ttl leaves it)
deploys deployment deploy -project acme -location gke.cluster-rcf2 -name web \
  -image registry.deploys.app/acme/web:v2 -ttl 0
```

Just roll out a new image:

```bash
deploys deployment set image web -project acme -location gke.cluster-rcf2 \
  -image registry.deploys.app/acme/web:v2
```

Link a GitHub repository so its Actions deploy on push (see the
**Deploy from GitHub** guide in the deploys.app docs):

```bash
deploys github link -project acme -repository acme/web \
  -service-account ci -trigger all -production-branch main
```

## Command reference

`deployment` (aliases `deploy`, `d`) and the short aliases `ps`, `wi`, `sa`, `eg`
match the corresponding resource.

### me

- `get` — current user info.
- `authorized` `-permissions a,b` — check whether you hold the given permissions in `-project`.
- `permissions` — your effective permissions in `-project` (and whether you're an admin).
- `generate-token` `-project -permissions a,b [-ttl 900]` — mint a short-lived bearer token scoped to `-project` and a subset of your permissions (allowed: `dropbox.upload`, `site.publish`; TTL 60–3600s, default 900). Useful for handing a narrow credential to an automated tool (e.g. to `curl` a file upload to dropbox) without exposing a full token.

### location

- `list`, `get` `-id`.

### project

- `create` `-id -name -billingaccount`, `list`, `get`, `update` `-name -billingaccount`, `delete`, `usage`.

### role

- `create` `-role -name -permissions a,b`, `list`, `get` `-role`, `delete` `-role`.
- `grant` / `revoke` `-role -email`, `users`, `bind` `-email -roles a,b`.
- `permissions` — list every assignable permission string.

### deployment (`deploy`, `d`)

Lifecycle: `list`, `get` `-revision`, `delete`, `revisions`, `pause`, `resume`,
`rollback` `-revision`, `metrics` `-time-range 1h|6h|12h|1d`, `set image <name> -image ...`.

`deploy` — create or update a deployment (a merge; omitted flags are preserved).

| Flag | Notes |
|---|---|
| `-image` | container image (required for non-static types) |
| `-type` | `WebService` (default), `Worker`, `CronJob`, `TCPService`, `InternalTCPService`, `Static` |
| `-port` | container port |
| `-minReplicas` / `-maxReplicas` | autoscaling bounds |
| `-protocol` | WebService protocol: `http`, `https`, `h2c` |
| `-internal` | run the WebService as internal-only |
| `-env KEY=VAL` | set an env var; **repeatable**; replaces all env |
| `-addEnv KEY=VAL` | add an env var (repeatable); keeps the rest |
| `-removeEnv a,b` | remove env keys |
| `-envGroups` / `-addEnvGroups` / `-removeEnvGroups` | env groups (comma separated) |
| `-command a,b` / `-args a,b` | container entrypoint / args |
| `-workloadIdentity` / `-pullSecret` | identity / private-registry secret |
| `-diskName -diskMountPath -diskSubPath` | attach a disk (`Stateful`) |
| `-cpuRequest -memRequest -cpuLimit -memLimit` | resource requests/limits (e.g. `250m`, `512Mi`) |
| `-schedule` | cron schedule (`CronJob`) |
| `-ttl <seconds>` | auto-delete after N seconds; `-ttl 0` clears an existing TTL |
| `-requireGoogleLogin -allowedEmails a,b -allowedDomains a,b` | access control |
| `-mountData PATH=VAL` | mount file content at PATH; **repeatable** |
| `-sidecarsFile <path>` | YAML/JSON file describing sidecars |

> Static deployments are published by the GitHub build-and-deploy action (they
> carry a `site://` release reference), so `-type Static` isn't driven from here.

### route

- `list`, `get` `-domain -path`, `create` `-domain -path -target` (or `-deployment <name>` for the v1 shorthand), `delete` `-domain -path`.

### domain

- `create` `-domain -wildcard`, `get` `-domain`, `list`, `delete` `-domain`.
- `purgecache` `-domain` with `-file <path>` or `-prefix <path>`.

### waf

- `get`, `list`, `delete`.
- `set -f <spec.yaml>` `-description` — apply a WAF zone from a YAML spec (description, rules, limits). `-f` is required.
- `metrics` / `limitmetrics` `-time-range 1h|6h|12h|1d|7d|30d`.

### cache

Edge cache-override zone (one per project + location), applied at the CDN edge —
separate from the WAF (a role can hold `cache.*` without `waf.*`).

- `get`, `list`, `delete`.
- `set -f <spec.yaml>` `-description` — replace the zone's overrides from a YAML spec (`description`, `overrides`), all-or-nothing. `-f` is required.
- `metrics` `-time-range 1h|6h|12h|1d|7d|30d`.

### disk

- `create` `-size <Gi>`, `get`, `list`, `update` `-size <Gi>`, `delete`, `metrics` `-time-range 1h|6h|12h|1d|2d|7d|30d`.

### pullsecret (`ps`)

- `create` `-server -username -password`, `list`, `get`, `delete`.

### workloadidentity (`wi`)

- `create` `-gsa <google-service-account>`, `get`, `list`, `delete`.

### serviceaccount (`sa`)

- `create` `-id -name -description`, `list`, `get` `-id`, `update` `-id -name -description`, `delete` `-id`.
- `createkey` `-id`, `deletekey` `-id -secret`.

### envgroup (`eg`)

- `create` `-name -env KEY=VAL` (repeatable), `get` `-name`, `list`, `delete` `-name`.
- `update` `-name` with `-env` (replace all, repeatable), `-add-env` (repeatable), `-remove-env a,b`.

### registry

- `list`, `get` `-repository`, `tags` `-repository`, `manifests` `-repository`, `storage`.
- `delete` `-repository`, `deletemanifest` `-repository -digest`, `untag` `-repository -tag`.
- `metrics` `-time-range 7d|30d|90d`.

### github

- `link` `-repository owner/name -service-account <sid> -trigger all|branch|pr -production-branch <branch>`.
- `unlink` `-repository owner/name` (or `-repository-id <id>`).
- `update` `-repository owner/name` (or `-repository-id`) — change `-service-account`, `-trigger`, `-production-branch` in place; omitted flags are preserved.
- `list`.

### billing

- `create` / `update` `-id -name -tax-id -tax-name -tax-address`, `list`, `get` `-id`, `delete` `-id`.
- `report` `-id -range -projects a,b`, `skus`, `project` `-project`.
- `invoices` `-id`, `invoice` / `downloadinvoice` / `downloadreceipt` `-id <invoice id>`.

### email

- `send` `-from -from-name -to a,b -subject -type text|html -content` (or `-content-file <path>`), `list`.

### dropbox

- `list` `-after -before -limit`, `metrics` `-time-range 7d|30d|90d`.
- `upload` `-project -file -filename -ttl`, e.g. `deploys dropbox upload -project acme -file site.tar.gz -ttl 7`.
- `upload` reads `-file` (or stdin when omitted or `-`) and prints the public, short-lived download URL; `-ttl` is 1-7 days (default 1). Requires the `dropbox.upload` permission.
- `-after`/`-before` accept RFC 3339 or `YYYY-MM-DD`.

### auditlog

- `list` `-resource-type -actor -outcome success|failure -after -before -limit`.

### scheduler

Cron-driven outbound HTTP requests (project-scoped, location-less).

- `list` `-project`, `get` / `delete` / `pause` / `resume` / `trigger` `-project -name`.
- `create` `-project -name -schedule <cron> [-timezone <IANA>] [-method GET] -url <url> [-header KEY=VALUE ...] [-body ...] [-auth-type none|basic|bearer -auth-user <u> -auth-secret <s>] [-insecure-tls] [-paused]`.
- `update` `-project -name [flags]` — omitted flags are preserved; omit `-auth-secret` to keep the stored secret.
- `trigger` runs the job once now and prints the invocation result.
- `logs` `-project -name [-limit] [-after] [-before]` — recent invocations (timestamp, success/failed, latency, status).
- `-schedule` is a 5-field cron expression or an `@descriptor` (`@hourly`, `@every 30m`). `-after`/`-before` accept RFC 3339 or `YYYY-MM-DD`.

Example:

```bash
deploys scheduler create -project acme -name daily-health-check \
  -schedule "0 9 * * *" -timezone Asia/Bangkok \
  -method POST -url https://example.com/health \
  -header Content-Type=application/json -body '{"check":true}'
```

## Development

```bash
go build ./...
go vet ./...
go test ./...
```

Releases are cut by GoReleaser on tag push. The container image is built and
pushed to `registry.deploys.app/public/deploys-cli` (mirrored to
`asia-southeast1-docker.pkg.dev/deploys-app/public/cli`) by:

- the `Release` workflow on a tag — tagged with the release tag and `latest`;
- the `Build` workflow on `master` — tagged with the commit sha and `nightly`.
