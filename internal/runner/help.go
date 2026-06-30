package runner

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// IsHelpArg reports whether s requests help instead of naming a command. It
// covers the word "help" and the -h/-help/--help flag spellings, so help works
// the same at every level (top, group, and subcommand).
func IsHelpArg(s string) bool {
	switch s {
	case "help", "-h", "-help", "--help":
		return true
	}
	return false
}

// command describes a top-level command group for help output. The registry
// below is the single source of truth for `deploys help`, `deploys <group>
// help`, and each subcommand's usage banner, so the three stay in sync.
type command struct {
	name    string
	aliases []string
	short   string
	subs    []subcommand
}

// subcommand describes one leaf command. args is an optional positional/flag
// hint shown in the usage line; hidden keeps an entry out of the group listing
// while still feeding its description to the subcommand banner (e.g. the
// internal "set image" leaf, listed to users as "set").
type subcommand struct {
	name    string
	aliases []string
	args    string
	short   string
	hidden  bool
}

// commands is the full CLI surface, in the order shown by the top-level help.
//
// INVARIANT: this must mirror the dispatch switches (Run in runner.go for the
// groups, and each group method's `switch args[0]` for the leaves). Adding a
// subcommand means adding both a `case` and an entry here; otherwise its -h
// banner renders without a description. TestRegistryDescriptions guards that
// every listed entry is described, but it cannot see a `case` that was never
// listed — keep them in sync by hand.
var commands = []command{
	{
		name:  "auth",
		short: "log in and manage stored accounts",
		subs: []subcommand{
			{name: "login", short: "sign in via the browser and store the account"},
			{name: "logout", args: "[-account email] [-all]", short: "revoke and remove a stored account"},
			{name: "status", args: "[-endpoint url]", short: "show the credential in effect and its expiry"},
			{name: "list", short: "list stored accounts grouped by endpoint"},
			{name: "switch", args: "-account email [-endpoint url]", short: "change the active account for an endpoint"},
			{name: "token", args: "[-endpoint url] [-force]", short: "print the resolved bearer token (for scripts)"},
		},
	},
	{
		name:  "me",
		short: "identity and access for the current credential",
		subs: []subcommand{
			{name: "get", short: "show the current authenticated identity"},
			{name: "authorized", args: "-project p -permissions a,b", short: "check whether you hold the given permissions in a project"},
			{name: "permissions", args: "-project p", short: "list your effective permissions in a project"},
			{name: "generate-token", aliases: []string{"generateToken"}, args: "-project p -permissions a,b [-ttl 900] [-label l]", short: "mint a short-lived, scope-limited bearer token"},
			{name: "list-tokens", aliases: []string{"listTokens"}, args: "-project p", short: "list your active scoped tokens for a project"},
			{name: "revoke-token", aliases: []string{"revokeToken"}, args: "-project p -id t", short: "revoke one of your scoped tokens by id"},
		},
	},
	{
		name:  "billing",
		short: "billing accounts, reports, and invoices",
		subs: []subcommand{
			{name: "create", args: "-name <n> [-type individual|company -tax-id <t> -tax-name <n> -tax-address <a>]", short: "create a billing account"},
			{name: "list", short: "list billing accounts"},
			{name: "get", args: "-id <id>", short: "show a billing account"},
			{name: "update", args: "-id <id> [-name <n> -type individual|company -tax-id <t> -tax-name <n> -tax-address <a>]", short: "update a billing account"},
			{name: "delete", args: "-id <id>", short: "delete a billing account"},
			{name: "report", args: "-id <id> -range <r> [-projects a,b]", short: "usage/cost report for a billing account"},
			{name: "skus", short: "list billable SKUs and their prices"},
			{name: "project", args: "-project p", short: "show a project's billing account"},
			{name: "invoices", args: "-id <id>", short: "list invoices for a billing account"},
			{name: "invoice", args: "-id <invoice id>", short: "show an invoice"},
			{name: "downloadinvoice", args: "-id <invoice id>", short: "download an invoice PDF"},
			{name: "downloadreceipt", args: "-id <invoice id>", short: "download a receipt PDF"},
		},
	},
	{
		name:  "location",
		short: "available deployment locations",
		subs: []subcommand{
			{name: "list", short: "list available locations"},
			{name: "get", args: "-id <id>", short: "show a location"},
		},
	},
	{
		name:  "project",
		short: "projects, the top-level container for resources",
		subs: []subcommand{
			{name: "create", args: "-id -name -billingaccount", short: "create a project"},
			{name: "list", short: "list your projects"},
			{name: "get", short: "show a project"},
			{name: "update", args: "[-name] [-billingaccount]", short: "update a project's name or billing account"},
			{name: "delete", short: "delete a project"},
			{name: "usage", short: "show a project's resource usage"},
		},
	},
	{
		name:  "role",
		short: "custom roles and role bindings (IAM)",
		subs: []subcommand{
			{name: "create", args: "-role -name -permissions a,b", short: "create a custom role"},
			{name: "list", short: "list roles"},
			{name: "get", args: "-role", short: "show a role"},
			{name: "delete", args: "-role", short: "delete a role"},
			{name: "grant", args: "-role -email", short: "grant a role to a user"},
			{name: "revoke", args: "-role -email", short: "revoke a role from a user"},
			{name: "users", short: "list users and their roles"},
			{name: "bind", args: "-email -roles a,b", short: "set a user's roles"},
			{name: "permissions", short: "list every assignable permission"},
		},
	},
	{
		name:    "deployment",
		aliases: []string{"deploy", "d"},
		short:   "deployments and their lifecycle",
		subs: []subcommand{
			{name: "list", short: "list deployments"},
			{name: "get", args: "[-revision n]", short: "show a deployment (optionally a specific revision)"},
			{name: "deploy", short: "create or update a deployment (a merge over the previous revision)"},
			{name: "delete", short: "delete a deployment"},
			{name: "revisions", short: "list a deployment's revisions"},
			{name: "pause", short: "pause a deployment"},
			{name: "resume", short: "resume a paused deployment"},
			{name: "restart", short: "restart a deployment"},
			{name: "rollback", args: "-revision n", short: "roll back to a previous revision"},
			{name: "metrics", args: "[-time-range 1h|6h|12h|1d]", short: "show deployment metrics"},
			{name: "status", short: "show pod health and failure reasons"},
			{name: "logs", args: "[-pod p] [-previous] [-tail n] [-follow]", short: "read a bounded snapshot of recent container logs"},
			{name: "extend-ttl", args: "-name n -ttl s", short: "re-stamp a preview's auto-delete window to now+ttl (keep-alive)"},
			// "set" is the user-facing listing; "set image" is the hidden leaf that
			// backs its banner. They share wording so the listing and banner agree.
			{name: "set", short: "roll out a new image (set image <name> -image <ref>)"},
			{name: "set image", args: "<name> -image <ref>", short: "roll out a new image for a deployment", hidden: true},
		},
	},
	{
		name:    "error",
		aliases: []string{"errors"},
		short:   "detected application error issues (Sentry-lite)",
		subs: []subcommand{
			{name: "list", args: "[-name <deployment>] [-status open|resolved|muted|all] [-sort lastSeen|firstSeen|count] [-limit n] [-cursor c]", short: "list error issues (omit -name for a project-wide view)"},
			{name: "get", args: "-name <deployment> -id <id>", short: "show an error issue with its sample stack and recent occurrences"},
			{name: "update", args: "-name <deployment> -id <id> -status resolved|open|muted", short: "change an error issue's triage status (resolve, reopen, or mute)"},
			{name: "report", args: "-name <deployment> -type <class> [-kind] [-title] [-sample]", short: "report a single application error directly"},
		},
	},
	{
		name:  "domain",
		short: "custom domains and edge cache",
		subs: []subcommand{
			{name: "create", args: "-domain [-wildcard]", short: "create a custom domain"},
			{name: "get", args: "-domain", short: "show a domain"},
			{name: "list", short: "list domains"},
			{name: "delete", args: "-domain", short: "delete a domain"},
			{name: "purgecache", args: "-domain (-file <path> | -prefix <path>)", short: "purge edge cache for a domain"},
		},
	},
	{
		name:  "route",
		short: "HTTP routes mapping a domain/path to a target",
		subs: []subcommand{
			{name: "create", args: "-domain -path (-target <t> | -deployment <name>) [-host <h>]", short: "create a route"},
			{name: "get", args: "-domain -path", short: "show a route"},
			{name: "list", short: "list routes"},
			{name: "delete", args: "-domain -path", short: "delete a route"},
		},
	},
	{
		name:  "waf",
		short: "web application firewall zones",
		subs: []subcommand{
			{name: "get", short: "show the WAF zone"},
			{name: "list", short: "list WAF zones in a project"},
			{name: "set", args: "-f <spec.yaml> [-description]", short: "apply a WAF zone from a YAML spec"},
			{name: "delete", short: "delete the WAF zone"},
			{name: "metrics", args: "[-time-range 1h|6h|12h|1d|7d|30d]", short: "show WAF request metrics"},
			{name: "limitmetrics", args: "[-time-range 1h|6h|12h|1d|7d|30d]", short: "show WAF rate-limit metrics"},
		},
	},
	{
		name:  "cache",
		short: "edge cache-override zones (separate from the WAF)",
		subs: []subcommand{
			{name: "get", short: "show the cache-override zone"},
			{name: "list", short: "list cache zones in a project"},
			{name: "set", args: "-f <spec.yaml> [-description]", short: "replace the cache zone's overrides from a YAML spec"},
			{name: "delete", short: "delete the cache zone"},
			{name: "metrics", args: "[-time-range 1h|6h|12h|1d|7d|30d]", short: "show edge cache metrics"},
		},
	},
	{
		name:  "transform",
		short: "edge request/response transform zones (separate from the WAF)",
		subs: []subcommand{
			{name: "get", short: "show the transform zone"},
			{name: "list", short: "list transform zones in a project"},
			{name: "set", args: "-f <spec.yaml> [-description]", short: "replace the transform zone's rules from a YAML spec"},
			{name: "delete", short: "delete the transform zone"},
		},
	},
	{
		name:  "disk",
		short: "persistent disks for stateful deployments",
		subs: []subcommand{
			{name: "create", args: "-size <Gi>", short: "create a disk"},
			{name: "get", short: "show a disk"},
			{name: "list", short: "list disks"},
			{name: "update", args: "-size <Gi>", short: "resize a disk"},
			{name: "delete", short: "delete a disk"},
			{name: "metrics", args: "[-time-range 1h|6h|12h|1d|2d|7d|30d]", short: "show disk usage metrics"},
		},
	},
	{
		name:    "pullsecret",
		aliases: []string{"ps"},
		short:   "private-registry pull secrets",
		subs: []subcommand{
			{name: "create", args: "-server -username -password", short: "create a pull secret"},
			{name: "get", short: "show a pull secret"},
			{name: "list", short: "list pull secrets"},
			{name: "delete", short: "delete a pull secret"},
		},
	},
	{
		name:    "workloadidentity",
		aliases: []string{"wi"},
		short:   "workload identities bound to a Google service account",
		subs: []subcommand{
			{name: "create", args: "-gsa <google-sa>", short: "create a workload identity"},
			{name: "get", short: "show a workload identity"},
			{name: "list", short: "list workload identities"},
			{name: "delete", short: "delete a workload identity"},
		},
	},
	{
		name:    "serviceaccount",
		aliases: []string{"sa"},
		short:   "service accounts and their keys",
		subs: []subcommand{
			{name: "create", args: "-id -name -description", short: "create a service account"},
			{name: "list", short: "list service accounts"},
			{name: "get", args: "-id", short: "show a service account"},
			{name: "update", args: "-id [-name -description]", short: "update a service account"},
			{name: "delete", args: "-id", short: "delete a service account"},
			{name: "createkey", args: "-id", short: "create a service-account key"},
			{name: "deletekey", args: "-id -secret", short: "delete a service-account key"},
		},
	},
	{
		name:  "email",
		short: "send transactional email and list sent messages",
		subs: []subcommand{
			{name: "send", args: "-from -to a,b -subject -type text|html (-content | -content-file)", short: "send an email"},
			{name: "list", short: "list sent emails"},
		},
	},
	{
		name:  "registry",
		short: "container image registry",
		subs: []subcommand{
			{name: "list", short: "list repositories"},
			{name: "get", args: "-repository", short: "show a repository"},
			{name: "tags", args: "-repository", short: "list a repository's tags"},
			{name: "manifests", args: "-repository", short: "list a repository's manifests"},
			{name: "storage", short: "show registry storage usage"},
			{name: "delete", args: "-repository", short: "delete a repository"},
			{name: "deletemanifest", args: "-repository -digest", short: "delete a manifest by digest"},
			{name: "untag", args: "-repository -tag", short: "remove a tag"},
			{name: "gc", args: "[-dry-run]", short: "garbage-collect manifests no deployment uses"},
			{name: "metrics", args: "[-time-range 7d|30d|90d]", short: "show registry storage metrics"},
		},
	},
	{
		name:    "envgroup",
		aliases: []string{"eg"},
		short:   "reusable environment-variable groups",
		subs: []subcommand{
			{name: "create", args: "-name -env KEY=VAL", short: "create an env group"},
			{name: "get", args: "-name", short: "show an env group"},
			{name: "list", short: "list env groups"},
			{name: "update", args: "-name [-env|-add-env|-remove-env]", short: "update an env group's variables"},
			{name: "delete", args: "-name", short: "delete an env group"},
		},
	},
	{
		name:  "auditlog",
		short: "query the project audit log",
		subs: []subcommand{
			{name: "list", args: "[-resource-type -actor -outcome -after -before -limit]", short: "list audit-log entries"},
		},
	},
	{
		name:  "dropbox",
		short: "short-lived file uploads (dropbox)",
		subs: []subcommand{
			{name: "list", args: "[-after -before -limit]", short: "list dropbox files"},
			{name: "metrics", args: "[-time-range 7d|30d|90d]", short: "show dropbox usage metrics"},
			{name: "upload", args: "-file <path> [-filename -ttl]", short: "upload a file and print a short-lived download URL"},
			{name: "upload-url", args: "-project <sid> [-filename -content-type -min-size -max-size -ttl -expires]", short: "mint a signed upload URL to hand off (recipient PUTs the file, no token needed)"},
		},
	},
	{
		name:  "github",
		short: "link GitHub repositories for build-and-deploy",
		subs: []subcommand{
			{name: "link", args: "-repository owner/name -service-account <sid> [-trigger -production-branch]", short: "link a GitHub repository"},
			{name: "unlink", args: "-repository owner/name | -repository-id <id>", short: "unlink a GitHub repository"},
			{name: "update", args: "-repository owner/name [-service-account -trigger -production-branch]", short: "change a linked repository's settings"},
			{name: "list", short: "list linked repositories"},
		},
	},
	{
		name:  "site",
		short: "publish static sites from the local filesystem",
		subs: []subcommand{
			{name: "publish", args: "-name -dir <path> [-environment -spa -notFound]", short: "publish a static site from a local directory (prints a site:// ref)"},
			{name: "deploy", args: "-name -dir <path> [-location] [-environment -spa -notFound]", short: "publish and deploy a static site as a permanent deployment"},
			{name: "preview", args: "-name -dir <path> [-ttl s] [-force] [-environment -spa -notFound]", short: "publish and deploy a throwaway preview (auto-deletes at ttl; -force to overwrite a non-preview)"},
		},
	},
	{
		name:  "scheduler",
		short: "run HTTP requests on a cron schedule",
		subs: []subcommand{
			{name: "create", args: "-name -schedule <cron> -url <url> [-method -timezone -header KEY=VALUE -body -auth-type -auth-user -auth-secret -insecure-tls -paused]", short: "create a scheduled HTTP request job"},
			{name: "get", args: "-name", short: "show a scheduled job"},
			{name: "list", short: "list scheduled jobs"},
			{name: "update", args: "-name [flags]", short: "update a scheduled job (omitted flags are preserved)"},
			{name: "delete", args: "-name", short: "delete a scheduled job"},
			{name: "pause", args: "-name", short: "pause a scheduled job"},
			{name: "resume", args: "-name", short: "resume a paused scheduled job"},
			{name: "trigger", args: "-name", short: "run a scheduled job once now and print the result"},
			{name: "logs", args: "-name [-limit -after -before]", short: "show a job's recent invocations"},
		},
	},
	{
		name:  "notification",
		short: "deliver project changes to webhook/discord channels, or pull them",
		subs: []subcommand{
			{name: "create", args: "-name -type <webhook|discord|pull> [-url -secret -insecure-tls -pull-ttl -event -outcome -disabled]", short: "create a notification channel"},
			{name: "get", args: "-name", short: "show a notification channel"},
			{name: "list", short: "list notification channels"},
			{name: "update", args: "-name [flags]", short: "update a notification channel (omitted flags are preserved)"},
			{name: "delete", args: "-name", short: "delete a notification channel"},
			{name: "test", args: "-name", short: "deliver a synthetic change now and print the result"},
			{name: "deliveries", args: "-name [-limit -after -before]", short: "show recent change deliveries"},
			{name: "pull", args: "-name [-ack -limit -follow -poll -interval]", short: "fetch a pull channel's change events (ack to advance; -follow streams over SSE)"},
		},
	},
}

