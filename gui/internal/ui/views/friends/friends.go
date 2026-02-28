// Package friends provides the friends management view.
package friends

import (
	"fmt"
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/mau-network/mau-gui-poc/internal/adapters/notification"
	"github.com/mau-network/mau-gui-poc/internal/domain/account"
	"github.com/mau-network/mau-gui-poc/internal/ui"
)

// View handles the friends management view
type View struct {
	accountMgr *account.Manager
	notifier   *notification.Notifier
	gtkApp     *adw.Application

	// UI components
	page        *gtk.Box
	friendsList *gtk.ListBox
}

// Config holds view configuration
type Config struct {
	AccountMgr *account.Manager
	Notifier   *notification.Notifier
	GtkApp     *adw.Application
}

// New creates a new friends view
func New(cfg Config) *View {
	v := &View{
		accountMgr: cfg.AccountMgr,
		notifier:   cfg.Notifier,
		gtkApp:     cfg.GtkApp,
	}
	v.buildUI()
	return v
}

// Widget returns the view's widget
func (v *View) Widget() *gtk.Box {
	return v.page
}

// Refresh reloads the friends list
func (v *View) Refresh() {
	v.friendsList.RemoveAll()

	keyring, err := v.accountMgr.Account().ListFriends()
	if err != nil {
		row := adw.NewActionRow()
		row.SetTitle("Error loading friends")
		v.friendsList.Append(row)
		return
	}

	friends := keyring.FriendsSet()
	if len(friends) == 0 {
		row := adw.NewActionRow()
		row.SetTitle("No friends yet")
		v.friendsList.Append(row)
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

		v.friendsList.Append(row)
	}
}

// buildUI creates and returns the view widget
func (v *View) buildUI() {
	v.page = gtk.NewBox(gtk.OrientationVertical, 12)
	v.page.SetMarginTop(12)
	v.page.SetMarginBottom(12)
	v.page.SetMarginStart(12)
	v.page.SetMarginEnd(12)

	friendsGroup := adw.NewPreferencesGroup()
	friendsGroup.SetTitle("Friends")
	friendsGroup.SetDescription("Manage your network")

	v.friendsList = ui.NewBoxedListBox()

	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(true)
	scrolled.SetChild(v.friendsList)

	v.page.Append(friendsGroup)
	v.page.Append(scrolled)

	// Add button
	addBtn := gtk.NewButton()
	addBtn.SetLabel("Add Friend")
	addBtn.AddCSSClass("suggested-action")
	addBtn.ConnectClicked(func() {
		v.showAddFriendDialog()
	})

	btnBox := gtk.NewBox(gtk.OrientationHorizontal, 6)
	btnBox.SetHAlign(gtk.AlignEnd)
	btnBox.Append(addBtn)
	v.page.Append(btnBox)

	v.Refresh()
}

func (v *View) showAddFriendDialog() {
	window := v.gtkApp.ActiveWindow()

	dialog := gtk.NewDialog()
	dialog.SetTitle("Add Friend")
	dialog.SetModal(true)
	dialog.SetTransientFor(window)
	dialog.SetDefaultSize(ui.DialogDefaultWidth, ui.DialogDefaultHeight)

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
	scrolled.SetSizeRequest(-1, ui.TextViewMinHeight)

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
				v.notifier.ShowError(ui.DialogValidateError, "Please enter a PGP public key")
				dialog.Destroy()
				return
			}

			if err := v.addFriend(keyData); err != nil {
				v.notifier.ShowError("Failed to Add Friend", fmt.Sprintf("Could not add friend: %v", err))
			} else {
				v.notifier.ShowToast(ui.ToastFriendAdded)
				v.Refresh()
			}
		}
		dialog.Destroy()
	})

	dialog.Show()
}

func (v *View) addFriend(armoredKey string) error {
	// Validate PGP key format
	if err := validatePGPKey(armoredKey); err != nil {
		return err
	}

	reader := strings.NewReader(armoredKey)
	_, err := v.accountMgr.Account().AddFriend(reader)
	return err
}

// validatePGPKey validates PGP armored key format
func validatePGPKey(armoredKey string) error {
	key := strings.TrimSpace(armoredKey)

	if key == "" {
		return fmt.Errorf(ui.ErrInvalidPGPKey)
	}

	// Check for PGP armor headers
	if !strings.Contains(key, "-----BEGIN PGP PUBLIC KEY BLOCK-----") {
		return fmt.Errorf(ui.ErrMissingPGPHeaders)
	}

	if !strings.Contains(key, "-----END PGP PUBLIC KEY BLOCK-----") {
		return fmt.Errorf(ui.ErrMissingPGPHeaders)
	}

	// Basic length check (typical PGP keys are 1-10KB)
	if len(key) < 200 {
		return fmt.Errorf(ui.ErrPGPKeyTooShort)
	}

	if len(key) > ui.MaxPGPKeySize {
		return fmt.Errorf(ui.ErrPGPKeyTooLarge)
	}

	return nil
}
