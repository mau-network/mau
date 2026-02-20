package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/mau-network/mau"
)

const (
	appID      = "com.mau.GuiPOC"
	appTitle   = "Mau - P2P Social Network"
	dummyEmail = "demo@mau.network"
	dummyName  = "Demo User"
)

type MauApp struct {
	app       *adw.Application
	account   *mau.Account
	dataDir   string
	mainStack *adw.ViewStack
	
	// UI Components
	homeView     *gtk.Box
	friendsView  *gtk.Box
	settingsView *gtk.Box
	postEntry    *gtk.TextView
	friendsList  *gtk.ListBox
	nameEntry    *gtk.Entry
	emailEntry   *gtk.Entry
}

func main() {
	app := adw.NewApplication(appID, 0)
	mauApp := &MauApp{
		app:     app,
		dataDir: filepath.Join(os.Getenv("HOME"), ".mau-gui-poc"),
	}

	app.ConnectActivate(func() {
		mauApp.activate()
	})

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}

func (m *MauApp) activate() {
	// Initialize or load account
	if err := m.initAccount(); err != nil {
		dialog := adw.NewMessageDialog(nil, "Error", "Failed to initialize account")
		dialog.AddResponse("ok", "OK")
		dialog.SetResponseAppearance("ok", adw.ResponseDefault)
		dialog.SetDefaultResponse("ok")
		dialog.SetBody(fmt.Sprintf("%v", err))
		dialog.Show()
		return
	}

	// Create main window
	window := adw.NewApplicationWindow(&m.app.Application)
	window.SetTitle(appTitle)
	window.SetDefaultSize(800, 600)

	// Create header bar with view switcher
	headerBar := adw.NewHeaderBar()
	
	// Create view switcher title
	viewSwitcher := adw.NewViewSwitcher()
	m.mainStack = adw.NewViewStack()
	viewSwitcher.SetStack(m.mainStack)
	
	headerBar.SetTitleWidget(viewSwitcher)

	// Create main content
	toolbarView := adw.NewToolbarView()
	toolbarView.AddTopBar(headerBar)
	toolbarView.SetContent(m.mainStack)

	// Build views
	m.buildHomeView()
	m.buildFriendsView()
	m.buildSettingsView()

	window.SetContent(toolbarView)
	window.Show()
}

func (m *MauApp) initAccount() error {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(m.dataDir, 0700); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	accountPath := filepath.Join(m.dataDir, ".mau", "account.pgp")
	
	// Check if account already exists
	if _, err := os.Stat(accountPath); os.IsNotExist(err) {
		// Create new account with dummy data (using "demo" as password)
		log.Println("Creating new account with dummy data...")
		
		acc, err := mau.NewAccount(m.dataDir, dummyName, dummyEmail, "demo")
		if err != nil {
			return fmt.Errorf("failed to create account: %w", err)
		}
		
		m.account = acc
		log.Printf("Account created successfully: %s (%s)", dummyName, dummyEmail)
	} else {
		// Account exists - create a new Account object pointing to existing data
		// Note: mau.LoadAccount doesn't exist, so we just create reference
		log.Println("Account directory exists...")
		// For POC, just create new account reference (it won't overwrite if exists)
		acc, err := mau.NewAccount(m.dataDir, dummyName, dummyEmail, "demo")
		if err != nil && err != mau.ErrAccountAlreadyExists {
			return fmt.Errorf("failed to access account: %w", err)
		}
		m.account = acc
		log.Println("Account accessed successfully")
	}

	return nil
}

func (m *MauApp) buildHomeView() {
	page := gtk.NewBox(gtk.OrientationVertical, 12)
	page.SetMarginTop(12)
	page.SetMarginBottom(12)
	page.SetMarginStart(12)
	page.SetMarginEnd(12)

	// Welcome label
	welcomeLabel := gtk.NewLabel(fmt.Sprintf("Welcome, %s!", dummyName))
	welcomeLabel.AddCSSClass("title-1")
	page.Append(welcomeLabel)

	// Post creation section
	postGroup := adw.NewPreferencesGroup()
	postGroup.SetTitle("Create a Post")
	postGroup.SetDescription("Share something with your network")

	// Text view for post content
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(true)
	scrolled.SetSizeRequest(-1, 150)
	
	m.postEntry = gtk.NewTextView()
	m.postEntry.SetWrapMode(gtk.WrapWord)
	m.postEntry.SetMarginTop(6)
	m.postEntry.SetMarginBottom(6)
	m.postEntry.SetMarginStart(6)
	m.postEntry.SetMarginEnd(6)
	
	scrolled.SetChild(m.postEntry)

	postRow := adw.NewActionRow()
	postRow.SetChild(scrolled)
	postGroup.Add(postRow)

	// Publish button
	publishBtn := gtk.NewButton()
	publishBtn.SetLabel("Publish")
	publishBtn.AddCSSClass("suggested-action")
	publishBtn.ConnectClicked(func() {
		m.publishPost()
	})

	btnBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	btnBox.SetHAlign(gtk.AlignEnd)
	btnBox.Append(publishBtn)
	
	page.Append(postGroup)
	page.Append(btnBox)

	// Add to stack
	m.mainStack.AddTitled(page, "home", "Home")
	m.homeView = page
}

