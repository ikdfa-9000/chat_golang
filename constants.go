package main

type appStatus int
type colorId string

const (
	// Статус приложения...
	appWelcomeStatus  appStatus = 0
	appMainStatus               = 1
	appChatListStatus           = 2
	appSettingsStatus           = 3
	// Цвета для текста
	colorReset  colorId = "\033[0m"
	colorRed            = "\033[31m"
	colorGreen          = "\033[32m"
	colorYellow         = "\033[33m"
	colorBlue           = "\033[34m"
	colorPurple         = "\033[35m"
	colorCyan           = "\033[36m"
	colorWhite          = "\033[37m"
)

type State struct {
	appState     appStatus
	messageColor colorId
	userColor    colorId
	dateColor    colorId
}
