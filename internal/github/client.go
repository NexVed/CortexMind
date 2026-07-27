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
	tokenURL      = "https://github.com/login/oauth/access_token"
	deviceCodeURL = "https://github.com/login/device/code"
	apiURL        = "https://api.github.com"
	defaultScope  = "read:user user:email repo read:org"
)

type Client struct {
	ClientID string
	HTTP     *http.Client
}

type DeviceCode struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type DeviceFlowError struct {
	Code        string `json:"error"`
	Description string `json:"error_description"`
}

func (e *DeviceFlowError) Error() string {
	if e.Description != "" {
		return fmt.Sprintf("GitHub device authorization failed: %s", e.Description)
	}
	return fmt.Sprintf("GitHub device authorization failed: %s", e.Code)
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

// StartDeviceFlow creates a one-time code that a user enters at GitHub. It uses
// only the public client ID; native desktop users never receive a client secret.
func (c Client) StartDeviceFlow(ctx context.Context) (DeviceCode, error) {
	var device DeviceCode
	values := url.Values{"client_id": {c.ClientID}, "scope": {defaultScope}}
	if err := c.postForm(ctx, deviceCodeURL, values, &device); err != nil {
		return DeviceCode{}, err
	}
	if device.DeviceCode == "" || device.UserCode == "" || device.VerificationURI == "" {
		return DeviceCode{}, fmt.Errorf("GitHub device authorization returned an incomplete response")
	}
	if device.Interval < 1 {
		device.Interval = 5
	}
	if device.ExpiresIn < 1 {
		device.ExpiresIn = 900
	}
	return device, nil
}

// ExchangeDeviceCode returns a token after the user approves the device code.
// authorization_pending and slow_down are returned as DeviceFlowError values so
// the caller can keep polling at GitHub's requested interval.
func (c Client) ExchangeDeviceCode(ctx context.Context, deviceCode string) (string, error) {
	values := url.Values{
		"client_id":   {c.ClientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}
	var body struct {
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := c.postForm(ctx, tokenURL, values, &body); err != nil {
		return "", err
	}
	if body.Error != "" {
		return "", &DeviceFlowError{Code: body.Error, Description: body.ErrorDescription}
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("GitHub device authorization returned no access token")
	}
	return body.AccessToken, nil
}

func (c Client) postForm(ctx context.Context, endpoint string, values url.Values, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(target); err != nil {
		return err
	}
	if res.StatusCode >= 300 {
		return fmt.Errorf("GitHub returned %s", res.Status)
	}
	return nil
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