// lookupCommand finds a group by its canonical name or any alias.
func lookupCommand(name string) *command {
	for i := range commands {
		if commands[i].name == name {
			return &commands[i]
		}
		for _, a := range commands[i].aliases {
			if a == name {
				return &commands[i]
			}
		}
	}
	return nil
}

// lookupSub finds a leaf by its name or alias within a group.
func (c *command) lookupSub(name string) *subcommand {
	if c == nil {
		return nil
	}
	for i := range c.subs {
		if c.subs[i].name == name {
			return &c.subs[i]
		}
		for _, a := range c.subs[i].aliases {
			if a == name {
				return &c.subs[i]
			}
		}
	}
	return nil
}

// PrintUsage writes the top-level help: every command group with the names of
// its subcommands, the global flags, and the auth environment variables.
func PrintUsage(w io.Writer) {
	fmt.Fprint(w, "deploys.app cli\n\n")
	fmt.Fprint(w, "Usage:\n  deploys <command> <subcommand> [flags]\n\n")

	fmt.Fprintln(w, "Commands:")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	// login/logout are the primary verbs (aliases of auth login/logout); list
	// them first, above the resource groups.
	fmt.Fprintf(tw, "  %s\t%s\n", "login", "sign in via the browser (alias of auth login)")
	fmt.Fprintf(tw, "  %s\t%s\n", "logout", "remove a stored account (alias of auth logout)")
	for _, c := range commands {
		name := c.name
		if len(c.aliases) > 0 {
			name += ", " + strings.Join(c.aliases, ", ")
		}
		var subs []string
		for _, s := range c.subs {
			if s.hidden {
				continue
			}
			subs = append(subs, s.name)
		}
		fmt.Fprintf(tw, "  %s\t%s\n", name, strings.Join(subs, ", "))
	}
	// version and check-update are standalone, API-less utility commands, not
	// groups with subcommands, so they live outside the registry above.
	fmt.Fprintf(tw, "  %s\t%s\n", "version", "print the cli version")
	fmt.Fprintf(tw, "  %s\t%s\n", "check-update", "check whether a newer cli version is available")
	tw.Flush()

	fmt.Fprint(w, "\nFlags:\n")
	fmt.Fprint(w, "  -output table|yaml|json   output mode (or the -oyaml, -ojson, -otable shorthands)\n")
	fmt.Fprint(w, "  -account email            use a specific stored account for this command\n")

	fmt.Fprint(w, "\nEnvironment:\n")
	tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprint(tw, "  DEPLOYS_TOKEN\tbearer api token (overrides a stored login)\n")
	fmt.Fprint(tw, "  DEPLOYS_AUTH_USER\tservice-account email (basic auth, with DEPLOYS_AUTH_PASS)\n")
	fmt.Fprint(tw, "  DEPLOYS_AUTH_PASS\tservice-account key secret\n")
	fmt.Fprint(tw, "  DEPLOYS_ENDPOINT\toverride the api endpoint\n")
	fmt.Fprint(tw, "  DEPLOYS_ACCOUNT\tselect a stored account by email (same as -account)\n")
	fmt.Fprint(tw, "  DEPLOYS_AUTH_ENDPOINT\toverride the auth (login) server\n")
	fmt.Fprint(tw, "  DEPLOYS_CONFIG_DIR\toverride the config/credentials directory\n")
	tw.Flush()

	fmt.Fprint(w, "\nRun \"deploys <command> -h\" for a command's subcommands and flags.\n")
}

