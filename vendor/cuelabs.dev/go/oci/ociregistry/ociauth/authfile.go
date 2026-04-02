package ociauth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// AuthConfig represents access to system level (e.g. config-file or command-execution based)
// configuration information.
//
// It's OK to call EntryForRegistry concurrently.
type Config interface {
	// EntryForRegistry returns auth information for the given host.
	// If there's no information available, it should return the zero ConfigEntry
	// and nil.
	EntryForRegistry(host string) (ConfigEntry, error)
}

// ConfigEntry holds auth information for a registry.
// It mirrors the information obtainable from the .docker/config.json
// file and from the docker credential helper protocol
type ConfigEntry struct {
	// RefreshToken holds a token that can be used to obtain an access token.
	RefreshToken string
	// AccessToken holds a bearer token to be sent to a registry.
	AccessToken string
	// Username holds the username for use with basic auth.
	Username string
	// Password holds the password for use with Username.
	Password string
}

// ConfigFile holds auth information for OCI registries as read from a configuration file.
// It implements [Config].
type ConfigFile struct {
	data   configData
	runner HelperRunner
}

var ErrHelperNotFound = errors.New("helper not found")

// HelperRunner is the function used to execute auth "helper"
// commands. It's passed the helper name as specified in the configuration file,
// without the "docker-credential-helper-" prefix.
//
// If the credentials are not found, it should return the zero AuthInfo
// and no error.
//
// If the helper doesn't exist, it should return an [ErrHelperNotFound] error.
type HelperRunner = func(helperName string, serverURL string) (ConfigEntry, error)

// configData holds the part of ~/.docker/config.json that pertains to auth.
type configData struct {
	Auths       map[string]authConfig `json:"auths"`
	CredsStore  string                `json:"credsStore,omitempty"`
	CredHelpers map[string]string     `json:"credHelpers,omitempty"`
}

// authConfig contains authorization information for connecting to a Registry.
type authConfig struct {
	// derivedFrom records the entries from which this one was derived.
	// If this is empty, the entry was explicitly present.
	derivedFrom []string

	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	// Auth is an alternative way of specifying username and password
	// (in base64(username:password) form.
	Auth string `json:"auth,omitempty"`

	// IdentityToken is used to authenticate the user and get
	// an access token for the registry.
	IdentityToken string `json:"identitytoken,omitempty"`

	// RegistryToken is a bearer token to be sent to a registry
	RegistryToken string `json:"registrytoken,omitempty"`
}

// LoadWithEnv is like [Load] but takes environment variables in the form
// returned by [os.Environ] instead of calling [os.Getenv]. If env
// is nil, the current process's environment will be used.
func LoadWithEnv(runner HelperRunner, env []string) (*ConfigFile, error) {
	if runner == nil {
		runner = ExecHelperWithEnv(env)
	}
	getenv := os.Getenv
	if env != nil {
		getenv = getenvFunc(env)
	}
	// DOCKER_AUTH_CONFIG has precedence, therefore check if
	// it has the inlined JSON.
	if data := getenv("DOCKER_AUTH_CONFIG"); data != "" {
		f, err := decodeConfigFile([]byte(data))
		if err != nil {
			return nil, fmt.Errorf("invalid config: %v", err)
		}
		return &ConfigFile{
			data:   f,
			runner: runner,
		}, nil
	}
	for _, f := range configFileLocations {
		filename := f(getenv)
		if filename == "" {
			continue
		}
		data, err := os.ReadFile(filename)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		f, err := decodeConfigFile(data)
		if err != nil {
			return nil, fmt.Errorf("invalid config file %q: %v", filename, err)
		}
		return &ConfigFile{
			data:   f,
			runner: runner,
		}, nil
	}
	return &ConfigFile{
		runner: runner,
	}, nil
}

