package keychain

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const service = "com.nexved.cortexmind"

// TokenStore keeps provider credentials in the operating system credential store,
// never in the application database.
type TokenStore interface {
	Get(account string) (string, error)
	Set(account, token string) error
	Delete(account string) error
}

type Store struct{}

func (Store) Get(account string) (string, error) { return keyring.Get(service, account) }
func (Store) Set(account, token string) error {
	if token == "" {
		return errors.New("refusing to store an empty access token")
	}
	return keyring.Set(service, account, token)
}
func (Store) Delete(account string) error { return keyring.Delete(service, account) }
