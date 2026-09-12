package cache_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"canvaslms-gui/internal/cache"
)

func TestNewSQLiteStore_CreatesDB(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cache.db")
	store, err := cache.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	require.NotNil(t, store)
	require.NoError(t, store.Close())
}

func TestSQLiteStore_SetGet(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cache.db")
	store, err := cache.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	type record struct {
		ID   int
		Name string
	}
	want := record{ID: 42, Name: "answer"}
	require.NoError(t, store.Set("rec", want, time.Hour))

	var got record
	hit, err := store.Get("rec", &got)
	require.NoError(t, err)
	assert.True(t, hit)
	assert.Equal(t, want, got)
}

func TestSQLiteStore_Get_Miss(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cache.db")
	store, err := cache.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	var got string
	hit, err := store.Get("missing", &got)
	require.NoError(t, err)
	assert.False(t, hit)
}

func TestSQLiteStore_Get_Expired(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cache.db")
	store, err := cache.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Set("old", "value", -time.Hour))

	var got string
	hit, err := store.Get("old", &got)
	require.NoError(t, err)
	assert.False(t, hit)
}

func TestSQLiteStore_Delete_Exact(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cache.db")
	store, err := cache.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Set("k", "v", time.Hour))
	require.NoError(t, store.Delete("k"))

	var got string
	hit, err := store.Get("k", &got)
	require.NoError(t, err)
	assert.False(t, hit)
}

func TestSQLiteStore_Delete_Pattern(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cache.db")
	store, err := cache.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Set("course:1:students", []int{1}, time.Hour))
	require.NoError(t, store.Set("course:1:assignments", []int{2}, time.Hour))
	require.NoError(t, store.Set("course:2:students", []int{3}, time.Hour))

	require.NoError(t, store.Delete(cache.PatternCourse(1)))

	var out []int
	hit, _ := store.Get("course:1:students", &out)
	assert.False(t, hit)
	hit, _ = store.Get("course:1:assignments", &out)
	assert.False(t, hit)

	hit, err = store.Get("course:2:students", &out)
	require.NoError(t, err)
	assert.True(t, hit)
	assert.Equal(t, []int{3}, out)
}

func TestSQLiteStore_Purge(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cache.db")
	store, err := cache.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Set("stale", "x", -time.Hour))
	require.NoError(t, store.Set("fresh", "y", time.Hour))

	require.NoError(t, store.Purge())

	var got string
	hit, _ := store.Get("stale", &got)
	assert.False(t, hit)

	hit, err = store.Get("fresh", &got)
	require.NoError(t, err)
	assert.True(t, hit)
	assert.Equal(t, "y", got)
}

func TestSQLiteStore_Overwrite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cache.db")
	store, err := cache.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Set("key", "first", time.Hour))
	require.NoError(t, store.Set("key", "second", time.Hour))

	var got string
	hit, err := store.Get("key", &got)
	require.NoError(t, err)
	assert.True(t, hit)
	assert.Equal(t, "second", got)
}

func TestSQLiteStore_Get_CorruptValue(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cache.db")
	store, err := cache.NewSQLiteStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.Set("bad", "not-json", time.Hour))

	// Manually overwrite with invalid JSON so we can test unmarshal failure.
	// We can't do this through the public API, but we can test that the
	// error path exists by using a type mismatch.
	require.NoError(t, store.Set("bad", "string", time.Hour))

	var got int
	hit, err := store.Get("bad", &got)
	require.Error(t, err)
	assert.False(t, hit)
	assert.Contains(t, err.Error(), "unmarshal")
}
