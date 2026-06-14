package analyzer

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// stringSet is an insertion-deduplicated string collection.
type stringSet struct {
	m map[string]bool
}

func newSet() *stringSet { return &stringSet{m: map[string]bool{}} }

func (s *stringSet) add(v ...string) {
	for _, x := range v {
		x = strings.TrimSpace(x)
		if x != "" {
			s.m[x] = true
		}
	}
}

func (s *stringSet) has(v string) bool { return s.m[v] }

func (s *stringSet) list() []string {
	out := make([]string, 0, len(s.m))
	for k := range s.m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// repoSignals accumulates everything learned from a repo walk.
type repoSignals struct {
	frameworks      *stringSet
	databases       *stringSet
	tools           *stringSet
	packageManagers *stringSet
	authMechanisms  *stringSet
	authProviders   *stringSet
	authLibraries   *stringSet
	endpoints       *stringSet
	featureSignals  *stringSet
}

func newSignals() *repoSignals {
	return &repoSignals{
		frameworks:      newSet(),
		databases:       newSet(),
		tools:           newSet(),
		packageManagers: newSet(),
		authMechanisms:  newSet(),
		authProviders:   newSet(),
		authLibraries:   newSet(),
		endpoints:       newSet(),
		featureSignals:  newSet(),
	}
}

func (r *repoSignals) auth() AuthAnalysis {
	a := AuthAnalysis{
		Mechanisms: r.authMechanisms.list(),
		Providers:  r.authProviders.list(),
		Libraries:  r.authLibraries.list(),
	}
	a.Detected = len(a.Mechanisms) > 0 || len(a.Providers) > 0 || len(a.Libraries) > 0
	return a
}

// dependencyMap maps a (lowercased) dependency/keyword to a (set, value) pair.
type depHit struct {
	target string // "framework" | "database" | "tool" | "auth_lib" | "auth_provider" | "auth_mech"
	value  string
}

// dependencyTable recognises common ecosystem packages.
var dependencyTable = map[string]depHit{
	// JS frameworks
	"react": {"framework", "React"}, "react-dom": {"framework", "React"},
	"next": {"framework", "Next.js"}, "vue": {"framework", "Vue"},
	"@angular/core": {"framework", "Angular"}, "svelte": {"framework", "Svelte"},
	"solid-js": {"framework", "SolidJS"}, "express": {"framework", "Express"},
	"fastify": {"framework", "Fastify"}, "@nestjs/core": {"framework", "NestJS"},
	"koa": {"framework", "Koa"}, "vite": {"tool", "Vite"}, "webpack": {"tool", "Webpack"},
	"tailwindcss": {"framework", "Tailwind CSS"},
	// Python
	"django": {"framework", "Django"}, "flask": {"framework", "Flask"},
	"fastapi": {"framework", "FastAPI"}, "sqlalchemy": {"database", "SQLAlchemy"},
	"pydantic": {"tool", "Pydantic"},
	// Go
	"github.com/gin-gonic/gin": {"framework", "Gin"},
	"github.com/labstack/echo": {"framework", "Echo"},
	"github.com/gofiber/fiber": {"framework", "Fiber"},
	"github.com/pocketbase/pocketbase": {"framework", "PocketBase"},
	"connectrpc.com/connect":            {"framework", "ConnectRPC"},
	"gorm.io/gorm":                      {"database", "GORM"},
	// Rust
	"actix-web": {"framework", "Actix"}, "axum": {"framework", "Axum"}, "rocket": {"framework", "Rocket"},
	// Databases
	"pg": {"database", "PostgreSQL"}, "postgres": {"database", "PostgreSQL"},
	"mysql": {"database", "MySQL"}, "mysql2": {"database", "MySQL"},
	"mongodb": {"database", "MongoDB"}, "mongoose": {"database", "MongoDB"},
	"redis": {"database", "Redis"}, "sqlite3": {"database", "SQLite"},
	"better-sqlite3": {"database", "SQLite"}, "prisma": {"database", "Prisma"},
	"@prisma/client": {"database", "Prisma"}, "drizzle-orm": {"database", "Drizzle"},
	"psycopg2": {"database", "PostgreSQL"}, "pymongo": {"database", "MongoDB"},
	"lancedb": {"database", "LanceDB"}, "ollama": {"tool", "Ollama"},
	// Tools
	"docker": {"tool", "Docker"}, "kubernetes": {"tool", "Kubernetes"},
	"graphql": {"tool", "GraphQL"}, "jest": {"tool", "Jest"}, "vitest": {"tool", "Vitest"},
	"typescript": {"tool", "TypeScript"}, "eslint": {"tool", "ESLint"},
	// Auth libraries
	"passport": {"auth_lib", "Passport"}, "next-auth": {"auth_lib", "NextAuth"},
	"jsonwebtoken": {"auth_lib", "jsonwebtoken"}, "bcrypt": {"auth_lib", "bcrypt"},
	"bcryptjs": {"auth_lib", "bcrypt"}, "argon2": {"auth_lib", "argon2"},
	"@clerk/clerk-sdk-node": {"auth_provider", "Clerk"}, "@clerk/nextjs": {"auth_provider", "Clerk"},
	"firebase": {"auth_provider", "Firebase"}, "firebase-admin": {"auth_provider", "Firebase"},
	"@supabase/supabase-js": {"auth_provider", "Supabase"},
	"@auth0/auth0-react": {"auth_provider", "Auth0"}, "auth0": {"auth_provider", "Auth0"},
	"github.com/golang-jwt/jwt": {"auth_lib", "golang-jwt"},
	"golang.org/x/oauth2":       {"auth_mech", "oauth2"},
	"passlib":                   {"auth_lib", "passlib"}, "python-jose": {"auth_lib", "python-jose"},
	"authlib": {"auth_lib", "Authlib"},
}

func (r *repoSignals) recordDep(name string) {
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return
	}
	if hit, ok := dependencyTable[key]; ok {
		r.applyHit(hit)
		return
	}
	// Prefix match for go modules / scoped paths.
	for k, hit := range dependencyTable {
		if strings.HasPrefix(key, k) {
			r.applyHit(hit)
			return
		}
	}
}

