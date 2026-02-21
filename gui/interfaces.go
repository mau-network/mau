package main

import (
	"github.com/mau-network/mau"
)

// ConfigStore defines the interface for configuration management
type ConfigStore interface {
	Get() AppConfig
	Update(updater func(*AppConfig)) error
	Load() error
	Save() error
}

// AccountStore defines the interface for account management
type AccountStore interface {
	Init() error
	Account() *mau.Account
	Info() AccountInfo
}

// PostStore defines the interface for post management
type PostStore interface {
	Save(post Post) error
	Load(file *mau.File) (Post, error)
	List(fingerprint mau.Fingerprint, limit int) ([]*mau.File, error)
}

// ToastNotifier defines the interface for showing toast messages
type ToastNotifier interface {
	ShowToast(message string)
	ShowError(title, message string)
}

// ServerController defines the interface for P2P server control
type ServerController interface {
	Start() error
	Stop() error
	IsRunning() bool
}

// Ensure concrete types implement interfaces
var (
	_ ConfigStore      = (*ConfigManager)(nil)
	_ AccountStore     = (*AccountManager)(nil)
	_ PostStore        = (*PostManager)(nil)
	_ ToastNotifier    = (*MauApp)(nil)
	_ ServerController = (*MauApp)(nil)
)
