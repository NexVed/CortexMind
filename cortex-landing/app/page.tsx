import {
  ArrowRight,
  Bell,
  Bot,
  BrainCircuit,
  CheckCircle2,
  ChevronDown,
  Circle,
  Code2,
  Command,
  Database,
  FolderGit2,
  Github,
  Network,
  Play,
  Plus,
  Search,
  ShieldCheck,
  Sparkles,
  Terminal,
  Workflow,
  Zap,
} from 'lucide-react';

const navigation = ['Platform', 'Solutions', 'Developers', 'Pricing'];

const agents = [
  { name: 'Code Agent', state: 'Running', tone: 'violet', icon: Code2 },
  { name: 'Research Agent', state: 'Idle', tone: 'blue', icon: Search },
  { name: 'UI Agent', state: 'Working', tone: 'pink', icon: Sparkles },
  { name: 'Deployment Agent', state: 'Ready', tone: 'mint', icon: Workflow },
];

const activity = [
  ['10:14', 'Generated landing page', 'UI Agent', 'Completed'],
  ['10:09', 'Scanned repository', 'Code Agent', 'Completed'],
  ['09:58', 'Fixed build errors', 'Debug Agent', 'Running'],
  ['09:42', 'Generated API docs', 'Documentation Agent', 'Completed'],
];

const capabilities = [
  { icon: Network, index: '01', title: 'Repository intelligence', description: 'Turn every repository into a living map of files, relationships, decisions, and dependencies.' },
  { icon: BrainCircuit, index: '02', title: 'Durable agent memory', description: 'Give every new agent the context of the work before it, without making your team repeat itself.' },
  { icon: Workflow, index: '03', title: 'Workflows that ship', description: 'Coordinate specialized agents across the work that moves an idea from first prompt to production.' },
];

