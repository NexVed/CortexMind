package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/rs/zerolog/log"
)

// RegisterOAuthHooks wires PocketBase OAuth2 events so that the GitHub access
// token and profile data are persisted on the user record after login.
func RegisterOAuthHooks(app core.App) {
	app.OnRecordAuthWithOAuth2Request().BindFunc(func(e *core.RecordAuthWithOAuth2RequestEvent) error {
		// Continue the default flow first so the user record exists.
		if err := e.Next(); err != nil {
			return err
		}
		if e.OAuth2User == nil || e.Record == nil {
			return nil
		}

		e.Record.Set("github_access_token", e.OAuth2User.AccessToken)
		if login, ok := e.OAuth2User.RawUser["login"].(string); ok {
			e.Record.Set("github_username", login)
		}
		if e.OAuth2User.AvatarURL != "" {
			e.Record.Set("github_avatar_url", e.OAuth2User.AvatarURL)
		}
		if e.OAuth2User.Name != "" {
			e.Record.Set("display_name", e.OAuth2User.Name)
		}

		if err := e.App.Save(e.Record); err != nil {
			log.Error().Err(err).Msg("failed to persist github token on user record")
			return err
		}
		log.Info().Str("user", e.Record.Id).Msg("stored github oauth token")
		return nil
	})
}

// Client is a thin wrapper around the GitHub REST API v3.
type Client struct {
	AccessToken string
	BaseURL     string
	HTTP        *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		AccessToken: token,
		BaseURL:     "https://api.github.com",
		HTTP:        &http.Client{Timeout: 20 * time.Second},
	}
}

// Repo is a minimal subset of the GitHub repository object.
type Repo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	HTMLURL     string `json:"html_url"`
	CloneURL    string `json:"clone_url"`
	Language    string `json:"language"`
	Default     string `json:"default_branch"`
}

// Org is a minimal subset of the GitHub organization object.
type Org struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
}

func (c *Client) do(method, path string, out any) error {
	req, err := http.NewRequest(method, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("github api %s %s: status %d: %s", method, path, resp.StatusCode, string(body))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

// ListUserRepos returns every repository the authenticated user can access:
// repos they own, repos they collaborate on, and repos in their organizations,
// across both public and private visibility. GitHub paginates at 100 items per
// page, so we walk pages until a short page signals the end. We explicitly pass
// affiliation + visibility because the API's defaults can silently omit org and
// collaborator repositories for some token types.
func (c *Client) ListUserRepos() ([]Repo, error) {
	var all []Repo
	const perPage = 100
	for page := 1; page <= 50; page++ { // cap at 5000 repos
		var batch []Repo
		path := fmt.Sprintf(
			"/user/repos?per_page=%d&page=%d&sort=updated&visibility=all&affiliation=owner,collaborator,organization_member",
			perPage, page)
		if err := c.do(http.MethodGet, path, &batch); err != nil {
			return all, err
		}
		all = append(all, batch...)
		if len(batch) < perPage {
			break
		}
	}
	return all, nil
}

func (c *Client) GetRepo(owner, name string) (*Repo, error) {
	var repo Repo
	err := c.do(http.MethodGet, fmt.Sprintf("/repos/%s/%s", owner, name), &repo)
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (c *Client) GetUserOrgs() ([]Org, error) {
	var orgs []Org
	err := c.do(http.MethodGet, "/user/orgs", &orgs)
	return orgs, err
}
