package mau

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncState(t *testing.T) {
	tmpDir := t.TempDir()
	accountDir := path.Join(tmpDir, "test_account")
	
	// Create account
	account, err := NewAccount(accountDir, "Test User", "test@example.com", "testpass123")
	require.NoError(t, err)
	require.NotNil(t, account)
	
	// Create test fingerprint
	fpr, err := FingerprintFromString("0123456789ABCDEF0123456789ABCDEF01234567")
	require.NoError(t, err)

	t.Run("GetLastSyncTime returns zero time when no sync exists", func(t *testing.T) {
		lastSync := account.GetLastSyncTime(fpr)
		assert.True(t, lastSync.IsZero(), "Expected zero time for first sync")
	})

	t.Run("UpdateLastSyncTime creates sync state file", func(t *testing.T) {
		syncTime := time.Now().UTC().Truncate(time.Second)
		
		err := account.UpdateLastSyncTime(fpr, syncTime)
		assert.NoError(t, err)
		
		// Verify file was created
		stateFile := syncStateFile(accountDir)
		_, err = os.Stat(stateFile)
		assert.NoError(t, err, "sync_state.json should exist")
	})

	t.Run("GetLastSyncTime returns updated time", func(t *testing.T) {
		syncTime := time.Now().UTC().Truncate(time.Second)
		
		err := account.UpdateLastSyncTime(fpr, syncTime)
		require.NoError(t, err)
		
		retrievedTime := account.GetLastSyncTime(fpr)
		assert.Equal(t, syncTime, retrievedTime)
	})

	t.Run("Multiple friends tracked independently", func(t *testing.T) {
		fpr1, err := FingerprintFromString("1111111111111111111111111111111111111111")
		require.NoError(t, err)
		fpr2, err := FingerprintFromString("2222222222222222222222222222222222222222")
		require.NoError(t, err)
		
		time1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		time2 := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
		
		err = account.UpdateLastSyncTime(fpr1, time1)
		require.NoError(t, err)
		err = account.UpdateLastSyncTime(fpr2, time2)
		require.NoError(t, err)
		
		assert.Equal(t, time1, account.GetLastSyncTime(fpr1))
		assert.Equal(t, time2, account.GetLastSyncTime(fpr2))
	})

	t.Run("UpdateLastSyncTime overwrites previous time", func(t *testing.T) {
		fpr3, err := FingerprintFromString("3333333333333333333333333333333333333333")
		require.NoError(t, err)
		
		time1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		time2 := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
		
		err = account.UpdateLastSyncTime(fpr3, time1)
		require.NoError(t, err)
		assert.Equal(t, time1, account.GetLastSyncTime(fpr3))
		
		err = account.UpdateLastSyncTime(fpr3, time2)
		require.NoError(t, err)
		assert.Equal(t, time2, account.GetLastSyncTime(fpr3))
	})

	t.Run("Handles missing sync state file gracefully", func(t *testing.T) {
		// Create new account in different directory
		tmpDir2 := t.TempDir()
		accountDir2 := path.Join(tmpDir2, "test_account2")
		account2, err := NewAccount(accountDir2, "Test User 2", "test2@example.com", "testpass456")
		require.NoError(t, err)
		
		fpr4, err := FingerprintFromString("4444444444444444444444444444444444444444")
		require.NoError(t, err)
		
		// Should return zero time when file doesn't exist
		lastSync := account2.GetLastSyncTime(fpr4)
		assert.True(t, lastSync.IsZero())
	})

	t.Run("Preserves existing data when updating", func(t *testing.T) {
		fpr5, err := FingerprintFromString("5555555555555555555555555555555555555555")
		require.NoError(t, err)
		fpr6, err := FingerprintFromString("6666666666666666666666666666666666666666")
		require.NoError(t, err)
		
		time5 := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
		time6 := time.Date(2024, 4, 1, 12, 0, 0, 0, time.UTC)
		
		// Add first friend
		err = account.UpdateLastSyncTime(fpr5, time5)
		require.NoError(t, err)
		
		// Add second friend - should not overwrite first
		err = account.UpdateLastSyncTime(fpr6, time6)
		require.NoError(t, err)
		
		// Both should still be present
		assert.Equal(t, time5, account.GetLastSyncTime(fpr5))
		assert.Equal(t, time6, account.GetLastSyncTime(fpr6))
	})
}