export default function Home() {
  return (
    <main className="landing-shell">
      <div className="aurora aurora-one" aria-hidden="true" />
      <div className="aurora aurora-two" aria-hidden="true" />
      <div className="dot-field" aria-hidden="true" />

      <nav className="nav container" aria-label="Main navigation">
        <a className="brand" href="#top" aria-label="cortexMind home">
          <span className="brand-mark"><BrainCircuit size={18} strokeWidth={2.25} /></span>
          <span className="brand-cortex">cortex</span><span className="brand-mind">Mind</span>
        </a>
        <div className="nav-links">
          {navigation.map((item) => <a key={item} href={`#${item.toLowerCase()}`}>{item}</a>)}
        </div>
        <a className="nav-cta" href="#workspace">Start Building <ArrowRight size={15} /></a>
      </nav>

      <section id="top" className="hero container">
        <div className="hero-copy">
          <div className="hero-badge reveal"><span className="badge-spark"><Sparkles size={12} /></span> AI Workspace for Every Developer</div>
          <h1 className="hero-title reveal delay-1">Build with<br /><em>Intelligent</em><br />Agents</h1>
          <p className="hero-subtitle reveal delay-2">cortexMind is an AI-native development workspace where autonomous agents collaborate, understand your codebase, automate repetitive tasks, and accelerate software delivery from idea to deployment.</p>
          <div className="hero-actions reveal delay-3">
            <a className="button button-primary" href="#workspace">Start Building <ArrowRight size={17} /></a>
            <a className="watch-link" href="#workspace"><span className="play-button"><Play size={12} fill="currentColor" /></span> Watch Demo</a>
          </div>
          <div className="trust-row reveal delay-3"><span><ShieldCheck size={15} /> Local-first context</span><span><Database size={15} /> Your data, connected</span></div>
        </div>
        <div className="hero-signal" aria-hidden="true"><span>AGENT-NATIVE</span><i /> <span>REPOSITORY-AWARE</span><i /> <span>LOCAL-FIRST</span></div>
      </section>

      <section id="workspace" className="dashboard-stage container" aria-labelledby="workspace-title">
        <div className="dashboard-halo" aria-hidden="true" />
        <div className="workspace-caption"><span>cortexMind / workspace</span><span>LIVE ENVIRONMENT <i /></span></div>
        <div className="dashboard-shell reveal delay-2">
          <aside className="workspace-sidebar" aria-label="Workspace navigation">
            <div className="sidebar-logo"><BrainCircuit size={17} /><span>cortex<span>Mind</span></span></div>
            <div className="workspace-project"><span className="project-avatar">C</span><span>cortexMind</span><ChevronDown size={13} /></div>
            <div className="sidebar-nav">
              <a className="active" href="#workspace"><Sparkles size={16} /> Home</a>
              <a href="#platform"><FolderGit2 size={16} /> Projects <span>18</span></a>
              <a href="#developers"><Github size={16} /> Repositories</a>
              <a href="#platform"><Bot size={16} /> Agents <span>42</span></a>
              <a href="#solutions"><BrainCircuit size={16} /> Knowledge</a>
              <a href="#developers"><Terminal size={16} /> Terminal</a>
              <a href="#solutions"><Workflow size={16} /> Automations</a>
            </div>
            <a className="sidebar-bottom" href="#pricing"><Circle size={15} /> Settings</a>
          </aside>

          <div className="workspace-main">
            <div className="workspace-topbar">
              <div className="command-search"><Search size={15} /><span>Search repos, files, prompts...</span><kbd><Command size={10} /> K</kbd></div>
              <div className="topbar-actions"><button className="deploy-button"><Zap size={13} /> Deploy</button><button aria-label="Notifications" className="icon-button"><Bell size={16} /><i /></button><span className="user-avatar">UR</span></div>
            </div>

            <div className="workspace-content">
              <div className="welcome-row">
                <div><span className="micro-label">MONDAY, JULY 14</span><h2 id="workspace-title">Welcome back, Utkarsh</h2><p>Your development system is moving.</p></div>
                <button className="new-button"><Plus size={15} /> New project</button>
              </div>
              <div className="action-pills" aria-label="Quick actions"><button>New Project</button><button>Create Agent</button><button>Import Repo</button><button>Run Workflow</button><button>Generate UI</button></div>

              <div className="dashboard-grid">
                <section className="task-card card">
                  <div className="card-heading"><div><span className="micro-label">ACTIVE WORKSPACE</span><h3>148 <span>running AI tasks</span></h3></div><span className="task-live"><i /> live</span></div>
                  <div className="task-stats"><span><strong>+42</strong> Agents</span><span><strong>18</strong> Projects</span><span><strong>6</strong> Workflows</span></div>
                  <ActivityGraph />
                  <div className="chart-footer"><span>Agent throughput</span><span>Task completion <strong>+27%</strong></span></div>
                </section>

                <section className="agent-card card">
                  <div className="card-heading"><div><span className="micro-label">COORDINATION</span><h3>AI Agents</h3></div><button className="card-plus" aria-label="Add agent"><Plus size={15} /></button></div>
                  <div className="agent-list">
                    {agents.map(({ name, state, tone, icon: Icon }) => <div className="agent-row" key={name}><span className={`agent-icon ${tone}`}><Icon size={15} /></span><span className="agent-name">{name}</span><span className={`agent-state ${state.toLowerCase()}`}><i /> {state}</span></div>)}
                  </div>
                  <a className="card-link" href="#platform">Manage agents <ArrowRight size={14} /></a>
                </section>
              </div>

              <section className="activity-card card">
                <div className="activity-heading"><div><span className="micro-label">SYSTEM LOG</span><h3>Recent Activity</h3></div><a href="#developers">View all <ArrowRight size={13} /></a></div>
                <div className="activity-table">
                  <div className="activity-table-head"><span>Time</span><span>Task</span><span>Agent</span><span>Status</span></div>
                  {activity.map(([time, task, agent, status]) => <div className="activity-table-row" key={time}><time>{time}</time><span className="activity-task">{task}</span><span className="table-agent"><span className="tiny-agent">{agent.charAt(0)}</span>{agent}</span><span className={`status ${status.toLowerCase()}`}><CheckCircle2 size={12} /> {status}</span></div>)}
                </div>
              </section>
            </div>
          </div>
        </div>
      </section>

      <section id="platform" className="platform-section container">
        <div className="platform-intro"><span className="section-kicker">ONE CONNECTED SYSTEM</span><h2>Development has<br /><em>more context now.</em></h2></div>
        <p className="platform-copy">Every repository, decision, task, and agent handoff becomes a useful part of the system. cortexMind gives your team an operating layer that compounds with every change.</p>
      </section>

      <section id="solutions" className="capabilities container">
        {capabilities.map(({ icon: Icon, index, title, description }) => <article className="capability" key={index}><div className="capability-top"><span>{index}</span><Icon size={20} /></div><div className="capability-orbit"><i /><i /><b /></div><h3>{title}</h3><p>{description}</p><a href="#workspace">Explore <ArrowRight size={15} /></a></article>)}
      </section>

      <section id="developers" className="developer-band">
        <div className="container developer-grid">
          <div><span className="section-kicker light">WORK WITH THE TOOLS YOU ALREADY USE</span><h2>One brain,<br /><em>many entry points.</em></h2></div>
          <div className="integration-stack">
            <div><Terminal size={18} /><span><b>Connect through MCP</b><small>Persistent project memory in every capable IDE</small></span><CheckCircle2 size={17} /></div>
            <div><Github size={18} /><span><b>Sync the repository</b><small>Stay current with your code and your team</small></span><CheckCircle2 size={17} /></div>
            <div><Code2 size={18} /><span><b>Bring your favorite agents</b><small>Codex, Claude, Cursor, and more</small></span><CheckCircle2 size={17} /></div>
          </div>
        </div>
      </section>

      <section id="pricing" className="closing container">
        <div className="closing-card">
          <div className="closing-orbit" aria-hidden="true"><i /><i /><i /><b /></div>
          <span className="hero-badge hero-badge-dark"><span className="badge-spark"><Sparkles size={12} /></span> Powered by multi-agent intelligence</span>
          <h2>Ideas move faster<br />when agents <em>understand.</em></h2>
          <p>Build the development environment your team and your agents have been waiting for.</p>
          <a className="button button-light" href="mailto:hello@cortex.dev?subject=cortexMind%20early%20access">Start Building <ArrowRight size={17} /></a>
        </div>
      </section>

      <footer className="footer container">
        <a className="brand" href="#top"><span className="brand-mark"><BrainCircuit size={16} /></span><span className="brand-cortex">cortex</span><span className="brand-mind">Mind</span></a>
        <span>AI operating system for modern development.</span>
        <a href="https://github.com/NexVed/Cortex" target="_blank" rel="noreferrer"><Github size={15} /> GitHub</a>
      </footer>
    </main>
  );
}

