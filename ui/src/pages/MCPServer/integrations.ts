// Per-platform MCP integration catalog.
//
// CORTEX exposes a Streamable HTTP MCP endpoint authorized with a per-connection
// Bearer token. Each AI client expects that wired up in its own config format —
// this catalog produces the exact snippet + steps for every supported platform.

import claudeIcon from '../../icons/claude-color.svg';
import claudeCodeIcon from '../../icons/claudecode-color.svg';
import codexIcon from '../../icons/codex-color.svg';
import cursorIcon from '../../icons/cursor.svg';
import geminiCliIcon from '../../icons/geminicli-color.svg';
import githubCopilotIcon from '../../icons/githubcopilot.svg';
import kiroIcon from '../../icons/kiro-color.svg';
import antigravityIcon from '../../icons/antigravity-color.svg';
import openaiIcon from '../../icons/openai.svg';
import opencodeIcon from '../../icons/opencode.svg';
import windsurfIcon from '../../icons/windsurf.svg';

export type IntegrationGroup = 'cli' | 'web' | 'ide';

export interface PlatformIntegration {
  id: string;
  label: string;
  subtitle?: string;
  group: IntegrationGroup;
  icon?: string;
  language: 'json' | 'toml' | 'yaml' | 'bash' | 'text';
  /** File the snippet goes into (omitted for CLI commands). */
  file?: string;
  /** Web clients require a public HTTPS URL (localhost won't work). */
  requiresPublicUrl?: boolean;
  build: (endpoint: string, token: string) => string;
  steps: string[];
  note?: string;
}

const TOKEN_PLACEHOLDER = 'YOUR_CORTEX_TOKEN';

// Shared snippet builders ---------------------------------------------------

const mcpServersUrl = (endpoint: string, token: string, urlKey = 'url') =>
  JSON.stringify(
    { mcpServers: { cortex: { [urlKey]: endpoint, headers: { Authorization: `Bearer ${token}` } } } },
    null,
    2
  );

// Catalog -------------------------------------------------------------------