// Load loads the auth configuration from the first location it can find.
// It uses runner to run any external helper commands; if runner
// is nil, [ExecHelper] will be used.
//
// In order, it tries:
// - $DOCKER_AUTH_CONFIG (inlined JSON)
// - $DOCKER_CONFIG/config.json
// - ~/.docker/config.json
// - $XDG_RUNTIME_DIR/containers/auth.json
func Load(runner HelperRunner) (*ConfigFile, error) {
	return LoadWithEnv(runner, nil)
}

func getenvFunc(env []string) func(string) string {
	return func(key string) string {
		for i := len(env) - 1; i >= 0; i-- {
			if e := env[i]; len(e) >= len(key)+1 && e[len(key)] == '=' && e[:len(key)] == key {
				return e[len(key)+1:]
			}
		}
		return ""
	}
}

var configFileLocations = []func(func(string) string) string{
	func(getenv func(string) string) string {
		if d := getenv("DOCKER_CONFIG"); d != "" {
			return filepath.Join(d, "config.json")
		}
		return ""
	},
	func(getenv func(string) string) string {
		if home := userHomeDir(getenv); home != "" {
			return filepath.Join(home, ".docker", "config.json")
		}
		return ""
	},
	// If neither of the above locations was found, look for Podman's auth at
	// $XDG_RUNTIME_DIR/containers/auth.json and attempt to load it as a
	// Docker config.
	func(getenv func(string) string) string {
		if d := getenv("XDG_RUNTIME_DIR"); d != "" {
			return filepath.Join(d, "containers", "auth.json")
		}
		return ""
	},
}

// userHomeDir returns the current user's home directory.
// The logic in this is directly derived from the logic in
// [os.UserHomeDir] as of go 1.22.0.
//
// It's defined as a variable so it can be patched in tests.
var userHomeDir = func(getenv func(string) string) string {
	env := "HOME"
	switch runtime.GOOS {
	case "windows":
		env = "USERPROFILE"
	case "plan9":
		env = "home"
	}
	if v := getenv(env); v != "" {
		return v
	}
	// On some geese the home directory is not always defined.
	switch runtime.GOOS {
	case "android":
		return "/sdcard"
	case "ios":
		return "/"
	}
	return ""
}

// EntryForRegistry implements [Authorizer.InfoForRegistry].
// If no registry is found, it returns the zero [ConfigEntry] and a nil error.
func (c *ConfigFile) EntryForRegistry(registryHostname string) (ConfigEntry, error) {
	helper, ok := c.data.CredHelpers[registryHostname]
	explicit := true
	if !ok {
		helper = c.data.CredsStore
		explicit = false
	}
	if helper != "" {
		entry, err := c.runner(helper, registryHostname)
		if err == nil || explicit || !errors.Is(err, ErrHelperNotFound) {
			return entry, err
		}
		// The helper command isn't found and it's a fallback default.
		// Don't treat that as an error, because it's common for
		// a helper default to be set up without the helper actually
		// existing. See https://github.com/cue-lang/cue/issues/2934.
	}
	auth := c.data.Auths[registryHostname]
	if auth.IdentityToken != "" && auth.Username != "" {
		return ConfigEntry{}, fmt.Errorf("ambiguous auth credentials")
	}
	if len(auth.derivedFrom) > 1 {
		return ConfigEntry{}, fmt.Errorf("more than one auths entry for %q (%s)", registryHostname, strings.Join(auth.derivedFrom, ", "))
	}

	return ConfigEntry{
		RefreshToken: auth.IdentityToken,
		AccessToken:  auth.RegistryToken,
		Username:     auth.Username,
		Password:     auth.Password,
	}, nil
}

