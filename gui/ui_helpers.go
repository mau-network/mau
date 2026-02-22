package main

import (
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// UI Helper Functions

// escapeMarkup escapes GTK markup characters (&, <, >, ", ')
func escapeMarkup(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// NewBoxedListBox creates a styled ListBox with the "boxed-list" CSS class
func NewBoxedListBox() *gtk.ListBox {
	list := gtk.NewListBox()
	list.AddCSSClass("boxed-list")
	return list
}

// NewScrolledListBox creates a ListBox in a ScrolledWindow
func NewScrolledListBox() (*gtk.ScrolledWindow, *gtk.ListBox) {
	list := NewBoxedListBox()
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(true)
	scrolled.SetChild(list)
	return scrolled, list
}

// NewPreferencesGroup creates an adw.PreferencesGroup with title
func NewPreferencesGroup(title string) *adw.PreferencesGroup {
	group := adw.NewPreferencesGroup()
	group.SetTitle(title)
	return group
}

// NewActionRowWithIcon creates an ActionRow with icon prefix
func NewActionRowWithIcon(title, subtitle, iconName string) *adw.ActionRow {
	row := adw.NewActionRow()
	row.SetTitle(title)
	if subtitle != "" {
		row.SetSubtitle(subtitle)
	}
	if iconName != "" {
		icon := gtk.NewImageFromIconName(iconName)
		row.AddPrefix(icon)
	}
	return row
}
