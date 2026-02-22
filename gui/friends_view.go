package main

import (
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/mau-network/mau"
)

// FriendsView handles the friends management view
type FriendsView struct {
	app         *MauApp
	page        *gtk.Box
	friendsList *gtk.ListBox
}

// NewFriendsView creates a new friends view
func NewFriendsView(app *MauApp) *FriendsView {
	return &FriendsView{app: app}
}

// Build creates and returns the view widget
func (fv *FriendsView) Build() *gtk.Box {
	fv.initializePage()
	fv.addFriendsGroup()
	fv.addFriendsScrollArea()
	fv.addAddButton()
	fv.Refresh()
	return fv.page
}

func (fv *FriendsView) initializePage() {
	fv.page = gtk.NewBox(gtk.OrientationVertical, 12)
	fv.page.SetMarginTop(12)
	fv.page.SetMarginBottom(12)
	fv.page.SetMarginStart(12)
	fv.page.SetMarginEnd(12)
}

func (fv *FriendsView) addFriendsGroup() {
	friendsGroup := adw.NewPreferencesGroup()
	friendsGroup.SetTitle("Friends")
	friendsGroup.SetDescription("Manage your network")
	fv.page.Append(friendsGroup)
}

func (fv *FriendsView) addFriendsScrollArea() {
	fv.friendsList = NewBoxedListBox()
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(true)
	scrolled.SetChild(fv.friendsList)
	fv.page.Append(scrolled)
}

func (fv *FriendsView) addAddButton() {
	addBtn := gtk.NewButton()
	addBtn.SetLabel("Add Friend")
	addBtn.AddCSSClass("suggested-action")
	addBtn.ConnectClicked(func() {
		fv.showAddFriendDialog()
	})
	btnBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	btnBox.SetHAlign(gtk.AlignEnd)
	btnBox.Append(addBtn)
	fv.page.Append(btnBox)
}

// Refresh reloads the friends list
func (fv *FriendsView) Refresh() {
	fv.friendsList.RemoveAll()
	keyring, err := fv.loadFriendKeyring()
	if err != nil {
		fv.showError()
		return
	}
	fv.displayFriends(keyring.FriendsSet())
}

func (fv *FriendsView) loadFriendKeyring() (*mau.Keyring, error) {
	return fv.app.accountMgr.Account().ListFriends()
}

func (fv *FriendsView) showError() {
	row := adw.NewActionRow()
	row.SetTitle("Error loading friends")
	fv.friendsList.Append(row)
}

func (fv *FriendsView) displayFriends(friends []*mau.Friend) {
	if len(friends) == 0 {
		fv.showEmptyState()
		return
	}
	for _, friend := range friends {
		fv.friendsList.Append(fv.createFriendRow(friend))
	}
}

func (fv *FriendsView) showEmptyState() {
	row := adw.NewActionRow()
	row.SetTitle("No friends yet")
	fv.friendsList.Append(row)
}

func (fv *FriendsView) createFriendRow(friend *mau.Friend) *adw.ActionRow {
	row := adw.NewActionRow()
	row.SetTitle(friend.Name())
	row.SetSubtitle(fv.formatFriendSubtitle(friend))
	icon := gtk.NewImageFromIconName("avatar-default-symbolic")
	row.AddPrefix(icon)
	return row
}

func (fv *FriendsView) formatFriendSubtitle(friend *mau.Friend) string {
	subtitle := friend.Email()
	if subtitle == "" {
		subtitle = friend.Fingerprint().String()[:16] + "..."
	}
	return subtitle
}

func (fv *FriendsView) showAddFriendDialog() {
	dialog := fv.createDialog()
	box := fv.createDialogContent()
	textView := fv.createTextView(box)
	dialog.SetChild(box)
	fv.addDialogButtons(dialog)
	fv.connectDialogResponse(dialog, textView)
	dialog.Show()
}