func decodeConfigFile(data []byte) (configData, error) {
	var f configData
	if err := json.Unmarshal(data, &f); err != nil {
		return configData{}, fmt.Errorf("decode failed: %v", err)
	}
	for addr, ac := range f.Auths {
		if ac.Auth != "" {
			var err error
			ac.Username, ac.Password, err = decodeAuth(ac.Auth)
			if err != nil {
				return configData{}, fmt.Errorf("cannot decode auth field for %q: %v", addr, err)
			}
		}
		f.Auths[addr] = ac
		if !strings.Contains(addr, "//") {
			continue
		}
		// It looks like it might be a URL, so follow the original logic
		// and extract the host name for later lookup. Explicit
		// entries override implicit, and if several entries map to
		// the same host, we record that so we can return an error
		// later if that host is looked up (this avoids the nondeterministic
		// behavior found in the original code when this happens).
		addr1 := urlHost(addr)
		if addr1 == addr {
			continue
		}
		if ac1, ok := f.Auths[addr1]; ok {
			if len(ac1.derivedFrom) == 0 {
				// Don't override an explicit entry.
				continue
			}
			ac = ac1
		}
		ac.derivedFrom = append(ac.derivedFrom, addr)
		slices.Sort(ac.derivedFrom)
		f.Auths[addr1] = ac
	}
	return f, nil
}

// urlHost returns the host part of a registry URL.
// Mimics [github.com/docker/docker/registry.ConvertToHostname]
// to keep the logic the same as that.
func urlHost(url string) string {
	stripped := url
	if strings.HasPrefix(url, "http://") {
		stripped = strings.TrimPrefix(url, "http://")
	} else if strings.HasPrefix(url, "https://") {
		stripped = strings.TrimPrefix(url, "https://")
	}

	hostName, _, _ := strings.Cut(stripped, "/")
	return hostName
}

// decodeAuth decodes a base64 encoded string and returns username and password
func decodeAuth(authStr string) (string, string, error) {
	s, err := base64.StdEncoding.DecodeString(authStr)
	if err != nil {
		return "", "", fmt.Errorf("invalid base64-encoded string")
	}
	username, password, ok := strings.Cut(string(s), ":")
	if !ok || username == "" {
		return "", "", errors.New("no username found")
	}
	// The zero-byte-trimming logic here mimics the logic in the
	// docker CLI configfile package.
	return username, strings.Trim(password, "\x00"), nil
}

// ExecHelper executes an external program to get the credentials from a native store.
// It implements [HelperRunner].
func ExecHelper(helperName string, serverURL string) (ConfigEntry, error) {
	return ExecHelperWithEnv(nil)(helperName, serverURL)
}

// ExecHelperWithEnv returns a [HelperRunner] that behaves like [ExecHelper]
// except that, if env is non-nil, it will be used as the set of environment
// variables to pass to the executed helper command. If env is nil,
// the current process's environment will be used.
func ExecHelperWithEnv(env []string) HelperRunner {
	return func(helperName string, serverURL string) (ConfigEntry, error) {
		var out bytes.Buffer
		cmd := exec.Command("docker-credential-"+helperName, "get")
		// TODO this doesn't produce a decent error message for
		// other helpers such as gcloud that print errors to stderr.
		cmd.Stdin = strings.NewReader(serverURL)
		cmd.Stdout = &out
		cmd.Stderr = &out
		cmd.Env = env
		if err := cmd.Run(); err != nil {
			if !errors.As(err, new(*exec.ExitError)) {
				if errors.Is(err, exec.ErrNotFound) {
					return ConfigEntry{}, fmt.Errorf("%w: %v", ErrHelperNotFound, err)
				}
				return ConfigEntry{}, fmt.Errorf("cannot run auth helper: %v", err)
			}
			t := strings.TrimSpace(out.String())
			if t == "credentials not found in native keychain" {
				return ConfigEntry{}, nil
			}
			return ConfigEntry{}, fmt.Errorf("error getting credentials: %s", t)
		}

		// helperCredentials defines the JSON encoding of the data printed
		// by credentials helper programs.
		type helperCredentials struct {
			Username string
			Secret   string
		}
		var creds helperCredentials
		if err := json.Unmarshal(out.Bytes(), &creds); err != nil {
			return ConfigEntry{}, err
		}
		if creds.Username == "<token>" {
			return ConfigEntry{
				RefreshToken: creds.Secret,
			}, nil
		}
		return ConfigEntry{
			Password: creds.Secret,
			Username: creds.Username,
		}, nil
	}
}
