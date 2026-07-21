package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	authorizeURL = "https://github.com/login/oauth/authorize"
	tokenURL     = "https://github.com/login/oauth/access_token"
	apiURL       = "https://api.github.com"
)

type Client struct {
	ClientID, ClientSecret string
	HTTP                   *http.Client
}
type Profile struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}
type Organization struct {
	Login     string `json:"login"`
	AvatarURL string `json:"avatar_url"`
}
type Repository struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	FullName  string `json:"full_name"`
	Private   bool   `json:"private"`
	CloneURL  string `json:"clone_url"`
	HTMLURL   string `json:"html_url"`
	UpdatedAt string `json:"updated_at"`
}

func (c Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}
func (c Client) AuthorizationURL(redirectURI, state, challenge string) string {
	q := url.Values{"client_id": {c.ClientID}, "redirect_uri": {redirectURI}, "state": {state}, "scope": {"read:user user:email repo read:org"}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}}
	return authorizeURL + "?" + q.Encode()
}
func (c Client) Exchange(ctx context.Context, code, redirectURI, verifier string) (string, error) {
	values := url.Values{"client_id": {c.ClientID}, "client_secret": {c.ClientSecret}, "code": {code}, "redirect_uri": {redirectURI}, "code_verifier": {verifier}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(values.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var body struct {
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return "", err
	}
	if res.StatusCode >= 300 || body.Error != "" {
		return "", fmt.Errorf("GitHub token exchange failed: %s %s", body.Error, body.ErrorDescription)
	}
	return body.AccessToken, nil
}
func (c Client) get(ctx context.Context, token, path string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	res, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("GitHub API returned %s", res.Status)
	}
	return json.NewDecoder(res.Body).Decode(target)
}
func (c Client) Profile(ctx context.Context, token string) (Profile, error) {
	var p Profile
	return p, c.get(ctx, token, "/user", &p)
}
func (c Client) Organizations(ctx context.Context, token string) ([]Organization, error) {
	var v []Organization
	return v, c.get(ctx, token, "/user/orgs?per_page=100", &v)
}
func (c Client) Repositories(ctx context.Context, token string) ([]Repository, error) {
	var all []Repository
	for page := 1; ; page++ {
		var current []Repository
		if err := c.get(ctx, token, "/user/repos?affiliation=owner,collaborator,organization_member&per_page=100&page="+strconv.Itoa(page)+"&sort=updated", &current); err != nil {
			return nil, err
		}
		all = append(all, current...)
		if len(current) < 100 {
			return all, nil
		}
	}
}