function ActivityGraph() {
  return (
    <div className="activity-graph" aria-label="Agent throughput graph">
      <div className="graph-axis"><span>140</span><span>100</span><span>60</span><span>20</span></div>
      <svg viewBox="0 0 520 166" preserveAspectRatio="none" role="img" aria-hidden="true">
        <defs>
          <linearGradient id="graph-fill" x1="0" x2="0" y1="0" y2="1"><stop offset="0%" stopColor="#d64b6d" stopOpacity=".32" /><stop offset="100%" stopColor="#d64b6d" stopOpacity="0" /></linearGradient>
          <linearGradient id="graph-stroke" x1="0" x2="1"><stop offset="0%" stopColor="#ee93a9" /><stop offset="55%" stopColor="#d64768" /><stop offset="100%" stopColor="#ed7896" /></linearGradient>
        </defs>
        <path className="graph-grid" d="M0 28H520M0 73H520M0 118H520M0 163H520" />
        <path d="M0 139 C30 132 38 125 61 127 C86 129 95 95 121 105 C145 114 153 78 178 88 C203 98 217 60 241 70 C264 79 274 62 297 67 C324 72 336 29 363 40 C388 50 400 27 423 45 C448 65 461 20 485 36 C502 46 510 25 520 17 L520 166 L0 166Z" fill="url(#graph-fill)" />
        <path className="graph-line" d="M0 139 C30 132 38 125 61 127 C86 129 95 95 121 105 C145 114 153 78 178 88 C203 98 217 60 241 70 C264 79 274 62 297 67 C324 72 336 29 363 40 C388 50 400 27 423 45 C448 65 461 20 485 36 C502 46 510 25 520 17" />
        <circle cx="423" cy="45" r="4" className="graph-dot" /><circle cx="520" cy="17" r="5" className="graph-dot final" />
      </svg>
      <div className="graph-days"><span>MON</span><span>TUE</span><span>WED</span><span>THU</span><span>FRI</span><span>SAT</span><span>SUN</span></div>
    </div>
  );
}