func (r *repoSignals) applyHit(hit depHit) {
	switch hit.target {
	case "framework":
		r.frameworks.add(hit.value)
	case "database":
		r.databases.add(hit.value)
	case "tool":
		r.tools.add(hit.value)
	case "auth_lib":
		r.authLibraries.add(hit.value)
		r.authMechanisms.add(authMechForLib(hit.value))
	case "auth_provider":
		r.authProviders.add(hit.value)
		r.authMechanisms.add("oauth2")
	case "auth_mech":
		r.authMechanisms.add(hit.value)
	}
}

func authMechForLib(lib string) string {
	switch lib {
	case "jsonwebtoken", "golang-jwt", "python-jose":
		return "jwt"
	case "bcrypt", "argon2", "passlib", "Passport":
		return "password"
	default:
		return ""
	}
}

// ── Repo walk ──────────────────────────────────────────

var (
	reRoute     = regexp.MustCompile(`(?i)(?:app|router|r|mux|api)\.(get|post|put|patch|delete)\(\s*["` + "`" + `']([^"` + "`" + `']+)`)
	reGoRoute   = regexp.MustCompile(`(?i)\.(?:Handle|HandleFunc|GET|POST|PUT|PATCH|DELETE)\(\s*"([^"]+)"`)
	authKeyword = regexp.MustCompile(`(?i)\b(jwt|oauth2?|openid|saml|session|cookie|bearer\s+token|api[_-]?key|login|signin|sign-in|authenticate|authorization|middleware\s+auth)\b`)
)

// scanRepo reads dependency manifests and performs a bounded content scan for
// auth/route/feature signals.
func scanRepo(repoPath string) *repoSignals {
	s := newSignals()
	parseManifests(repoPath, s)

	const maxFiles = 2500
	const maxBytes = 256 * 1024
	scanned := 0

	_ = filepath.Walk(repoPath, func(path string, fi os.FileInfo, err error) error {
		if err != nil || scanned >= maxFiles {
			if scanned >= maxFiles {
				return filepath.SkipDir
			}
			return nil
		}
		if fi.IsDir() {
			n := fi.Name()
			if n == "node_modules" || n == ".git" || n == "vendor" || n == "dist" || n == "build" || n == "target" || n == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}
		if !isSourceFile(path) || fi.Size() > maxBytes {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		scanned++
		content := string(data)
		scanContent(content, s)
		return nil
	})
	return s
}

func scanContent(content string, s *repoSignals) {
	for _, m := range reRoute.FindAllStringSubmatch(content, -1) {
		s.endpoints.add(strings.ToUpper(m[1]) + " " + m[2])
	}
	for _, m := range reGoRoute.FindAllStringSubmatch(content, -1) {
		if strings.HasPrefix(m[1], "/") {
			s.endpoints.add(m[1])
		}
	}
	for _, m := range authKeyword.FindAllStringSubmatch(content, -1) {
		kw := strings.ToLower(strings.TrimSpace(m[1]))
		switch {
		case strings.HasPrefix(kw, "jwt") || kw == "bearer token":
			s.authMechanisms.add("jwt")
		case strings.HasPrefix(kw, "oauth") || kw == "openid":
			s.authMechanisms.add("oauth2")
		case kw == "saml":
			s.authMechanisms.add("saml")
		case kw == "session" || kw == "cookie":
			s.authMechanisms.add("session")
		case strings.Contains(kw, "api") && strings.Contains(kw, "key"):
			s.authMechanisms.add("api-key")
		case kw == "login" || kw == "signin" || kw == "sign-in" || kw == "authenticate" || strings.HasPrefix(kw, "middleware"):
			s.featureSignals.add("Authentication")
		}
	}
	low := strings.ToLower(content)
	if strings.Contains(low, "github.com/login/oauth") || strings.Contains(low, "provider: 'github'") || strings.Contains(low, "provider:\"github\"") {
		s.authProviders.add("GitHub")
		s.authMechanisms.add("oauth2")
	}
	if strings.Contains(low, "accounts.google.com") || strings.Contains(low, "provider: 'google'") {
		s.authProviders.add("Google")
		s.authMechanisms.add("oauth2")
	}
}

func isSourceFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".rs", ".java", ".rb", ".php", ".cs", ".kt", ".swift", ".vue", ".svelte":
		return true
	}
	return false
}
