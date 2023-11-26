package main

type appStatus int

// type colorId string
type scrollStat int
type User struct {
	id         int
	firstname  string
	secondname string
	nickname   string
	status     string
}

type ChatInList struct {
	name string
	id   int
}

type Message struct {
	user    string
	content string
	date    string
	id      int
	chat_id int
	isRead  bool
}

const (
	// Статус приложения...
	appWelcomeStatus     appStatus = 0
	appMainStatus                  = 1
	appChatListStatus              = 2
	appSettingsStatus              = 3
	appChatOpennedStatus           = 4
	// Цвета для текста
	colorReset  string = "\033[0m"
	colorRed           = "\033[31m"
	colorGreen         = "\033[32m"
	colorYellow        = "\033[33m"
	colorBlue          = "\033[34m"
	colorPurple        = "\033[35m"
	colorCyan          = "\033[36m"
	colorWhite         = "\033[37m"
	// Статус прокрутки
	scrollUp   scrollStat = 1
	scrollNo   scrollStat = 0
	scrollDown scrollStat = 2
)

type State struct {
	appState     appStatus
	messageColor string
	userColor    string
	dateColor    string
}