// writeLoginUsage writes the banner for the standalone `deploys login` verb.
func writeLoginUsage(w io.Writer) {
	fmt.Fprint(w, "login — sign in via the browser and store the account (alias of auth login)\n\n")
	fmt.Fprint(w, "Usage:\n  deploys login [-endpoint url] [-no-browser] [-port n] [-timeout 3m]\n\n")
	fmt.Fprint(w, "Opens a browser to authorize, then saves a credential under your config dir.\n")
	fmt.Fprint(w, "On a remote/SSH host, use -no-browser and forward the printed callback port.\n")
}

// writeLogoutUsage writes the banner for the standalone `deploys logout` verb.
func writeLogoutUsage(w io.Writer) {
	fmt.Fprint(w, "logout — revoke and remove a stored account (alias of auth logout)\n\n")
	fmt.Fprint(w, "Usage:\n  deploys logout [-account email] [-endpoint url] [-all] [-yes]\n\n")
	fmt.Fprint(w, "With no flags, logs out the active account for the endpoint.\n")
}

// writeGroupUsage writes a group's help: its description, aliases, and the list
// of subcommands with one-line descriptions.
func writeGroupUsage(w io.Writer, c *command) {
	fmt.Fprintf(w, "%s — %s\n\n", c.name, c.short)
	fmt.Fprintf(w, "Usage:\n  deploys %s <subcommand> [flags]\n", c.name)
	if len(c.aliases) > 0 {
		fmt.Fprintf(w, "\nAliases: %s\n", strings.Join(c.aliases, ", "))
	}

	fmt.Fprint(w, "\nSubcommands:\n")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, s := range c.subs {
		if s.hidden {
			continue
		}
		fmt.Fprintf(tw, "  %s\t%s\n", s.name, s.short)
	}
	tw.Flush()

	fmt.Fprintf(w, "\nRun \"deploys %s <subcommand> -h\" for a subcommand's flags.\n", c.name)
}

