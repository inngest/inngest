package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "inngest-cli"
	metadataFile   = "auth.json"
	storageKeyring = "keyring"
	storageFile    = "file"
)

var ErrNotLoggedIn = errors.New("not logged in")

type Metadata struct {
	Issuer               string    `json:"issuer"`
	Resource             string    `json:"resource"`
	ClientID             string    `json:"client_id"`
	SessionID            string    `json:"session_id"`
	SessionExpiresAt     time.Time `json:"session_expires_at"`
	AccountID            string    `json:"account_id"`
	AccountName          string    `json:"account_name"`
	ResourceBoundaryMode string    `json:"resource_boundary_mode"`
	WorkspaceID          *string   `json:"workspace_id,omitempty"`
	WorkspaceName        string    `json:"workspace_name,omitempty"`
	Scopes               []string  `json:"scopes"`
	Storage              string    `json:"storage"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type Credential struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

type keyringStore interface {
	Get(service, user string) (string, error)
	Set(service, user, password string) error
	Delete(service, user string) error
}

type systemKeyring struct{}

func (systemKeyring) Get(service, user string) (string, error) {
	return keyring.Get(service, user)
}

func (systemKeyring) Set(service, user, password string) error {
	return keyring.Set(service, user, password)
}

func (systemKeyring) Delete(service, user string) error {
	return keyring.Delete(service, user)
}

type Store struct {
	dir       string
	keyring   keyringStore
	writeFile func(string, any, fs.FileMode) error
}

func NewStore() (*Store, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	return &Store{dir: dir, keyring: systemKeyring{}, writeFile: writeJSONFile}, nil
}

func newStore(dir string, keyring keyringStore) *Store {
	return &Store{dir: dir, keyring: keyring, writeFile: writeJSONFile}
}

func (s *Store) Save(metadata Metadata, credential Credential, insecure bool) error {
	if metadata.Resource == "" || metadata.SessionID == "" || credential.AccessToken == "" || credential.RefreshToken == "" {
		return errors.New("OAuth credential is incomplete")
	}
	previous, _ := s.Metadata()
	secret, err := json.Marshal(credential)
	if err != nil {
		return fmt.Errorf("encode OAuth credential: %w", err)
	}
	metadata.Storage = storageKeyring
	if insecure {
		metadata.Storage = storageFile
		if err := s.writeFile(s.credentialPath(&metadata), credential, 0o600); err != nil {
			return fmt.Errorf("write OAuth credential: %w", err)
		}
	} else if err := s.keyring.Set(keyringService, keyringUser(&metadata), string(secret)); err != nil {
		return fmt.Errorf("store OAuth credential in the OS credential store: %w", err)
	}
	metadata.UpdatedAt = time.Now().UTC()
	if err := s.writeFile(filepath.Join(s.dir, metadataFile), metadata, 0o600); err != nil {
		if previous == nil || credentialLocation(previous) != credentialLocation(&metadata) {
			_ = s.deleteCredential(&metadata)
		}
		return fmt.Errorf("write OAuth session metadata: %w", err)
	}
	if previous != nil && credentialLocation(previous) != credentialLocation(&metadata) {
		_ = s.deleteCredential(previous)
	}
	return nil
}

func (s *Store) Metadata() (*Metadata, error) {
	metadata := &Metadata{}
	if err := readJSONFile(filepath.Join(s.dir, metadataFile), metadata, false); errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotLoggedIn
	} else if err != nil {
		return nil, fmt.Errorf("read OAuth session metadata: %w", err)
	}
	return metadata, nil
}

func (s *Store) Load() (*Metadata, *Credential, error) {
	metadata, err := s.Metadata()
	if err != nil {
		return nil, nil, err
	}
	credential := &Credential{}
	switch metadata.Storage {
	case storageKeyring:
		secret, err := s.keyring.Get(keyringService, keyringUser(metadata))
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, nil, ErrNotLoggedIn
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read OAuth credential from the OS credential store: %w", err)
		}
		if err := json.Unmarshal([]byte(secret), credential); err != nil {
			return nil, nil, errors.New("stored OAuth credential is invalid")
		}
	case storageFile:
		if err := readJSONFile(s.credentialPath(metadata), credential, true); errors.Is(err, fs.ErrNotExist) {
			return nil, nil, ErrNotLoggedIn
		} else if err != nil {
			return nil, nil, fmt.Errorf("read OAuth credential file: %w", err)
		}
	default:
		return nil, nil, errors.New("stored OAuth credential has an unknown storage type")
	}
	if credential.AccessToken == "" || credential.RefreshToken == "" {
		return nil, nil, errors.New("stored OAuth credential is incomplete")
	}
	return metadata, credential, nil
}

func (s *Store) Delete(metadata *Metadata) error {
	if metadata == nil {
		loaded, err := s.Metadata()
		if errors.Is(err, ErrNotLoggedIn) {
			return nil
		}
		if err != nil {
			return err
		}
		metadata = loaded
	}
	secretErr := s.deleteCredential(metadata)
	metadataErr := os.Remove(filepath.Join(s.dir, metadataFile))
	if errors.Is(metadataErr, fs.ErrNotExist) {
		metadataErr = nil
	}
	return errors.Join(secretErr, metadataErr)
}

func configDir() (string, error) {
	if dir := os.Getenv("INNGEST_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return filepath.Join(home, ".config", "inngest"), nil
}

func keyringUser(metadata *Metadata) string {
	return "oauth:" + credentialID(metadata)
}

func credentialID(metadata *Metadata) string {
	hash := sha256.Sum256([]byte(metadata.Resource + "\x00" + metadata.SessionID))
	return hex.EncodeToString(hash[:8])
}

func credentialLocation(metadata *Metadata) string {
	return metadata.Storage + ":" + credentialID(metadata)
}

func (s *Store) credentialPath(metadata *Metadata) string {
	return filepath.Join(s.dir, "oauth-credentials-"+credentialID(metadata)+".json")
}

func (s *Store) deleteCredential(metadata *Metadata) error {
	switch metadata.Storage {
	case storageKeyring:
		err := s.keyring.Delete(keyringService, keyringUser(metadata))
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return err
	case storageFile:
		err := os.Remove(s.credentialPath(metadata))
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	default:
		return nil
	}
}

func writeJSONFile(path string, value any, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".inngest-auth-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(encoded); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func readJSONFile(path string, value any, requirePrivate bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if requirePrivate && runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return errors.New("credential file permissions must be 0600")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}
