package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"

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

type ChatInList struct {
	name string
	id   int
}

func HashPassword(password string) (string, error) { // Хэширование пароля
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool { // Проверка пароля на верность
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func readInput(scanLink *bufio.Scanner, isOnlyEnglish bool, symbolsMin int, symbolsMax int) string {
	// goodSymbols := "AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz_-0123456789" // Допустимые символы, для паролей и ников
	for {
		scanLink.Scan()
		input := scanLink.Text()
		if len(input) >= symbolsMin && len(input) <= symbolsMax { // Если длина инпута нормальная, то:
			// if !isOnlyEnglish { // Если символы могут быть любые, то возврат инпута
			return input
			// TODO: Доделать

			// } else {
			// 	notAcceptable := false
			// 	for _, symbolInput := range input { // Проход по символам инпута
			// 		for _, symbolGood := range goodSymbols { // Проход по допустимым символам
			// 			if symbolInput == symbolGood {
			// 				break
			// 			}
			// 			if symbolGood == '9' {
			// 				notAcceptable = true
			// 			}
			// 		}
			// 		if notAcceptable {
			// 			fmt.Println(colorRed, "Введенное значение может иметь лишь буквы английского алфавита, тире, дефис и цифры!", colorReset)
			// 			break
			// 		}
			// 	}
			// }
		} else {
			fmt.Println(colorRed, "Введенное значение должно быть больше", symbolsMin, "и меньше", symbolsMax, "символов по длине!", colorWhite)
		}
	}
}

func main() {
	state := initState()
	var currUser User                // Текущий пользователь
	var intInput int                 // Инпут на число
	firstIdSeen := 0                 // Первый айди, который видно в списке чатов
	maxChatOnePage := 10             // Количество видимых за 1 раз чатов
	lastIdSeen := maxChatOnePage - 1 // Айди последнего чата в видимом списке
	scrollStatus := scrollNo         // Режим скроллинга
	scanner := bufio.NewScanner(os.Stdin)
	// Подключение и инициализация таблицы
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
	statement, _ = db.Prepare("CREATE TABLE IF NOT EXISTS chat_list (chat_id INTEGER PRIMARY KEY, creator_id INTEGER, chat_name TEXT)") // Для групповых чатов, если будут
	statement.Exec()
	fmt.Println("Подключение успешно!")
	for {
		switch state.appState {
		case appWelcomeStatus:
			// Вход/регистрация
			fmt.Println("Выберите действие:")
			fmt.Println(colorGreen, "1. Логин", colorReset)
			fmt.Println(colorGreen, "2. Регистрация", colorReset)
			fmt.Scanf("%d\n", &intInput) // Для чисел оставлю fmt.Scanf, чтобы потом не конвертировать строку каждый раз в число
			if intInput < 1 || intInput > 2 {
				fmt.Println("Введите 1 или 2")
			} else {
				// Переменные, нужные для ввода
				var nickname string
				var password string
				var firstname string
				var secondname string
				defaultStatus := "Живу" // Чёта там
				if intInput == 1 {
					fmt.Print("Введите свой никнейм: ", colorGreen)
					nickname = readInput(scanner, true, 4, 32)
					fmt.Print(colorReset)
					fmt.Print("Введите свой пароль: ", colorGreen)
					password = readInput(scanner, true, 10, 32)
					fmt.Print(colorReset)
					// Вход в аккаунт
					checkLogin, err := db.Query("SELECT nickname FROM chat_users WHERE nickname = (?);", nickname)
					if err != nil {
						fmt.Println("Что-то не то...")
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
					firstname = readInput(scanner, false, 0, 32)
					fmt.Println("Введите Вашу фамилию: ")
					secondname = readInput(scanner, false, 0, 32)
					// ПРОВЕРКА НИКНЕЙМА
					fmt.Println("Введите никнейм для аккаунта: ")
					nickname = readInput(scanner, true, 4, 32)
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
					password = readInput(scanner, true, 6, 32)
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

			fmt.Println("Велкам", currUser.firstname, currUser.secondname, "AKA", currUser.nickname)
			fmt.Println("Выберите действие:")
			fmt.Println(colorCyan, "1. Списки чатов", colorReset)
			fmt.Println(colorYellow, "2. Настройки приложения", colorReset)
			fmt.Println(colorRed, "3. Выйти из аккаунта", colorReset)
			fmt.Println("Чтобы выйти из программы, нажмите Esc или Ctrl+C")
			fmt.Scanf("%d\n", &intInput)
			if intInput < 1 && intInput > 3 {
				fmt.Println("Нормально действие выбери да")
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
			fmt.Println("--------------------------------------------------")
			// Проходимся по списку чатов и подбираем те айди чатов, где есть текущий пользователь
			chatIdsQuery, _ := db.Query("SELECT chat_id FROM chat_members WHERE user_id = (?);", currUser.id)
			var chatIdentificator int              // Айди чата, нужен для работы с таблицей
			chat_list := make([]ChatInList, 0, 10) // Слайс из всех чатов пользователя
			var chatTemp ChatInList                // Переменная, которую в последствии будем впихивать в слайс
			for chatIdsQuery.Next() {
				chatIdsQuery.Scan(&chatIdentificator)
				// Поиск в таблице чатов названия по айди
				chatNamesQuery, _ := db.Query("SELECT chat_name FROM chat_list WHERE chat_id = (?) ;", chatIdentificator)
				chatNamesQuery.Next()
				chatNamesQuery.Scan(&chatTemp.name) // Формируем чат для слайса
				chatTemp.id = chatIdentificator
				chat_list = append(chat_list, chatTemp) // Впихиваем чат в слайс
				chatNamesQuery.Close()                  // Закрываем Query. Хз зачем :D
			}
			if len(chat_list) < maxChatOnePage {
				lastIdSeen = len(chat_list) - 1
			}
			firstIdSeen, lastIdSeen = printchats(chat_list, lastIdSeen, maxChatOnePage, scrollStatus, firstIdSeen)
			scrollStatus = scrollNo
			fmt.Println("--------------------------------------------------")
			chatIdsQuery.Close()
			fmt.Println("Хотите сделать новый чат с человеком? Нажмите 0!")
			fmt.Println("Для загрузки следующей страницы чатов введите -1")
			fmt.Println("Для загрузки предыдущей страницы чатов введите -2")
			fmt.Print("Ну, куда отправимся? ")
			fmt.Scanf("%d\n", &intInput)
			switch intInput {
			case 0:
				fmt.Print("Введите никнейм человека, с которым вы хотите создать чат: ")
				secondUserNick := readInput(scanner, true, 4, 32)
				if secondUserNick == currUser.nickname {
					fmt.Println(colorRed, "Нельзя сделать чат с самим собой (зачем?)", colorReset)
				} else {
					fmt.Println("Введите название вашего чата: ")
					newChatName := readInput(scanner, false, 4, 64)
					_, err = db.Exec("INSERT INTO chat_list (creator_id, chat_name) VALUES(?1, ?2);", currUser.id, newChatName)
					if err != nil {
						log.Fatal(err)
					}
					// Получаем айди человека по никнейму
					rowsId, _ := db.Query("SELECT id FROM chat_users WHERE nickname = (?) ", secondUserNick)
					if !rowsId.Next() {
						fmt.Println("Такой пользователь не найден!")
					} else {
						var secondUserId int
						rowsId.Scan(&secondUserId)
						rowsId.Close()
						// Получаем айди чата, созданного текущим пользователем, сортировка по убыванию,
						// ибо айди только что созданного чата всегда будет наибольшим
						newChatIdQuery, _ := db.Query("SELECT chat_id FROM chat_list WHERE creator_id = (?) ORDER BY chat_id DESC;", currUser.id)
						newChatIdQuery.Next()
						var newChatId int
						newChatIdQuery.Scan(&newChatId)
						newChatIdQuery.Close()
						_, _ = db.Exec("INSERT INTO chat_members (chat_id, user_id) VALUES(?1, ?2);", newChatId, secondUserId)
						_, _ = db.Exec("INSERT INTO chat_members (chat_id, user_id) VALUES(?1, ?2);", newChatId, currUser.id)
						firstIdSeen = 0
						lastIdSeen = maxChatOnePage
					}
				}
			case -1:
				scrollStatus = scrollDown
			case -2:
				scrollStatus = scrollUp
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
	fmt.Println(idToInitiate, fname, sname, nick)
	*userToInitiate = initUser(idToInitiate, fname, sname, nick, status)
	getInfo.Close()
	fmt.Println("-------------------------------------------------------------------")
}

func printchats(chatlink []ChatInList, lastid int, maxChats int, scrolling scrollStat, firstid int) (int, int) {
	if len(chatlink) == 0 {
		fmt.Println(colorYellow, "У вас ещё нет никаких чатов", colorReset)
	}
	if firstid == lastid && firstid == 0 {
		fmt.Println("Имя чата:", colorCyan, chatlink[lastid].name, colorReset, "номер чата:", lastid+1, "id чата:", chatlink[lastid].id)
	} else {
		switch scrolling {
		case scrollDown:
			if lastid == len(chatlink)-1 {
				fmt.Println(colorRed, "Больше прокручивать нечего! Загружаем первые чаты.", colorReset)
				firstid = 0
				if len(chatlink)-1 <= maxChats-1 {
					lastid = len(chatlink) - 1
				} else {
					lastid = maxChats - 1
				}
				for i := firstid; i <= lastid; i++ {
					if i == len(chatlink) {
						break
					}
					fmt.Println("Имя чата:", colorCyan, chatlink[i].name, colorReset, "номер чата:", i+1, "id чата:", chatlink[i].id)
				}
			} else {
				firstid = lastid + 1
				lastid = lastid + maxChats
				for i := firstid; i <= lastid; i++ {
					fmt.Println("Имя чата:", colorCyan, chatlink[i].name, colorReset, "номер чата:", i+1, "id чата:", chatlink[i].id)
					if i == len(chatlink)-1 {
						lastid = i
						break
					}
				}
			}
		case scrollUp:
			if firstid == 0 {
				fmt.Println(colorRed, "Больше прокручивать нечего!", colorReset)
				firstid = 0
				if len(chatlink)-1 <= maxChats-1 {
					lastid = len(chatlink) - 1
				} else {
					lastid = maxChats - 1
				}
				for i := firstid; i <= lastid; i++ {
					if i == len(chatlink) {
						break
					}
					fmt.Println("Имя чата:", colorCyan, chatlink[i].name, colorReset, "номер чата:", i+1, "id чата:", chatlink[i].id)
				}
			} else {
				firstid = lastid - firstid - 1
				if firstid < 0 {
					firstid = 0
				}
				lastid = firstid + maxChats - 1
				for i := firstid; i <= lastid; i++ {
					fmt.Println("Имя чата:", colorCyan, chatlink[i].name, colorReset, "номер чата:", i+1, "id чата:", chatlink[i].id)
				}
			}
		case scrollNo:
			if len(chatlink)-1 <= maxChats-1 {
				lastid = len(chatlink) - 1
			} else {
				lastid = maxChats - 1
			}
			for i := firstid; i <= lastid; i++ {
				if i == len(chatlink) {
					break
				}
				fmt.Println("Имя чата:", colorCyan, chatlink[i].name, colorReset, "номер чата:", i+1, "id чата:", chatlink[i].id)
			}
		}
	}
	fmt.Println(firstid, lastid)
	return firstid, lastid
}
