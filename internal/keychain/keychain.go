package keychain

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const service = "com.nexved.cortexmind"

var ErrNotFound = keyring.ErrNotFound

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
func (Store) Delete(account string) error {
	err := keyring.Delete(service, account)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