export const PLATFORMS: PlatformIntegration[] = [
  // ── AI Agent CLIs ──────────────────────────────────
  {
    id: 'claude-code',
    label: 'Claude Code',
    subtitle: 'Anthropic',
    group: 'cli',
    icon: claudeCodeIcon,
    language: 'bash',
    build: (endpoint, token) =>
      `claude mcp add --transport http cortex ${endpoint} \\\n  --header "Authorization: Bearer ${token}"`,
    steps: [
      'Run the command in your terminal (from any directory; use --scope project to scope it to a repo).',
      'Verify with: claude mcp list — "cortex" should report Connected.',
      'In a session, the cortex_get_context tool loads this project\'s memory.',
    ],
  },
  {
    id: 'codex',
    label: 'Codex',
    subtitle: 'OpenAI',
    group: 'cli',
    icon: codexIcon,
    language: 'toml',
    file: '~/.codex/config.toml',
    build: (endpoint, token) =>
      `[mcp_servers.cortex]\ncommand = "npx"\nargs = [\n  "-y", "mcp-remote",\n  "${endpoint}",\n  "--header", "Authorization: Bearer ${token}"\n]`,
    steps: [
      'Codex CLI talks to MCP servers over stdio, so we bridge to CORTEX with mcp-remote.',
      'Add the block to ~/.codex/config.toml (requires Node/npx installed).',
      'Restart Codex; the cortex_* tools become available.',
    ],
    note: 'Needs Node.js for the npx mcp-remote bridge.',
  },
  {
    id: 'gemini-cli',
    label: 'Gemini CLI',
    subtitle: 'Google',
    group: 'cli',
    icon: geminiCliIcon,
    language: 'json',
    file: '~/.gemini/settings.json',
    build: (endpoint, token) =>
      JSON.stringify(
        { mcpServers: { cortex: { httpUrl: endpoint, headers: { Authorization: `Bearer ${token}` } } } },
        null,
        2
      ),
    steps: [
      'Open ~/.gemini/settings.json (or .gemini/settings.json in your project).',
      'Merge the mcpServers block. Gemini CLI uses "httpUrl" for Streamable HTTP servers.',
      'Run `gemini` and check /mcp to confirm the server is listed.',
    ],
  },
  {
    id: 'github-copilot',
    label: 'GitHub Copilot',
    subtitle: 'GitHub',
    group: 'cli',
    icon: githubCopilotIcon,
    language: 'json',
    file: '.vscode/mcp.json',
    build: (endpoint, token) =>
      JSON.stringify(
        { servers: { cortex: { type: 'http', url: endpoint, headers: { Authorization: `Bearer ${token}` } } } },
        null,
        2
      ),
    steps: [
      'Create .vscode/mcp.json in your workspace.',
      'Open Copilot Chat and switch to Agent mode, then enable the "cortex" tools.',
      'Copilot reads the same MCP config as VS Code.',
    ],
  },
  {
    id: 'opencode',
    label: 'OpenCode',
    subtitle: 'OpenAI',
    group: 'cli',
    icon: opencodeIcon,
    language: 'json',
    file: 'opencode.json',
    build: (endpoint, token) =>
      JSON.stringify(
        {
          $schema: 'https://opencode.ai/config.json',
          mcp: { cortex: { type: 'remote', url: endpoint, headers: { Authorization: `Bearer ${token}` }, enabled: true } },
        },
        null,
        2
      ),
    steps: [
      'Add the mcp block to opencode.json (project) or ~/.config/opencode/opencode.json (global).',
      'OpenCode uses type "remote" for HTTP MCP servers.',
      'Restart OpenCode to load the cortex tools.',
    ],
  },


  // ── Web clients ────────────────────────────────────
  {
    id: 'claude-ai',
    label: 'Claude.ai',
    subtitle: 'Anthropic',
    group: 'web',
    icon: claudeIcon,
    language: 'text',
    requiresPublicUrl: true,
    build: (endpoint, token) =>
      `Server URL : ${endpoint.replace('http://127.0.0.1:8090', 'https://<your-public-tunnel>')}\nAuth header: Authorization: Bearer ${token}`,
    steps: [
      'Claude.ai only connects to MCP servers reachable over public HTTPS — expose CORTEX with a tunnel (e.g. cloudflared or ngrok) pointing at /mcp.',
      'Settings → Connectors → Add custom connector → paste the public HTTPS URL.',
      'Claude connectors authenticate via OAuth/headers depending on plan; supply the Bearer token if a header field is offered.',
    ],
    note: 'Requires a public HTTPS tunnel — localhost is not reachable from Claude.ai.',
  },
  {
    id: 'chatgpt',
    label: 'ChatGPT',
    subtitle: 'OpenAI',
    group: 'web',
    icon: openaiIcon,
    language: 'text',
    requiresPublicUrl: true,
    build: (endpoint, token) =>
      `Server URL : ${endpoint.replace('http://127.0.0.1:8090', 'https://<your-public-tunnel>')}\nAuth header: Authorization: Bearer ${token}`,
    steps: [
      'Enable Developer mode / Connectors in ChatGPT settings (availability depends on plan).',
      'Expose CORTEX over public HTTPS with a tunnel, then add it as a custom MCP connector using that URL.',
      'Provide the Bearer token as the connector authorization.',
    ],
    note: 'Requires a public HTTPS tunnel — localhost is not reachable from ChatGPT.',
  },


  // ── IDEs ───────────────────────────────────────────
  {
    id: 'cursor',
    label: 'Cursor',
    subtitle: 'Cursor',
    group: 'ide',
    icon: cursorIcon,
    language: 'json',
    file: '~/.cursor/mcp.json (or .cursor/mcp.json)',
    build: (endpoint, token) => mcpServersUrl(endpoint, token),
    steps: [
      'Create ~/.cursor/mcp.json (global) or .cursor/mcp.json (per project).',
      'Cursor Settings → MCP should show "cortex" with a green dot.',
      'Ask the agent to call cortex_get_context to load this project\'s memory.',
    ],
  },
  {
    id: 'antigravity',
    label: 'Antigravity',
    subtitle: 'Antigravity',
    group: 'ide',
    icon: antigravityIcon,
    language: 'json',
    file: 'Antigravity → Settings → MCP',
    build: (endpoint, token) => mcpServersUrl(endpoint, token),
    steps: [
      'Open Antigravity Settings → MCP / Tools → Add custom server (or edit its mcp config file).',
      'Paste the cortex server block.',
      'Reload; the agent can now read the project characterization and memory.',
    ],
  },
  {
    id: 'kiro',
    label: 'Kiro',
    subtitle: 'Kiro',
    group: 'ide',
    icon: kiroIcon,
    language: 'json',
    file: '.kiro/settings/mcp.json (or ~/.kiro/settings/mcp.json)',
    build: (endpoint, token) =>
      JSON.stringify(
        {
          mcpServers: {
            cortex: {
              url: endpoint,
              headers: { Authorization: `Bearer ${token}` },
              disabled: false,
              autoApprove: ['cortex_get_context', 'cortex_list_memories', 'cortex_get_tasks'],
            },
          },
        },
        null,
        2
      ),
    steps: [
      'Add the block to .kiro/settings/mcp.json (workspace) or ~/.kiro/settings/mcp.json (user).',
      'Kiro reconnects automatically; check the MCP Server panel.',
      'autoApprove lets read-only memory tools run without prompts.',
    ],
  },
  {
    id: 'windsurf',
    label: 'Windsurf',
    subtitle: 'Codeium',
    group: 'ide',
    icon: windsurfIcon,
    language: 'json',
    file: '~/.codeium/windsurf/mcp_config.json',
    build: (endpoint, token) => mcpServersUrl(endpoint, token, 'serverUrl'),
    steps: [
      'Open ~/.codeium/windsurf/mcp_config.json (Windsurf uses "serverUrl" for HTTP servers).',
      'Click "Refresh" in Windsurf\'s MCP / Cascade settings.',
      'The cortex tools appear in Cascade.',
    ],
  },
];

export const GROUP_LABELS: Record<IntegrationGroup, string> = {
  cli: 'AI Agent CLI',
  web: 'Web Clients',
  ide: 'IDE',
};

export function platformsByGroup(group: IntegrationGroup): PlatformIntegration[] {
  return PLATFORMS.filter((p) => p.group === group);
}

export function getPlatform(id: string): PlatformIntegration | undefined {
  return PLATFORMS.find((p) => p.id === id);
}

/** Builds the config snippet, using a placeholder when the token is unknown. */
export function buildSnippet(platform: PlatformIntegration, endpoint: string, token?: string): string {
  return platform.build(endpoint || 'http://127.0.0.1:8090/mcp', token || TOKEN_PLACEHOLDER);
}
