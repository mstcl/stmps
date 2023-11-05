// Copyright 2023 The STMPS Authors
// SPDX-License-Identifier: GPL-3.0-only

package main

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spezifisch/stmps/logger"
	"github.com/spezifisch/stmps/mpvplayer"
	"github.com/spezifisch/stmps/subsonic"
)

// struct contains all the updatable elements of the Ui
type Ui struct {
	app   *tview.Application
	pages *tview.Pages

	// top bar
	startStopStatus *tview.TextView
	playerStatus    *tview.TextView

	// bottom bar
	menuWidget *MenuWidget

	// browser page
	browserPage *BrowserPage

	// queue page
	queuePage *QueuePage

	// playlist page
	playlistPage *PlaylistPage

	// log page
	logPage *LogPage

	// modals
	addToPlaylistList *tview.List
	messageBox        *tview.Modal

	starIdList map[string]struct{}

	eventLoop *eventLoop
	mpvEvents chan mpvplayer.UiEvent

	playlists  []subsonic.SubsonicPlaylist
	connection *subsonic.SubsonicConnection
	player     *mpvplayer.Player
	logger     *logger.Logger
}

const (
	// page identifiers (use these instead of hardcoding page names for showing/hiding)
	PageBrowser   = "browser"
	PageQueue     = "queue"
	PagePlaylists = "playlists"
	PageLog       = "log"

	PageDeletePlaylist = "deletePlaylist"
	PageNewPlaylist    = "newPlaylist"
	PageAddToPlaylist  = "addToPlaylist"
	PageMessageBox     = "messageBox"
)

func InitGui(indexes *[]subsonic.SubsonicIndex,
	playlists *[]subsonic.SubsonicPlaylist,
	connection *subsonic.SubsonicConnection,
	player *mpvplayer.Player,
	logger *logger.Logger) (ui *Ui) {
	ui = &Ui{
		starIdList: map[string]struct{}{},

		eventLoop: nil, // initialized by initEventLoops()
		mpvEvents: make(chan mpvplayer.UiEvent, 5),

		playlists:  *playlists,
		connection: connection,
		player:     player,
		logger:     logger,
	}

	ui.initEventLoops()

	ui.app = tview.NewApplication()
	ui.pages = tview.NewPages()

	// status text at the top
	statusLeft := fmt.Sprintf("[::b]%s[::-] v%s", clientName, clientVersion)
	ui.startStopStatus = tview.NewTextView().SetText(statusLeft).
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetScrollable(false)

	statusRight := formatPlayerStatus(0, 0, 0)
	ui.playerStatus = tview.NewTextView().SetText(statusRight).
		SetTextAlign(tview.AlignRight).
		SetDynamicColors(true).
		SetScrollable(false)

	ui.menuWidget = NewMenuWidget(ui)

	// same as 'playlistList' except for the addToPlaylistModal
	// - we need a specific version of this because we need different keybinds
	ui.addToPlaylistList = tview.NewList().ShowSecondaryText(false)

	// message box for small notes
	ui.messageBox = tview.NewModal().
		SetText("hi there").
		SetBackgroundColor(tcell.ColorBlack)
	ui.messageBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		ui.pages.HidePage(PageMessageBox)
		return event
	})

	// top bar: status text
	topBarFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(ui.startStopStatus, 0, 1, false).
		AddItem(ui.playerStatus, 20, 0, false)

	// browser page
	ui.browserPage = ui.createBrowserPage(indexes)

	// queue page
	ui.queuePage = ui.createQueuePage()

	// playlist page
	ui.playlistPage = ui.createPlaylistPage()

	// log page
	ui.logPage = ui.createLogPage()

	ui.pages.AddPage(PageBrowser, ui.browserPage.Root, true, true).
		AddPage(PageQueue, ui.queuePage.Root, true, false).
		AddPage(PagePlaylists, ui.playlistPage.Root, true, false).
		AddPage(PageDeletePlaylist, ui.playlistPage.DeletePlaylistModal, true, false).
		AddPage(PageNewPlaylist, ui.playlistPage.NewPlaylistModal, true, false).
		AddPage(PageAddToPlaylist, ui.browserPage.AddToPlaylistModal, true, false).
		AddPage(PageMessageBox, ui.messageBox, true, false).
		AddPage(PageLog, ui.logPage.Root, true, false)

	rootFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(topBarFlex, 1, 0, false).
		AddItem(ui.pages, 0, 1, true).
		AddItem(ui.menuWidget.Root, 1, 0, false)

	// add main input handler
	rootFlex.SetInputCapture(ui.handlePageInput)

	ui.app.SetRoot(rootFlex, true).
		SetFocus(rootFlex).
		EnableMouse(true)

	return ui
}

func (ui *Ui) Run() error {
	// receive events from mpv wrapper
	ui.player.RegisterEventConsumer(ui)

	// run gui/background event handler
	ui.runEventLoops()

	// run mpv event handler
	go ui.player.EventLoop()

	// gui main loop (blocking)
	return ui.app.Run()
}

func (ui *Ui) showMessageBox(text string) {
	ui.pages.ShowPage(PageMessageBox)
	ui.messageBox.SetText(text)
	ui.app.SetFocus(ui.messageBox)
}
