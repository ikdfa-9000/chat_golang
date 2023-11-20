package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	id         int
	firstname  string
	secondname string
	nickname   string
	status     string
}

// type ChatInList struct {
// 	name string
// 	id   int
// }

func HashPassword(password string) (string, error) { // Хэширование пароля
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool { // Проверка пароля на верность
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func main() {
	state := initState()
	var currUser User
	var intInput int
	var stringInput string
	fmt.Println("Подключение к таблице...")
	db, err := sql.Open("sqlite3", "database.db")
	if err != nil {
		fmt.Println("Ошибка подключения к базе данных! Закрываем программу.")
		log.Fatal(err)
	}
	defer db.Close()
	statement, _ := db.Prepare("CREATE TABLE IF NOT EXISTS chat_users (id INTEGER PRIMARY KEY, firstname TEXT, secondname TEXT, nickname TEXT, password TEXT, status TEXT)")
	statement.Exec()
	statement, _ = db.Prepare("CREATE TABLE IF NOT EXISTS messages_list (message_id INTEGER PRIMARY KEY, chat_id INTEGER, user_id INTEGER, content TEXT, date_sent DATETIME)")
	statement.Exec()
	statement, _ = db.Prepare("CREATE TABLE IF NOT EXISTS messages_stat (message_id INTEGER PRIMARY KEY, user_id INTEGER, is_read BOOLEAN )")
	statement.Exec()
	statement, _ = db.Prepare("CREATE TABLE IF NOT EXISTS chat_members (chat_id INTEGER, user_id INTEGER)")
	statement.Exec()
	statement, err = db.Prepare("CREATE TABLE IF NOT EXISTS chat_list (chat_id INTEGER PRIMARY KEY, creator_id INTEGER, chat_name TEXT)") // Для групповых чатов, если будут
	statement.Exec()
	fmt.Println("Подключение успешно!")
	for {
		switch state.appState {
		case appWelcomeStatus:
			fmt.Println("Выберите действие:")
			fmt.Println(colorGreen, "1. Логин", colorReset)
			fmt.Println(colorGreen, "2. Регистрация", colorReset)
			fmt.Scanf("%d\n", &intInput)
			if intInput < 1 || intInput > 2 {
				fmt.Println("Это так не работает)")
			} else {
				// Переменные, нужные для ввода
				var nickname string
				var password string
				var firstname string
				var secondname string
				defaultStatus := "Живу"
				if intInput == 1 {
					fmt.Print("Введите свой никнейм: ", colorGreen)
					fmt.Scanf("%s\n", &nickname)
					fmt.Print(colorReset)
					fmt.Print("Введите свой пароль: ", colorGreen)
					fmt.Scanf("%s\n", &password)
					fmt.Print(colorReset)
					// Вход в аккаунт
					checkLogin, err := db.Query("SELECT nickname FROM chat_users WHERE nickname = (?);", nickname)
					if err != nil {
						fmt.Println("Пиздец")
						log.Fatal(err)
					}
					if checkLogin.Next() {
						checkLogin.Close()
						fmt.Println("Логин существует в базе данных")
						checkLogin, err = db.Query("SELECT password FROM chat_users WHERE nickname = (?);", nickname) // Получаем хэш пользователя с определенным ником
						if err != nil {
							log.Fatal(err)
						}
						var hashPassword string
						checkLogin.Next()
						checkLogin.Scan(&hashPassword)
						if CheckPasswordHash(password, hashPassword) {
							checkLogin.Close()
							fmt.Println("Вход завершён")
							state.appState = appMainStatus
							authorize(db, nickname, &currUser, defaultStatus)
						} else {
							fmt.Println("Пароль введён неверно")
						}
					} else {
						fmt.Println("Такого логина нет в базе данных")
					}
				} else {
					fmt.Println("Введите Ваше имя: ")
					fmt.Scanf("%s\n", &firstname)
					fmt.Println("Введите Вашу фамилию: ")
					fmt.Scanf("%s\n", &secondname)
					// ПРОВЕРКА НИКНЕЙМА
					fmt.Println("Введите никнейм для аккаунта: ")
					fmt.Scanf("%s\n", &nickname)
					// Смотрим, есть ли этот никнейм в таблице chat_users
					checkLogin, err := db.Query("SELECT nickname FROM chat_users WHERE nickname = (?);", nickname)
					if err != nil {
						fmt.Println("Пиздец")
						log.Fatal(err)
					}
					for {
						if checkLogin.Next() {
							checkLogin.Close()
							fmt.Println("Этот ник занят! Введите новый: ")
							fmt.Scanf("%s\n", &nickname)
							checkLogin, _ = db.Query("SELECT nickname FROM chat_users WHERE nickname = (?);", nickname)
						} else {
							break
						}
					}
					checkLogin.Close()
					fmt.Println("Введите пароль для аккаунта: ")
					fmt.Scanf("%s\n", &password)
					password, _ = HashPassword(password)
					_, err = db.Exec("INSERT INTO chat_users (firstname, secondname, nickname, password, status) VALUES(?1, ?2, ?3, ?4, ?5);", firstname, secondname, nickname, password, defaultStatus)
					if err != nil {
						log.Fatal(err)
					}
					fmt.Println("Регистрация прошла успешно")
					state.appState = appMainStatus
					authorize(db, nickname, &currUser, defaultStatus)
				}
			}
		case appMainStatus:
			fmt.Println("Велкам,", state.userColor, currUser.firstname, currUser.secondname, "AKA", currUser.nickname, colorReset)
			fmt.Println("Выберите действие:")
			fmt.Println(colorCyan, "1. Списки чатов", colorReset)
			fmt.Println(colorYellow, "2. Настройки приложения", colorReset)
			fmt.Println(colorRed, "3. Выйти из аккаунта", colorReset)
			fmt.Println("Чтобы выйти из программы, нажмите Esc или Ctrl+C")
			fmt.Scanf("%d\n", &intInput)
			if intInput < 1 && intInput > 3 {
				fmt.Println("Нормально действие выбери да")
				break
			} else {
				switch intInput {
				case 1:
					state.appState = appChatListStatus
				case 2:
					state.appState = appSettingsStatus
				case 3:
					currUser.id = 0
					currUser.firstname = "empty"
					currUser.secondname = "empty"
					currUser.status = "empty"
					currUser.nickname = "empty"
					state.appState = appWelcomeStatus
				}
			}
		case appChatListStatus:
			fmt.Println("Список чатов: ")
			// Проходимся по списку чатов и подбираем те айди чатов, где есть текущий пользователь
			chatIdsQuery, _ := db.Query("SELECT chat_id FROM chat_members WHERE user_id = (?);", currUser.id)
			var chatIdentificator int
			var chatName string
			for chatIdsQuery.Next() {
				chatIdsQuery.Scan(&chatIdentificator)
				// Поиск в таблице чатов названия по айди
				chatNamesQuery, _ := db.Query("SELECT chat_name FROM chat_list WHERE chat_id = (?) ORDER BY chat_id DESC;", chatIdentificator)
				chatNamesQuery.Next()
				chatNamesQuery.Scan(&chatName)
				chatNamesQuery.Close()
				// Вывод списка чатов на экран
				fmt.Println("Имя чата:", chatName, "id чата:", chatIdentificator)
			}
			chatIdsQuery.Close()
			fmt.Println("Хотите сделать новый чат с человеком? Нажмите 0!")
			fmt.Print("Ну, куда отправимся? ")
			fmt.Scanf("%d\n", &intInput)
			if intInput == 0 {
				fmt.Print("Введите никнейм человека, с которым вы хотите создать чат: ")
				fmt.Scanf("%s\n", &stringInput)
				secondUserNick := stringInput
				fmt.Println("Введите название вашего чата: ")
				fmt.Scanf("%s\n", &stringInput)
				newChatName := stringInput
				_, _ = db.Exec("INSERT INTO chat_list (creator_id, chat_name) VALUES(?1, ?2);", currUser.id, newChatName)
				rowsId, _ := db.Query("SELECT id FROM chat_users WHERE nickname = (?) ", secondUserNick)
				rowsId.Next()
				var secondUserId int
				rowsId.Scan(&secondUserId)
				rowsId.Close()
				newChatIdQuery, _ := db.Query("SELECT chat_id FROM chat_list WHERE creator_id = (?) ORDER BY chat_id DESC;", currUser.id)
				newChatIdQuery.Next()
				var newChatId int
				newChatIdQuery.Scan(&newChatId)
				newChatIdQuery.Close()
				_, _ = db.Exec("INSERT INTO chat_members (chat_id, user_id) VALUES(?1, ?2);", newChatId, secondUserId)
				_, _ = db.Exec("INSERT INTO chat_members (chat_id, user_id) VALUES(?1, ?2);", newChatId, currUser.id)

			}
		case appSettingsStatus:
		}
	}
}
func initState() State { // Инициализация статуса приложения
	return State{
		appState:     appWelcomeStatus,
		messageColor: colorWhite,
		userColor:    colorWhite,
		dateColor:    colorWhite,
	}
}

func initUser(identific int, name string, secname string, nick string, stat string) User {
	return User{
		id:         identific,
		firstname:  name,
		secondname: secname,
		nickname:   nick,
		status:     stat,
	}
}

func authorize(database *sql.DB, nick string, userToInitiate *User, status string) {
	getInfo, err := database.Query("SELECT id, firstname, secondname FROM chat_users WHERE nickname = (?);", nick)
	if err != nil {
		log.Fatal(err)
	}
	getInfo.Next()
	var idToInitiate int
	var fname string
	var sname string
	getInfo.Scan(&idToInitiate, &fname, &sname)
	*userToInitiate = initUser(idToInitiate, fname, sname, nick, status)
	getInfo.Close()
	fmt.Println("-------------------------------------------------------------------")
}