// writeSubUsage writes a leaf command's help: its description, a usage line, and
// the flag list (rendered from the live flag set, so it always matches what the
// command actually accepts).
func writeSubUsage(w io.Writer, f *flag.FlagSet, group, sub string) {
	c := lookupCommand(group)
	entry := c.lookupSub(sub)

	if entry != nil && entry.short != "" {
		fmt.Fprintf(w, "%s\n\n", entry.short)
	}

	usage := "Usage: deploys " + group + " " + sub
	if entry != nil && entry.args != "" {
		usage += " " + entry.args
	}
	usage += " [flags]"
	fmt.Fprintf(w, "%s\n\nFlags:\n", usage)

	f.SetOutput(w)
	f.PrintDefaults()
}

// topUsage prints the top-level help.
func (rn Runner) topUsage() error {
	PrintUsage(rn.output())
	return nil
}

// groupUsage prints a group's help. group is always a canonical name passed by
// the dispatcher, so the lookup never misses.
func (rn Runner) groupUsage(group string) error {
	c := lookupCommand(group)
	if c == nil {
		return rn.topUsage()
	}
	writeGroupUsage(rn.output(), c)
	return nil
}

// unknownSub reports an unrecognized subcommand and points at the group help.
func (rn Runner) unknownSub(group, sub string) error {
	return fmt.Errorf("deploys %s: unknown subcommand %q (run \"deploys %s -h\")", group, sub, group)
}

// subFlagSet builds the flag set for a leaf command: a named, ExitOnError set
// that prints writeSubUsage on -h/-help/--help. registerFlags binds the shared
// -output flag to this same Runner so the chosen output mode reaches print();
// a pointer receiver keeps that binding pointing at the caller's Runner.
func (rn *Runner) subFlagSet(group, sub string) *flag.FlagSet {
	f := flag.NewFlagSet("deploys "+group+" "+sub, flag.ExitOnError)
	f.SetOutput(rn.output())
	rn.registerFlags(f)
	f.Usage = func() { writeSubUsage(rn.output(), f, group, sub) }
	return f
}
