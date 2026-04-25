package migrate_test

import (
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kevintrivedi/zoo-api/migrations"
)

// TestEmbeddedFS_ContainsExpectedMigrations is a cheap guard: if go:embed
// stops picking up the SQL files, this test catches it before integration.
func TestEmbeddedFS_ContainsExpectedMigrations(t *testing.T) {
	t.Parallel()

	var names []string
	err := fs.WalkDir(migrations.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(p, ".sql") {
			names = append(names, path.Base(p))
		}
		return nil
	})
	require.NoError(t, err)
	require.Contains(t, names, "00001_create_animals.sql")
}
