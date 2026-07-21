package repositories

import (
	"fmt"

	"github.com/NexVed/Cortex/internal/database"
	gh "github.com/NexVed/Cortex/internal/github"
)

type UserRepository struct{ DB *database.DB }

func (r UserRepository) Current() (*database.User, error) { return r.DB.CurrentUser() }
func (r UserRepository) SaveGitHub(p gh.Profile) (*database.User, error) {
	u := &database.User{ID: "github:" + p.Login, Provider: "github", GitHubID: fmtID(p.ID), Username: p.Login, DisplayName: p.Name, AvatarURL: p.AvatarURL}
	if u.DisplayName == "" {
		u.DisplayName = p.Login
	}
	return u, r.DB.SaveUser(*u)
}
func (r UserRepository) SaveOffline(displayName string) (*database.User, error) {
	if displayName == "" {
		return nil, fmt.Errorf("workspace name is required")
	}
	u := &database.User{ID: "offline:" + displayName, Provider: "offline", Username: displayName, DisplayName: displayName, Offline: true}
	return u, r.DB.SaveUser(*u)
}
func fmtID(id int64) string { return fmt.Sprintf("%d", id) }
