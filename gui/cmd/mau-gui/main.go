// Mau GUI - P2P Social Network Client
package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mau-network/mau-gui-poc/internal/adapters/storage"
	"github.com/mau-network/mau-gui-poc/internal/app"
	"github.com/mau-network/mau-gui-poc/internal/domain/account"
	"github.com/mau-network/mau-gui-poc/internal/domain/config"
	"github.com/mau-network/mau-gui-poc/internal/domain/post"
	"github.com/mau-network/mau-gui-poc/internal/domain/server"
)

func main() {
	// Get data directory
	dataDir := getDataDir()

	// Initialize storage adapters
	configStore := storage.NewConfigStore(dataDir)
	accountStore := storage.NewAccountStore(dataDir)

	// Initialize domain managers
	configMgr := config.NewManager(configStore)
	accountMgr := account.NewManager(accountStore)

	// Initialize account (create if needed)
	if err := accountMgr.Init(); err != nil {
		log.Fatalf("Failed to initialize account: %v", err)
	}

	// Update config with account info
	accountInfo := accountMgr.Info()
	configMgr.Update(func(cfg *config.AppConfig) {
		// Add account if not exists
		exists := false
		for _, acc := range cfg.Accounts {
			if acc.Fingerprint == accountInfo.Fingerprint {
				exists = true
				break
			}
		}
		if !exists {
			cfg.Accounts = append(cfg.Accounts, config.AccountInfo{
				Name:        accountInfo.Name,
				Email:       accountInfo.Email,
				Fingerprint: accountInfo.Fingerprint,
				DataDir:     accountInfo.DataDir,
			})
			cfg.LastAccount = accountInfo.Fingerprint
		}
	})

	// Initialize post infrastructure
	postStore := storage.NewPostStore(accountMgr.Account())
	postCache := post.NewCache(100, 30*time.Minute)
	postMgr := post.NewManager(postStore, postCache)

	// Initialize server manager
	serverMgr := server.NewManager(server.Config{
		Account: accountMgr.Account(),
	})

	// Create and run application
	application := app.New(app.Config{
		ConfigMgr:  configMgr,
		AccountMgr: accountMgr,
		PostMgr:    postMgr,
		ServerMgr:  serverMgr,
	})

	os.Exit(application.Run(os.Args))
}

// getDataDir returns the application data directory
func getDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}
	return filepath.Join(home, ".local", "share", "mau-gui")
}