func (m *MauApp) buildFriendsView() {
	page := gtk.NewBox(gtk.OrientationVertical, 12)
	page.SetMarginTop(12)
	page.SetMarginBottom(12)
	page.SetMarginStart(12)
	page.SetMarginEnd(12)

	// Friends list
	friendsGroup := adw.NewPreferencesGroup()
	friendsGroup.SetTitle("Friends")
	friendsGroup.SetDescription("Manage your network")

	m.friendsList = gtk.NewListBox()
	m.friendsList.AddCSSClass("boxed-list")
	m.loadFriendsList()

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(true)
	scrolled.SetChild(m.friendsList)

	page.Append(friendsGroup)
	page.Append(scrolled)

	// Add friend button
	addBtn := gtk.NewButton()
	addBtn.SetLabel("Add Friend")
	addBtn.AddCSSClass("suggested-action")
	addBtn.ConnectClicked(func() {
		m.showAddFriendDialog()
	})

	btnBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	btnBox.SetHAlign(gtk.AlignEnd)
	btnBox.Append(addBtn)
	page.Append(btnBox)

	m.mainStack.AddTitled(page, "friends", "Friends")
	m.friendsView = page
}

func (m *MauApp) buildSettingsView() {
	page := gtk.NewBox(gtk.OrientationVertical, 12)
	page.SetMarginTop(12)
	page.SetMarginBottom(12)
	page.SetMarginStart(12)
	page.SetMarginEnd(12)

	// Account settings
	accountGroup := adw.NewPreferencesGroup()
	accountGroup.SetTitle("Account Settings")
	accountGroup.SetDescription("Update your profile information")

	// Name entry
	nameRow := adw.NewActionRow()
	nameRow.SetTitle("Name")
	m.nameEntry = gtk.NewEntry()
	m.nameEntry.SetText(dummyName)
	m.nameEntry.SetHExpand(true)
	nameRow.AddSuffix(m.nameEntry)
	accountGroup.Add(nameRow)

	// Email entry
	emailRow := adw.NewActionRow()
	emailRow.SetTitle("Email")
	m.emailEntry = gtk.NewEntry()
	m.emailEntry.SetText(dummyEmail)
	m.emailEntry.SetHExpand(true)
	emailRow.AddSuffix(m.emailEntry)
	accountGroup.Add(emailRow)

	page.Append(accountGroup)

	// Fingerprint display
	infoGroup := adw.NewPreferencesGroup()
	infoGroup.SetTitle("Account Information")

	fpRow := adw.NewActionRow()
	fpRow.SetTitle("Fingerprint")
	if m.account != nil {
		fpRow.SetSubtitle(m.account.Fingerprint().String())
	}
	infoGroup.Add(fpRow)

	page.Append(infoGroup)

	// Save button
	saveBtn := gtk.NewButton()
	saveBtn.SetLabel("Save Changes")
	saveBtn.AddCSSClass("suggested-action")
	saveBtn.ConnectClicked(func() {
		m.saveSettings()
	})

	btnBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	btnBox.SetHAlign(gtk.AlignEnd)
	btnBox.Append(saveBtn)
	page.Append(btnBox)

	m.mainStack.AddTitled(page, "settings", "Settings")
	m.settingsView = page
}

func (m *MauApp) publishPost() {
	buffer := m.postEntry.Buffer()
	start := buffer.StartIter()
	end := buffer.EndIter()
	text := buffer.Text(start, end, false)

	if text == "" {
		m.showInfoDialog("Please enter some content to publish")
		return
	}

	// Create a simple SocialMediaPosting using schema.org format
	post := map[string]interface{}{
		"@context":  "https://schema.org",
		"@type":     "SocialMediaPosting",
		"headline":  "New Post",
		"articleBody": text,
		"datePublished": "",
		"author": map[string]string{
			"@type": "Person",
			"name":  dummyName,
			"email": dummyEmail,
		},
	}

	// In a real implementation, you would:
	// 1. Serialize the post to JSON
	// 2. Use m.account to sign and encrypt it
	// 3. Write it to the filesystem
	// 4. Announce it to peers
	
	log.Printf("Publishing post: %+v", post)
	
	m.showInfoDialog("Post published successfully!\n(In POC mode - not actually persisted)")
	buffer.SetText("")
}

func (m *MauApp) loadFriendsList() {
	// In a real implementation, load friends from .mau directory
	// For now, show placeholder
	if m.friendsList == nil {
		return
	}

	// Clear existing items (using RemoveAll for GTK4)
	m.friendsList.RemoveAll()

	// Add placeholder
	row := adw.NewActionRow()
	row.SetTitle("No friends yet")
	row.SetSubtitle("Add friends to start sharing")
	m.friendsList.Append(row)
}

func (m *MauApp) showAddFriendDialog() {
	dialog := adw.NewMessageDialog(
		m.app.ActiveWindow(),
		"Add Friend",
		"Enter friend's fingerprint or address",
	)
	dialog.AddResponse("cancel", "Cancel")
	dialog.AddResponse("add", "Add")
	dialog.SetResponseAppearance("add", adw.ResponseSuggested)
	dialog.SetDefaultResponse("add")
	dialog.SetCloseResponse("cancel")

	// In a real implementation, add entry field for fingerprint
	
	dialog.ConnectResponse(func(response string) {
		if response == "add" {
			// Add friend logic here
			m.showInfoDialog("Friend added!\n(In POC mode - not actually persisted)")
			m.loadFriendsList()
		}
	})

	dialog.Show()
}

func (m *MauApp) saveSettings() {
	name := m.nameEntry.Text()
	email := m.emailEntry.Text()

	// In a real implementation, update the account
	log.Printf("Saving settings: name=%s, email=%s", name, email)
	
	m.showInfoDialog("Settings saved!\n(In POC mode - changes not persisted)")
}

func (m *MauApp) showInfoDialog(message string) {
	dialog := adw.NewMessageDialog(
		m.app.ActiveWindow(),
		"Information",
		message,
	)
	dialog.AddResponse("ok", "OK")
	dialog.SetDefaultResponse("ok")
	dialog.SetCloseResponse("ok")
	dialog.Show()
}
