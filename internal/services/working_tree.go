package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	cgit "github.com/NexVed/Cortex/internal/git"
)

type WorkingTreeScan struct {
	Scan    *ScanResult              `json:"scan"`
	Changes *cgit.WorkingTreeChanges `json:"changes"`
}

func (s ScanService) ValidateWorkingTree(ctx context.Context, projectID, path string) (string, error) {
	project, err := s.DB.Project(projectID)
	if err != nil {
		return "", fmt.Errorf("project not found")
	}
	if path == "" {
		path, _ = s.DB.WorkingTreePath(projectID)
		if path == "" {
			path = project.Path
		}
	}
	if path == "" {
		return "", fmt.Errorf("provide repo_path pointing to your local clone, for example C:\\Users\\you\\source\\repo")
	}
	root, err := cgit.CheckoutRoot(ctx, path)
	if err != nil {
		return "", err
	}
	if project.GitHubURL != "" {
		origin, _, err := cgit.GitOutput(ctx, root, 8192, "remote", "get-url", "origin")
		if err != nil {
			return "", fmt.Errorf("checkout needs an origin remote matching the selected project")
		}
		expected := cgit.RepositoryIdentity(project.GitHubURL)
		if expected == "" || expected != cgit.RepositoryIdentity(origin) {
			return "", fmt.Errorf("checkout origin does not match the selected project")
		}
	} else if project.Path != "" && !cgit.SamePath(project.Path, root) {
		return "", fmt.Errorf("checkout must match the local project's directory")
	}
	return root, nil
}

// ResolveWorkingTree selects a project from the registered local path or origin URL.
// Multiple matches require an explicit project_id, avoiding mixed project memories.
func (s ScanService) ResolveWorkingTree(ctx context.Context, path string) (string, error) {
	root, err := cgit.CheckoutRoot(ctx, path)
	if err != nil {
		return "", err
	}
	origin, _, _ := cgit.GitOutput(ctx, root, 8192, "remote", "get-url", "origin")
	identity := cgit.RepositoryIdentity(origin)
	projects, err := s.DB.ListProjects()
	if err != nil {
		return "", err
	}
	matches := []string{}
	for _, p := range projects {
		bound, _ := s.DB.WorkingTreePath(p.ID)
		if cgit.SamePath(root, p.Path) || cgit.SamePath(root, bound) || (identity != "" && identity == cgit.RepositoryIdentity(p.GitHubURL)) {
			matches = append(matches, p.ID)
		}
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("checkout matches %d projects; call cortex_list_projects and provide project_id (matches: %s)", len(matches), strings.Join(matches, ", "))
	}
	return matches[0], nil
}

func (s ScanService) WorkingTreeChanges(ctx context.Context, projectID, path string, includeDiff bool) (*cgit.WorkingTreeChanges, error) {
	root, err := s.ValidateWorkingTree(ctx, projectID, path)
	if err != nil {
		return nil, err
	}
	return cgit.ReviewWorkingTree(ctx, root, includeDiff)
}

func (s ScanService) ScanWorkingTree(ctx context.Context, projectID, path string, includeDiff bool) (*WorkingTreeScan, error) {
	if s.ScanMutex != nil {
		s.ScanMutex.Lock()
		defer s.ScanMutex.Unlock()
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	root, err := s.ValidateWorkingTree(ctx, projectID, path)
	if err != nil {
		return nil, err
	}
	changes, err := cgit.ReviewWorkingTree(ctx, root, includeDiff)
	if err != nil {
		return nil, err
	}
	scan, err := s.indexDirectory(ctx, projectID, root, true)
	if err != nil {
		return nil, err
	}
	if err = s.DB.BindWorkingTree(projectID, root); err != nil {
		return nil, err
	}
	return &WorkingTreeScan{Scan: scan, Changes: changes}, nil
}
