package repository_test

import (
	"database/sql"
	"testing"

	"go-file-server/internal/repository"
	"go-file-server/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupConfigRepo(t *testing.T) (*repository.SQLiteCloudConfigRepo, *sql.DB) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	repo := repository.NewSQLiteCloudConfigRepo(db)
	return repo, db
}

func TestConfigRepo_ListEnabled(t *testing.T) {
	repo, _ := setupConfigRepo(t)

	configs, err := repo.ListEnabled()
	require.NoError(t, err)
	assert.Len(t, configs, 3)

	names := make(map[string]bool)
	for _, c := range configs {
		names[c.ConfigName] = true
		assert.True(t, c.IsEnabled)
	}
	assert.True(t, names["service_name"])
	assert.True(t, names["upload_chunk_size"])
	assert.True(t, names["storage_limit"])
}

func TestConfigRepo_ListAllNotDeleted(t *testing.T) {
	repo, _ := setupConfigRepo(t)

	configs, err := repo.ListAllNotDeleted()
	require.NoError(t, err)
	assert.Len(t, configs, 3)

	for _, c := range configs {
		assert.Greater(t, c.ID, 0, "each config should have a valid ID")
	}
}

func TestConfigRepo_Update(t *testing.T) {
	repo, _ := setupConfigRepo(t)

	// Get the initial configs to find a valid ID
	configs, err := repo.ListAllNotDeleted()
	require.NoError(t, err)
	require.NotEmpty(t, configs)

	targetID := configs[0].ID
	newValue := "Test Service Name"
	enabled := true

	rowsAffected, err := repo.Update(targetID, &newValue, &enabled, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	// Verify by listing enabled configs
	configsAfter, err := repo.ListEnabled()
	require.NoError(t, err)

	var found bool
	for _, c := range configsAfter {
		if c.ConfigName == configs[0].ConfigName {
			require.NotNil(t, c.ConfigValue)
			assert.Equal(t, newValue, *c.ConfigValue)
			found = true
		}
	}
	assert.True(t, found, "updated config should appear in enabled list")
}

func TestConfigRepo_Update_SoftDelete(t *testing.T) {
	repo, _ := setupConfigRepo(t)

	configs, err := repo.ListAllNotDeleted()
	require.NoError(t, err)
	require.NotEmpty(t, configs)

	targetID := configs[0].ID
	deleted := true

	rowsAffected, err := repo.Update(targetID, nil, nil, &deleted)
	require.NoError(t, err)
	assert.Equal(t, int64(1), rowsAffected)

	// Should not appear in ListAllNotDeleted
	configsAfter, err := repo.ListAllNotDeleted()
	require.NoError(t, err)
	assert.Len(t, configsAfter, 2)
}
