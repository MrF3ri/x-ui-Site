package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func RunMigrations(db *sql.DB, dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(b)) == "" {
			continue
		}
		if _, err = db.Exec(string(b)); err != nil {
			return err
		}
	}
	return nil
}
