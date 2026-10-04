<p align="center">
  <img src="ui/public/logowithname-readme.png" alt="CortexMind" width="380" />
</p>

# CortexMind

CortexMind is a local desktop workspace for repository context, code graphs, tasks,
knowledge, and shared AI session memory. It includes a Go daemon, a SolidJS UI, and
an MCP server. GitHub authentication uses Device Flow and opens the verification
page in your operating system's default browser.

## What works

- Import GitHub repositories, including private repositories authorized by your account.
- Create projects with a local directory and scan them while offline.
- Manually scan source files and build graphs of files, symbols, packages, and dependencies.
- View GitHub repository metadata, files, and README content through the authenticated backend.
- Save tasks, handoffs, vault entries, agent memories, and session digests locally.
- Generate a local system prompt from project memory and selected tasks and knowledge.
- Connect an MCP client to all projects, or restrict its token to one project.
- Scan your actual local clone, including uncommitted changes, before pushing.
- Resize the Windows desktop window from any edge or corner, maximize it, or use fullscreen.

Scanning, prompts, and digests currently use local heuristics. Provider preferences
can be saved securely, but cloud enrichment, semantic embeddings, automatic file
watching, repository export/push, and automatic updates are not active in the daemon.
Some settings are stored preferences for those future integrations. Daemon status
reports the active capabilities instead of treating a saved preference as running.

Project data is stored in SQLite. GitHub, Mistral, and local API credentials are kept
in the operating system credential store: Windows Credential Manager, macOS
Keychain, or the Linux Secret Service. GitHub requests send the GitHub token only to
GitHub; AI clients receive the project context allowed by their MCP connection.

## Requirements

- Go 1.25 or later.
- Node.js 18 or later and npm.
- Windows 10/11 with the Microsoft Edge WebView2 runtime for the desktop app.
- NSIS is optional and is used only to generate the Windows installer.
- Linux needs a working Secret Service/keyring for saved credentials.

## Install dependencies

From the repository root in PowerShell:

```powershell
./setup.ps1
```

Or install manually:

```powershell
go mod download
Set-Location ui
npm ci
Set-Location ..
```

## Build and run on Windows

```powershell
powershell -ExecutionPolicy Bypass -File .\build\build-desktop.ps1
```

The script builds the UI, embeds it in the daemon, and produces:

- `build/dist/CortexMind.exe` - native desktop app.
- `build/dist/CortexMind-Setup-0.1.0.exe` - installer, when NSIS is available.

Close older CortexMind and cortexd processes before opening the rebuilt app.
Drag a window edge or corner to resize. The titlebar provides minimize, maximize,
fullscreen, and close controls. GitHub sign-in opens your default browser; enter
the code shown in CortexMind and return to the app after approving it.

The desktop app verifies a running daemon using a credential-bound health challenge
before reusing it. An unrelated application occupying the configured port causes a
startup error rather than being loaded into the desktop window.

## Run the daemon with a browser UI

The daemon opens an authenticated UI link in your browser. The credential is passed
in the URL fragment, removed by the UI, and kept only in that tab's session storage.
Opening a plain URL without this credential cannot access the local API.

```powershell
go run ./cmd/cortexd

# Choose another loopback port.
go run ./cmd/cortexd serve --http 127.0.0.1:8100

# Run only the server, for an already configured MCP client.
go run ./cmd/cortexd serve --no-browser
```

`build-windows.ps1` builds the separate browser-based daemon at `CortexMind.exe` in
the repository root. Use `build/build-desktop.ps1` for the native desktop window.

There is no SQLite admin web page, superuser command, or admin OAuth setup.
GitHub sign-in uses the public client ID included in the build and requires no
client secret.

## Develop the UI

Use two terminals. Set an explicit development origin so the daemon accepts requests
forwarded by Vite, then let cortexd open the authenticated development URL.

Terminal 1:

```powershell
Set-Location ui
npm run dev
```

Terminal 2, from the repository root:

```powershell
$env:CORTEX_DEV_ORIGIN = 'http://localhost:3000'
go run ./cmd/cortexd
```

Vite proxies `/api` and `/mcp` to port 47831. If the daemon uses another port, set
`CORTEX_API_TARGET` before starting Vite, for example `http://127.0.0.1:8100`.
Use relative API URLs in the production UI; it automatically uses the daemon's
configured port. Remove `CORTEX_DEV_ORIGIN` for production runs.

```powershell
Remove-Item Env:CORTEX_DEV_ORIGIN -ErrorAction SilentlyContinue
```

The authenticated launch link is private. Do not share it or put its credential in
source control. If session storage is unavailable, the UI can still use the credential
in memory, but a full reload needs a new authenticated launch link.

## Configure the daemon

`cortex.yaml` is read from the current directory, `.cortex/`, or `~/.cortex/`.
Environment variables override configuration values.

