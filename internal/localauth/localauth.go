// Package localauth protects the desktop API and verifies the local daemon.
package localauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/NexVed/Cortex/internal/keychain"
)

func Account(dataDir string) string {
	absolute, _ := filepath.Abs(dataDir)
	if runtime.GOOS == "windows" {
		absolute = strings.ToLower(absolute)
	}
	digest := sha256.Sum256([]byte(filepath.Clean(absolute)))
	return "local-api:" + hex.EncodeToString(digest[:])
}

// Load keeps a credential separate from the database and never serves it over HTTP.
func Load(dataDir string, store keychain.TokenStore) (string, error) {
	account := Account(dataDir)
	token, err := store.Get(account)
	if err == nil && len(token) >= 32 {
		return token, nil
	}
	if err != nil && !errors.Is(err, keychain.ErrNotFound) {
		return "", fmt.Errorf("read local API credential: %w", err)
	}
	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return "", err
	}
	token = base64.RawURLEncoding.EncodeToString(bytes)
	if err = store.Set(account, token); err != nil {
		return "", fmt.Errorf("store local API credential: %w", err)
	}
	return token, nil
}

func Proof(token, challenge string) string {
	hash := hmac.New(sha256.New, []byte(token))
	_, _ = hash.Write([]byte("cortexMind-health:" + challenge))
	return hex.EncodeToString(hash.Sum(nil))
}

// Alive does not send the credential to a process that might have occupied our port.
func Alive(addr, token string) bool {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return false
	}
	challenge := hex.EncodeToString(bytes)
	client := &http.Client{Timeout: time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Get("http://" + addr + "/api/health?challenge=" + challenge)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	var body struct {
		App   string `json:"app"`
		Ready bool   `json:"ready"`
		Proof string `json:"proof"`
	}
	if res.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&body) != nil {
		return false
	}
	return body.App == "cortexMind" && body.Ready && hmac.Equal([]byte(body.Proof), []byte(Proof(token, challenge)))
}

func ValidDevOrigin(origin string) bool {
	u, err := url.Parse(origin)
	return err == nil && u.Scheme == "http" && u.User == nil && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1") && u.Port() != "" && u.RawQuery == "" && u.Fragment == "" && (u.Path == "" || u.Path == "/")
}

func LaunchURL(addr, token, devOrigin string) string {
	base := "http://" + addr
	if ValidDevOrigin(devOrigin) {
		base = strings.TrimSuffix(devOrigin, "/")
	}
	return base + "/#local_auth=" + url.QueryEscape(token)
}