func (fv *FriendsView) createDialog() *gtk.Dialog {
	window := fv.app.app.ActiveWindow()
	dialog := gtk.NewDialog()
	dialog.SetTitle("Add Friend")
	dialog.SetModal(true)
	dialog.SetTransientFor(window)
	dialog.SetDefaultSize(dialogDefaultWidth, dialogDefaultHeight)
	return dialog
}

func (fv *FriendsView) createDialogContent() *gtk.Box {
	box := gtk.NewBox(gtk.OrientationVertical, 12)
	box.SetMarginTop(12)
	box.SetMarginBottom(12)
	box.SetMarginStart(12)
	box.SetMarginEnd(12)
	label := gtk.NewLabel("Enter friend's PGP public key (armored format):")
	label.SetHAlign(gtk.AlignStart)
	box.Append(label)
	return box
}

func (fv *FriendsView) createTextView(box *gtk.Box) *gtk.TextView {
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(true)
	scrolled.SetSizeRequest(-1, textViewMinHeight)
	textView := gtk.NewTextView()
	textView.SetWrapMode(gtk.WrapWord)
	scrolled.SetChild(textView)
	box.Append(scrolled)
	return textView
}

func (fv *FriendsView) addDialogButtons(dialog *gtk.Dialog) {
	dialog.AddButton("Cancel", int(gtk.ResponseCancel))
	dialog.AddButton("Add Friend", int(gtk.ResponseAccept))
}

func (fv *FriendsView) connectDialogResponse(dialog *gtk.Dialog, textView *gtk.TextView) {
	dialog.ConnectResponse(func(responseID int) {
		if responseID == int(gtk.ResponseAccept) {
			fv.handleAddFriend(dialog, textView)
		} else {
			dialog.Destroy()
		}
	})
}

func (fv *FriendsView) handleAddFriend(dialog *gtk.Dialog, textView *gtk.TextView) {
	keyData := fv.extractKeyData(textView)
	if keyData == "" {
		fv.app.ShowError(dialogValidateError, "Please enter a PGP public key")
		dialog.Destroy()
		return
	}
	fv.processAddFriend(keyData)
	dialog.Destroy()
}

func (fv *FriendsView) extractKeyData(textView *gtk.TextView) string {
	buffer := textView.Buffer()
	start := buffer.StartIter()
	end := buffer.EndIter()
	return buffer.Text(start, end, false)
}

func (fv *FriendsView) processAddFriend(keyData string) {
	if err := fv.addFriend(keyData); err != nil {
		fv.app.ShowError("Failed to Add Friend", fmt.Sprintf("Could not add friend: %v", err))
	} else {
		fv.app.showToast(toastFriendAdded)
		fv.Refresh()
	}
}

func (fv *FriendsView) addFriend(armoredKey string) error {
	if err := validatePGPKey(armoredKey); err != nil {
		return err
	}
	reader := strings.NewReader(armoredKey)
	_, err := fv.app.accountMgr.Account().AddFriend(reader)
	return err
}

// validatePGPKey validates PGP armored key format
func validatePGPKey(armoredKey string) error {
	key := strings.TrimSpace(armoredKey)
	if err := checkKeyNotEmpty(key); err != nil {
		return err
	}
	if err := checkPGPHeaders(key); err != nil {
		return err
	}
	return checkKeyLength(key)
}

func checkKeyNotEmpty(key string) error {
	if key == "" {
		return fmt.Errorf(errInvalidPGPKey)
	}
	return nil
}

func checkPGPHeaders(key string) error {
	if !strings.Contains(key, "-----BEGIN PGP PUBLIC KEY BLOCK-----") {
		return fmt.Errorf(errMissingPGPHeaders)
	}
	if !strings.Contains(key, "-----END PGP PUBLIC KEY BLOCK-----") {
		return fmt.Errorf(errMissingPGPHeaders)
	}
	return nil
}

func checkKeyLength(key string) error {
	if len(key) < 200 {
		return fmt.Errorf(errPGPKeyTooShort)
	}
	if len(key) > maxPGPKeySize {
		return fmt.Errorf(errPGPKeyTooLarge)
	}
	return nil
}
