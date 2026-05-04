package store

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xuyuanzhang1122/bililive-server-update/internal/model"
)

var ErrNotFound = errors.New("not found")

type FileStore struct {
	root string
	mu   sync.RWMutex
}

func NewFileStore(root string) (*FileStore, error) {
	if strings.TrimSpace(root) == "" {
		root = "./data"
	}
	if err := os.MkdirAll(filepath.Join(root, "backups"), 0755); err != nil {
		return nil, err
	}
	store := &FileStore{root: root}
	if _, err := os.Stat(store.catalogPath()); errors.Is(err, os.ErrNotExist) {
		if err := store.SaveCatalog(defaultCatalog()); err != nil {
			return nil, err
		}
	}
	return store, nil
}

func (s *FileStore) LoadCatalog() (model.Catalog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var catalog model.Catalog
	if err := readJSON(s.catalogPath(), &catalog); err != nil {
		return model.Catalog{}, err
	}
	return catalog, nil
}

func (s *FileStore) SaveCatalog(catalog model.Catalog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if catalog.UpdatedAt.IsZero() {
		catalog.UpdatedAt = time.Now().UTC()
	}
	return writeJSONAtomic(s.catalogPath(), catalog)
}

func (s *FileStore) SaveBackup(bundle model.BackupBundle) (model.BackupBundle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if bundle.ID == "" {
		bundle.ID = newBackupID()
	}
	if bundle.CreatedAt.IsZero() {
		bundle.CreatedAt = time.Now().UTC()
	}
	if bundle.SchemaVersion == 0 {
		bundle.SchemaVersion = 1
	}
	if err := validateBackup(bundle); err != nil {
		return model.BackupBundle{}, err
	}
	if err := writeJSONAtomic(s.backupPath(bundle.ID), bundle); err != nil {
		return model.BackupBundle{}, err
	}
	return bundle, nil
}

func (s *FileStore) LoadBackup(id string) (model.BackupBundle, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id = cleanID(id)
	if id == "" {
		return model.BackupBundle{}, ErrNotFound
	}
	var bundle model.BackupBundle
	if err := readJSON(s.backupPath(id), &bundle); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return model.BackupBundle{}, ErrNotFound
		}
		return model.BackupBundle{}, err
	}
	return bundle, nil
}

func (s *FileStore) catalogPath() string {
	return filepath.Join(s.root, "catalog.json")
}

func (s *FileStore) backupPath(id string) string {
	return filepath.Join(s.root, "backups", cleanID(id)+".json")
}

func defaultCatalog() model.Catalog {
	now := time.Now().UTC()
	return model.Catalog{
		UpdatedAt: now,
		Releases:  []model.ReleaseArtifact{},
		Tools: []model.ToolArtifact{
			{
				Name:      "ffmpeg",
				Version:   "stable",
				Stable:    true,
				UpdatedAt: now,
				Metadata: map[string]string{
					"config_key": "ffmpeg_path",
					"env_key":    "FFMPEG_PATH",
				},
			},
			{
				Name:      "headless-browser",
				Version:   "stable",
				Stable:    true,
				UpdatedAt: now,
				Metadata: map[string]string{
					"config_key": "headless_browser.path",
					"env_key":    "BILILIVE_HEADLESS_BROWSER_PATH",
				},
			},
		},
	}
}

func validateBackup(bundle model.BackupBundle) error {
	if bundle.Server.BaseURL == "" && bundle.Server.RPCBind == "" && bundle.Server.Port == 0 {
		return fmt.Errorf("server 配置至少需要 base_url、rpc_bind 或 port 之一")
	}
	for _, room := range bundle.Rooms {
		if !strings.HasPrefix(strings.TrimSpace(room.URL), "http://") &&
			!strings.HasPrefix(strings.TrimSpace(room.URL), "https://") {
			return fmt.Errorf("直播间 URL 非法: %s", room.URL)
		}
	}
	return nil
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func writeJSONAtomic(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func newBackupID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return strings.ToLower(fmt.Sprintf("%d", time.Now().UnixNano()))
	}
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf[:])
	return strings.ToLower(encoded)
}

func cleanID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