```yaml
server:
  port: 47831
  data_dir: ~/.cortex

scanner:
  max_file_size_kb: 500
  ignored_dirs: [node_modules, .git, dist, build, target, __pycache__, .venv, vendor]

log_level: info
```

| Variable | Purpose |
|---|---|
| `CORTEX_SERVER_PORT` | Local HTTP and MCP port |
| `CORTEX_DATA_DIR` | SQLite and cloned repository cache directory |
| `CORTEX_DEV_ORIGIN` | Explicit local Vite origin; for development only |
| `CORTEX_GITHUB_CLIENT_ID` | Override the public GitHub OAuth app client ID |
| `CORTEX_LOG_LEVEL` | Logging level |

MCP uses `/mcp` on `server.port`; the legacy `mcp_port` setting does not open a
second listener. The daemon binds to `127.0.0.1` and rejects unexpected Host and
Origin headers. API requests require the local API credential; MCP requests require
a separate connection token. JSON request bodies are limited to 2 MiB.

## Connect an MCP client on Windows

1. Start the desktop app or daemon and keep it running.
2. Create a local project or sign in with GitHub to import repositories. Scan the
   project if your agent needs its source graph.
3. Open **MCP Server > New Connection**. Choose **All projects** (the default), or
   a specific project to restrict access. Select your client, then create
   the connection. Settings' **New Connection** link opens this setup screen.
4. Copy the generated token and client configuration. The token is displayed only
   once. All-project connections cover current and future projects; project
   connections cover only the selected project. Existing tokens retain their scope;
   create a new connection and replace your client's token to enable all projects.
5. Save the configuration in the location shown by the app and restart your AI client.

For Codex, add this to `%USERPROFILE%\.codex\config.toml`:

```toml
[mcp_servers.cortex]
url = "http://127.0.0.1:47831/mcp"
http_headers = { Authorization = "Bearer YOUR_CORTEX_TOKEN" }
```

For VS Code / GitHub Copilot Agent mode, use `.vscode/mcp.json`:

```json
{
  "servers": {
    "cortex": {
      "type": "http",
      "url": "http://127.0.0.1:47831/mcp",
      "headers": { "Authorization": "Bearer YOUR_CORTEX_TOKEN" }
    }
  }
}
```

