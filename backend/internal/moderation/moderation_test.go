package moderation

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupModerationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&RestrictedKeyword{}))
	return db
}

func TestCheckContentBlockedKeywordRejects(t *testing.T) {
	db := setupModerationTestDB(t)
	require.NoError(t, db.Create(&RestrictedKeyword{Keyword: "forbidden-keyword", Severity: "block", MessageEn: "blocked", IsActive: true}).Error)
	InitStore(db, nil)
	t.Setenv("FEATURE_MODERATION_AUTO", "BLOCK")

	blocked, reason := CheckContent("title", "contains forbidden-keyword here")
	require.True(t, blocked)
	require.Equal(t, "blocked", reason)
}

func TestCheckContentSafePasses(t *testing.T) {
	db := setupModerationTestDB(t)
	InitStore(db, nil)
	t.Setenv("FEATURE_MODERATION_AUTO", "BLOCK")

	blocked, _ := CheckContent("safe title", "normal clean description")
	require.False(t, blocked)
}

func TestCheckContentCacheHitNoDBQuery(t *testing.T) {
	db := setupModerationTestDB(t)
	require.NoError(t, db.Create(&RestrictedKeyword{Keyword: "cache-key", Severity: "block", MessageEn: "blocked", IsActive: true}).Error)
	InitStore(db, nil)
	t.Setenv("FEATURE_MODERATION_AUTO", "BLOCK")

	blocked, _ := CheckContent("title", "cache-key")
	require.True(t, blocked)

	require.NoError(t, db.Where("1=1").Delete(&RestrictedKeyword{}).Error)
	blocked, reason := CheckContent("title", "cache-key")
	require.True(t, blocked)
	require.Equal(t, "blocked", reason)
}
