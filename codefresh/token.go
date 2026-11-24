package codefresh

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/noperator/chromedb"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

var browserApps = map[string]struct{ app, profile string }{
	"chrome":   {"Google/Chrome", "Default"},
	"chromium": {"Chromium", "Default"},
	"brave":    {"BraveSoftware/Brave-Browser", "Default"},
	"edge":     {"Microsoft Edge", "Default"},
	"vivaldi":  {"Vivaldi", "Default"},
	"opera":    {"com.operasoftware.Opera", "Default"},
}

func userDataDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir(), "Library", "Application Support")
	case "linux":
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return xdg
		}
		return filepath.Join(homeDir(), ".config")
	case "windows":
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return local
		}
		return filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	default:
		return filepath.Join(homeDir(), ".config")
	}
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	h, _ := os.UserHomeDir()
	return h
}

func browserLevelDBPath(browser string) (string, error) {
	info, ok := browserApps[strings.ToLower(browser)]
	if !ok {
		return "", errors.Errorf("unsupported browser %q", browser)
	}
	var sub []string
	if info.profile != "" {
		sub = []string{info.profile}
	}
	sub = append(sub, "Local Storage", "leveldb")
	return filepath.Join(append([]string{userDataDir(), info.app}, sub...)...), nil
}

func AccessTokenFromBrowser(browser string) (string, error) {
	browser = lo.CoalesceOrEmpty(browser, "chrome")
	dbPath, err := browserLevelDBPath(browser)
	if err != nil {
		return "", err
	}
	return accessTokenFromLevelDB(dbPath)
}

func accessTokenFromLevelDB(dbPath string) (string, error) {
	db, err := chromedb.LoadLocalStorage(dbPath)
	if err != nil {
		return "", errors.WithMessage(err, "failed to open local storage")
	}
	defer db.Close()

	for _, r := range db.Records {
		if r.StorageKey == Host && r.ScriptKey == "accessToken" {
			return r.Decoded, nil
		}
	}
	return "", errors.Errorf("accessToken not found for origin %q", Host)
}

func RefreshTokenFromBrowser(browser string) (string, error) {
	browser = lo.CoalesceOrEmpty(browser, "chrome")
	dbPath, err := browserLevelDBPath(browser)
	if err != nil {
		return "", err
	}
	return refreshTokenFromLevelDB(dbPath)
}

func refreshTokenFromLevelDB(dbPath string) (string, error) {
	db, err := chromedb.LoadLocalStorage(dbPath)
	if err != nil {
		return "", errors.WithMessage(err, "failed to open local storage")
	}
	defer db.Close()

	for _, r := range db.Records {
		if r.StorageKey == Host && r.ScriptKey == "refreshToken" {
			return r.Decoded, nil
		}
	}
	return "", errors.Errorf("refreshToken not found for origin %q", Host)
}

func TokensFromBrowser(browser string) (accessToken, refreshToken string, err error) {
	browser = lo.CoalesceOrEmpty(browser, "chrome")
	dbPath, err := browserLevelDBPath(browser)
	if err != nil {
		return "", "", err
	}

	db, err := chromedb.LoadLocalStorage(dbPath)
	if err != nil {
		return "", "", errors.WithMessage(err, "failed to open local storage")
	}
	defer db.Close()

	for _, r := range db.Records {
		if r.StorageKey != Host {
			continue
		}
		switch r.ScriptKey {
		case "accessToken":
			accessToken = r.Decoded
		case "refreshToken":
			refreshToken = r.Decoded
		}
	}

	if accessToken == "" {
		return "", "", errors.Errorf("accessToken not found for origin %q", Host)
	}
	return accessToken, refreshToken, nil
}