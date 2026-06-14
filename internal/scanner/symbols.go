package scanner

import (
	"regexp"
	"strings"
)

// FunctionSymbol describes a detected function/method.
type FunctionSymbol struct {
	Name      string `json:"name"`
	Line      int    `json:"line"`
	Signature string `json:"signature"`
	IsPublic  bool   `json:"is_public"`
}

// ClassSymbol describes a detected class/struct/interface/type.
type ClassSymbol struct {
	Name string `json:"name"`
	Line int    `json:"line"`
}

// Symbols is the aggregate result of parsing a single file.
type Symbols struct {
	Functions []FunctionSymbol `json:"functions"`
	Classes   []ClassSymbol    `json:"classes"`
	Imports   []string         `json:"imports"`
}

// Lightweight, dependency-free symbol extraction via regular expressions.
// This is intentionally a pragmatic substitute for Tree-Sitter (Phase 3 of the
// backend guide) — it covers the common declaration forms for the primary
// languages without requiring CGO or native parser bindings.
var (
	reGoFunc    = regexp.MustCompile(`^\s*func\s+(?:\([^)]*\)\s*)?([A-Za-z_]\w*)\s*\(`)
	reGoType    = regexp.MustCompile(`^\s*type\s+([A-Za-z_]\w*)\s+(?:struct|interface)\b`)
	reGoImport  = regexp.MustCompile(`^\s*"([^"]+)"`)
	reTsFunc    = regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][\w$]*)\s*\(`)
	reTsArrow   = regexp.MustCompile(`^\s*(?:export\s+)?const\s+([A-Za-z_$][\w$]*)\s*=\s*(?:async\s+)?\(`)
	reTsClass   = regexp.MustCompile(`^\s*(?:export\s+)?(?:abstract\s+)?(?:class|interface|type|enum)\s+([A-Za-z_$][\w$]*)`)
	reTsImport  = regexp.MustCompile(`^\s*import\s+.*?from\s+['"]([^'"]+)['"]`)
	rePyFunc    = regexp.MustCompile(`^\s*def\s+([A-Za-z_]\w*)\s*\(`)
	rePyClass   = regexp.MustCompile(`^\s*class\s+([A-Za-z_]\w*)`)
	rePyImport  = regexp.MustCompile(`^\s*(?:from\s+([\w.]+)\s+import|import\s+([\w.]+))`)
	reRustFunc  = regexp.MustCompile(`^\s*(?:pub\s+)?(?:async\s+)?fn\s+([A-Za-z_]\w*)`)
	reRustType  = regexp.MustCompile(`^\s*(?:pub\s+)?(?:struct|enum|trait)\s+([A-Za-z_]\w*)`)
	reRustUse   = regexp.MustCompile(`^\s*use\s+([\w:]+)`)
)

// ExtractSymbols parses the given source content for the given language.
func ExtractSymbols(content, lang string) *Symbols {
	s := &Symbols{Functions: []FunctionSymbol{}, Classes: []ClassSymbol{}, Imports: []string{}}
	lines := strings.Split(content, "\n")

	addFunc := func(name string, ln int) {
		s.Functions = append(s.Functions, FunctionSymbol{
			Name:      name,
			Line:      ln,
			Signature: strings.TrimSpace(lines[ln-1]),
			IsPublic:  isPublic(name, lang),
		})
	}
	addClass := func(name string, ln int) {
		s.Classes = append(s.Classes, ClassSymbol{Name: name, Line: ln})
	}

	for i, line := range lines {
		ln := i + 1
		switch lang {
		case "Go":
			if m := reGoFunc.FindStringSubmatch(line); m != nil {
				addFunc(m[1], ln)
			} else if m := reGoType.FindStringSubmatch(line); m != nil {
				addClass(m[1], ln)
			} else if m := reGoImport.FindStringSubmatch(line); m != nil {
				s.Imports = append(s.Imports, m[1])
			}
		case "TypeScript", "JavaScript":
			if m := reTsFunc.FindStringSubmatch(line); m != nil {
				addFunc(m[1], ln)
			} else if m := reTsArrow.FindStringSubmatch(line); m != nil {
				addFunc(m[1], ln)
			} else if m := reTsClass.FindStringSubmatch(line); m != nil {
				addClass(m[1], ln)
			} else if m := reTsImport.FindStringSubmatch(line); m != nil {
				s.Imports = append(s.Imports, m[1])
			}
		case "Python":
			if m := rePyFunc.FindStringSubmatch(line); m != nil {
				addFunc(m[1], ln)
			} else if m := rePyClass.FindStringSubmatch(line); m != nil {
				addClass(m[1], ln)
			} else if m := rePyImport.FindStringSubmatch(line); m != nil {
				if m[1] != "" {
					s.Imports = append(s.Imports, m[1])
				} else if m[2] != "" {
					s.Imports = append(s.Imports, m[2])
				}
			}
		case "Rust":
			if m := reRustFunc.FindStringSubmatch(line); m != nil {
				addFunc(m[1], ln)
			} else if m := reRustType.FindStringSubmatch(line); m != nil {
				addClass(m[1], ln)
			} else if m := reRustUse.FindStringSubmatch(line); m != nil {
				s.Imports = append(s.Imports, m[1])
			}
		}
	}
	return s
}

func isPublic(name, lang string) bool {
	if name == "" {
		return false
	}
	switch lang {
	case "Go":
		r := rune(name[0])
		return r >= 'A' && r <= 'Z'
	case "Python":
		return !strings.HasPrefix(name, "_")
	default:
		return true
	}
}
