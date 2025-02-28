package secrets

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type Secret string

type SecretVault interface {
	Get(key string) (Secret, error)
	Set(key string, val Secret) error
}

type DirVault struct {
	dir string
}

func (s Secret) String() string {
	return string(s)
}

func (s Secret) LogValue() slog.Value {
	return slog.StringValue("REDACTED")
}

func (src *DirVault) Get(key string) (Secret, error) {
	buf, err := os.ReadFile(fmt.Sprintf("%s/%s", src.dir, key))
	if err != nil {
		return "", err
	}
	return Secret(buf), nil
}

// This is here so a private key for JWT can be generated
// and still exists after a service restart.
func (src *DirVault) Set(key string, val Secret) error {
	filePath := filepath.Join(src.dir, key)
	err := os.WriteFile(filePath, []byte(val), 0600)
	if err != nil {
		return fmt.Errorf("unable to write to file %s: %v", filePath, err)
	}
	return nil
}

func NewDirVault(dir string) (*DirVault, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist")
	} else if err != nil {
		return nil, fmt.Errorf("error checking directory: %v", err)
	}
	return &DirVault{dir: dir}, nil
}
