package services

import (
	"context"

	"github.com/NexVed/Cortex/internal/database"
	gh "github.com/NexVed/Cortex/internal/github"
	"github.com/NexVed/Cortex/internal/repositories"
)

// Onboarding coordinates trusted backend services; it never exposes access tokens.
type Onboarding struct {
	Users  repositories.UserRepository
	GitHub gh.Client
	DB     *database.DB
}

func (s Onboarding) CompleteGitHub(ctx context.Context, token string) (*database.User, error) {
	profile, err := s.GitHub.Profile(ctx, token)
	if err != nil {
		return nil, err
	}
	orgs, err := s.GitHub.Organizations(ctx, token)
	if err != nil {
		return nil, err
	}
	repos, err := s.GitHub.Repositories(ctx, token)
	if err != nil {
		return nil, err
	}
	user, err := s.Users.SaveGitHub(profile)
	if err != nil {
		return nil, err
	}
	localOrgs := make([]database.Organization, len(orgs))
	for i, o := range orgs {
		localOrgs[i] = database.Organization{Login: o.Login, AvatarURL: o.AvatarURL}
	}
	localRepos := make([]database.Repository, len(repos))
	for i, r := range repos {
		localRepos[i] = database.Repository{GitHubID: r.ID, Name: r.Name, FullName: r.FullName, Private: r.Private, CloneURL: r.CloneURL, HTMLURL: r.HTMLURL, UpdatedAt: r.UpdatedAt}
	}
	if err = s.DB.ReplaceGitHubData(localOrgs, localRepos); err != nil {
		return nil, err
	}
	if err = s.DB.SetActiveUser(user.ID); err != nil {
		return nil, err
	}
	return user, nil
}

func (s Onboarding) ContinueOffline(displayName string) (*database.User, error) {
	user, err := s.Users.SaveOffline(displayName)
	if err != nil {
		return nil, err
	}
	if err = s.DB.SetActiveUser(user.ID); err != nil {
		return nil, err
	}
	return user, nil
}