Replace the placeholder token. If you changed the daemon port, use the endpoint
shown by the app. Keep configurations containing tokens private and out of Git.
The formats are described in the [Codex configuration reference](https://developers.openai.com/codex/config-reference)
and [VS Code MCP reference](https://code.visualstudio.com/docs/agents/reference/mcp-configuration).

Verify the connection in PowerShell:

```powershell
$mcpToken = 'YOUR_CORTEX_TOKEN'
$headers = @{ Authorization = "Bearer $mcpToken"; Accept = 'application/json, text/event-stream' }
$body = @{ jsonrpc = '2.0'; id = 1; method = 'tools/list'; params = @{} } | ConvertTo-Json -Compress
(Invoke-RestMethod -Uri 'http://127.0.0.1:47831/mcp' -Method Post -Headers $headers -ContentType 'application/json' -Body $body).result.tools.name
```

Tools include `cortex_list_projects`, `cortex_scan_working_tree`,
`cortex_get_working_tree_changes`, `cortex_get_system_prompt`, `cortex_get_context`, `cortex_get_code_graph`,
`cortex_save_memory`, `cortex_list_memories`, `cortex_get_tasks`, and
`cortex_summarize_session`. Start a session by reading the system prompt and context;
save useful decisions and summarize the session before handing off work.
For all-project connections, call `cortex_list_projects` and pass the selected
`project_id` to project tools. Memories, tasks, graphs, and prompts stay separate
for each project. Restricted connections default to their bound project and reject
other IDs. Delete a connection in the app to revoke its token.

### Review a local clone before pushing

Install Git for Windows and restart CortexMind if Git was added to PATH while the
app was running. CortexMind must run on the same computer as the checkout.

In the Repository page, enter the absolute cloned folder under **Local checkout**
and choose **Scan local changes**. Alternatively, ask your MCP agent to call:

```json
{
  "name": "cortex_scan_working_tree",
  "arguments": {
    "project_id": "YOUR_PROJECT_ID",
    "repo_path": "C:\\Users\\you\\source\\your-repo",
    "include_diff": true
  }
}
```

For all-project connections, `project_id` can be omitted when `repo_path` uniquely
matches a project's saved path or GitHub `origin` remote. Otherwise, choose an ID
from `cortex_list_projects`. The folder must be the Git checkout root; GitHub
projects require a matching `origin` (HTTPS or SSH). Linked Git worktrees are supported.

The scan refreshes the file index and code graph from the local files, honors Git
ignore rules for untracked files, and remembers the checkout for subsequent scans.
It works with a private clone without contacting GitHub. Results include branch,
staged/unstaged/untracked file status, commits ahead/behind upstream, and local commit
subjects. When requested, tracked changes against HEAD and local commit changes
against upstream each include a patch capped at 64 KiB, with truncation flags.
Untracked file contents are not included in patches; the agent can read those files
in its workspace. Upstream comparisons use the last fetched local reference, so
they do not guarantee the current remote state.

Use `cortex_get_working_tree_changes` for a fresh review without rebuilding the
index. After the first scan, omit `repo_path` to use the saved checkout. The ordinary
**Scan Now** action also uses a registered checkout without pulling. This is an
on-demand review: it does not install a Git hook or automatically block pushes.
No fetch, pull, commit, or push is performed on the user's checkout.

A remote web client cannot reach your computer's loopback address. The default
server configuration supports clients running on the same computer.

## Reset application data

**Settings > Reset all data** deletes application records, imported and local project
entries, code graphs, indexes, prompts, provider settings, users, and saved memories.
It revokes all MCP connections, removes saved GitHub and Mistral credentials, and
clears the daemon's cloned repository cache and browser preferences. It cancels an
in-progress GitHub login and reports errors if any reset step fails. Database table
deletions run in a transaction.

User-selected project directories and their source files are preserved. The local
API credential remains in the operating system credential store so the app can
continue to work after reset. A reset cannot be undone.

## API overview

The app uses authenticated JSON REST endpoints. The generated ConnectRPC types are
retained in the source tree, but ConnectRPC services are not registered by this daemon.

| Endpoint | Purpose |
|---|---|
| `GET /api/health` | Public challenge response for verifying the local daemon |
| `GET /api/cortex/status` | Actual daemon readiness and capabilities |
| `/api/auth/*` | Session, GitHub Device Flow, offline profile, logout |
| `/api/projects` | List or create projects |
| `GET /api/projects/{id}` | Read a project |
| `GET /api/github/repositories/{id}/view` | GitHub metadata, files, and rendered README |
| `POST /api/github/sync` | Refresh imported repositories |
| `POST /api/cortex/scan/{id}` | Update a GitHub checkout or scan a local directory |
| `/api/cortex/code-graph/{id}` | Read or rebuild a source graph |
| `/api/cortex/system-prompt/{id}` | Read, generate, or save a prompt |
| `/api/cortex/session-digest/{id}` | Generate a session digest |
| `/api/cortex/session-digests/{id}` | List session digests |
| `/api/cortex/providers` | Read or save provider preferences; keys are redacted |
| `/api/cortex/mcp/connections` | List or create all-project or project connections |
| `POST /api/cortex/working-tree/{id}` | Scan and register a local checkout; report changes |
| `DELETE /api/cortex/mcp/connections/{id}` | Revoke a connection |
| `POST /api/cortex/reset` | Reset application data and revoke credentials |
| `POST /mcp` | MCP JSON-RPC with a separate scoped bearer token |

## Other desktop platforms

Build the native desktop shell on its target operating system. The headless daemon
is pure Go and can be cross-compiled separately.

```bash
# macOS, with Xcode Command Line Tools installed
./build/build-desktop.sh

# Linux, with GTK3/WebKitGTK development libraries installed
./build/build-desktop.sh
```

From Windows, `build-linux.ps1` can build the Linux release with Docker Desktop.
See the build scripts for their distribution and packaging requirements.

## Troubleshooting

- **Unauthorized / HTTP 401 in the UI:** reopen through the desktop app or the link
  cortexd opens. A plain localhost URL does not contain the local credential.
- **Port occupied or desktop startup fails:** close an older daemon or choose another
  `server.port`. Check `cortexmind-desktop.log` in the configured data directory.
- **Credential store unavailable:** check Windows Credential Manager or the platform
  keyring. Startup errors are reported rather than exposing an unprotected API.
- **Private README or files fail to load:** reconnect GitHub and check access to the
  repository. README content uses GitHub's authenticated
  [repository contents API](https://docs.github.com/en/rest/repos/contents?apiVersion=2022-11-28)
  and is displayed in an isolated frame.
- **Scan cannot update a repository:** fix the reported network, permission, or Git
  error and retry. A failed pull is not silently scanned as current data.
- **Local project cannot scan:** provide an existing directory path when creating it.
- **MCP HTTP 401:** create a new connection and update the client's token.
- **MCP project mismatch:** use the bound project, or create an All projects connection.
- **All-project tool needs a project:** call `cortex_list_projects`, then supply
  `project_id`; checkout tools can also match an absolute `repo_path` automatically.
- **Checkout origin mismatch:** select the correct project and cloned folder.
- **Development origin rejected:** set `CORTEX_DEV_ORIGIN` to the exact Vite origin
  before starting cortexd. Use Vite's proxy for API requests.

## Verification

```powershell
go test ./...
go vet ./...
Set-Location ui
npm run build
```

Regression tests cover API authentication and origin checks, daemon identity,
request limits, reset rollback and token revocation, private repository content,
cancellable GitHub login, local scanning, index replacement, and failed Git updates.
