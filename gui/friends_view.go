package main

import (
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
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
	fv.page = gtk.NewBox(gtk.OrientationVertical, 12)
	fv.page.SetMarginTop(12)
	fv.page.SetMarginBottom(12)
	fv.page.SetMarginStart(12)
	fv.page.SetMarginEnd(12)

	friendsGroup := adw.NewPreferencesGroup()
	friendsGroup.SetTitle("Friends")
	friendsGroup.SetDescription("Manage your network")

	fv.friendsList = NewBoxedListBox()

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(true)
	scrolled.SetChild(fv.friendsList)

	fv.page.Append(friendsGroup)
	fv.page.Append(scrolled)

	// Add button
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

	fv.Refresh()

	return fv.page
}

// Refresh reloads the friends list
func (fv *FriendsView) Refresh() {
	fv.friendsList.RemoveAll()

	keyring, err := fv.app.accountMgr.Account().ListFriends()
	if err != nil {
		row := adw.NewActionRow()
		row.SetTitle("Error loading friends")
		fv.friendsList.Append(row)
		return
	}

	friends := keyring.FriendsSet()
	if len(friends) == 0 {
		row := adw.NewActionRow()
		row.SetTitle("No friends yet")
		fv.friendsList.Append(row)
		return
	}

	for _, friend := range friends {
		row := adw.NewActionRow()
		row.SetTitle(friend.Name())

		subtitle := friend.Email()
		if subtitle == "" {
			subtitle = friend.Fingerprint().String()[:16] + "..."
		}
		row.SetSubtitle(subtitle)

		icon := gtk.NewImageFromIconName("avatar-default-symbolic")
		row.AddPrefix(icon)

		fv.friendsList.Append(row)
	}
}

func (fv *FriendsView) showAddFriendDialog() {
	window := fv.app.app.ActiveWindow()

	dialog := gtk.NewDialog()
	dialog.SetTitle("Add Friend")
	dialog.SetModal(true)
	dialog.SetTransientFor(window)
	dialog.SetDefaultSize(500, -1)

	box := gtk.NewBox(gtk.OrientationVertical, 12)
	box.SetMarginTop(12)
	box.SetMarginBottom(12)
	box.SetMarginStart(12)
	box.SetMarginEnd(12)

	label := gtk.NewLabel("Enter friend's PGP public key (armored format):")
	label.SetHAlign(gtk.AlignStart)
	box.Append(label)

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(true)
	scrolled.SetSizeRequest(-1, 200)

	textView := gtk.NewTextView()
	textView.SetWrapMode(gtk.WrapWord)
	scrolled.SetChild(textView)
	box.Append(scrolled)

	dialog.SetChild(box)

	dialog.AddButton("Cancel", int(gtk.ResponseCancel))
	dialog.AddButton("Add Friend", int(gtk.ResponseAccept))

	dialog.ConnectResponse(func(responseID int) {
		if responseID == int(gtk.ResponseAccept) {
			buffer := textView.Buffer()
			start := buffer.StartIter()
			end := buffer.EndIter()
			keyData := buffer.Text(start, end, false)

			if keyData == "" {
				fv.app.ShowError(dialogValidateError, "Please enter a PGP public key")
				dialog.Destroy()
				return
			}

			if err := fv.addFriend(keyData); err != nil {
				fv.app.ShowError("Failed to Add Friend", fmt.Sprintf("Could not add friend: %v", err))
			} else {
				fv.app.showToast(toastFriendAdded)
				fv.Refresh()
			}
		}
		dialog.Destroy()
	})

	dialog.Show()
}

func (fv *FriendsView) addFriend(armoredKey string) error {
	// Validate PGP key format
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
	
	if key == "" {
		return fmt.Errorf(errInvalidPGPKey)
	}

	// Check for PGP armor headers
	if !strings.Contains(key, "-----BEGIN PGP PUBLIC KEY BLOCK-----") {
		return fmt.Errorf(errMissingPGPHeaders)
	}

	if !strings.Contains(key, "-----END PGP PUBLIC KEY BLOCK-----") {
		return fmt.Errorf(errMissingPGPHeaders)
	}

	// Basic length check (typical PGP keys are 1-10KB)
	if len(key) < 200 {
		return fmt.Errorf(errPGPKeyTooShort)
	}

	if len(key) > 50000 {
		return fmt.Errorf(errPGPKeyTooLarge)
	}

	return nil
}
