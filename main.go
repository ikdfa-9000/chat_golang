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

func main() {
	state := initState()
	var currUser User            // Текущий пользователь
	var intInput int             // Инпут на число
	var stringInput string       // Инпут на строку
	firstIdSeen := 0             // Первый айди, который видно в списке чатов
	maxChatOnePage := 10         // Количество видимых за 1 раз чатов
	lastIdSeen := maxChatOnePage // Айди последнего чата в видимом списке
	scrollStatus := scrollNo     // Режим скроллинга
	reader := bufio.NewReader(os.Stdin)
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
					// Надо бы сделать функцию, которая бы анализировала
					// введенную строку на: длину, наличие только некоторых символов
					// чтобы не было неожиданной хуеты. Так и сделаю
					nickname, _ = reader.ReadString('\n')
					fmt.Print(colorReset)
					fmt.Print("Введите свой пароль: ", colorGreen)
					password, _ = reader.ReadString('\n')
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
			// ДИЧАЙШАЯ ШАЙТАН МАШИНА прокрутка чатов МОНСТР КОД
			// САМЫЙ СТРАШНЫЙ КОД на свете
			lastIdSeen, firstIdSeen = printchats(chat_list, lastIdSeen, maxChatOnePage, scrollStatus, firstIdSeen)
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
				fmt.Scanf("%s\n", &stringInput)
				secondUserNick := stringInput
				fmt.Println("Введите название вашего чата: ")
				fmt.Scanf("%s\n", &stringInput)
				newChatName := stringInput
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
					lastIdSeen = 0
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
	*userToInitiate = initUser(idToInitiate, fname, sname, nick, status)
	getInfo.Close()
	fmt.Println("-------------------------------------------------------------------")
}

func printchats(chatlink []ChatInList, lastid int, maxChats int, scrolling scrollStat, firstid int) (int, int) {
	// TODO: ПОЛНОСТЬЮ ПЕРЕДЕЛАТЬ ПЕРЕПИСАТЬ но пока норм

	// Поведение этой штуки определяется статусом скроллинга (вниз, вверх, нет). Должно быть
	// легко в понимании и использовании, но сейчас 2:41 и я не совсем понимаю, что делаю
	// Но оно работает! Пиздец... Надо переделать
	i := lastid
	k := firstid
	if scrolling == scrollDown {
		if lastid >= len(chatlink)-1 {
			fmt.Println(colorRed, "Чатов больше нет! Прокручивать нечего", colorReset)
			i = 0
			lastid = 0
		}
		k = i
		for i < lastid+maxChats {
			if i == len(chatlink) {
				break
			}
			fmt.Println("Имя чата:", colorCyan, chatlink[i].name, colorReset, "номер чата:", i+1, "id чата:", chatlink[i].id)
			i++
		}
	} else if scrolling == scrollUp {
		if i <= maxChats {
			fmt.Println(colorRed, "Чатов больше нет! Прокручивать нечего", colorReset)
			i = 0
			lastid = 0
			k = i
			for i < lastid+maxChats {
				if i == len(chatlink) {
					break
				}
				fmt.Println("Имя чата:", colorCyan, chatlink[i].name, colorReset, "номер чата:", i+1, "id чата:", chatlink[i].id)
				i++
			}
			return i, k
		}
		for l := firstid - maxChats; l < firstid; l++ {
			fmt.Println("Имя чата:", colorCyan, chatlink[l].name, colorReset, "номер чата:", l+1, "id чата:", chatlink[l].id)
		}
		k = firstid - maxChats
		i = firstid
	} else {
		for m := firstid; m < i; m++ {
			fmt.Println("Имя чата:", colorCyan, chatlink[m].name, colorReset, "номер чата:", m+1, "id чата:", chatlink[m].id)
		}
	}
	return i, k
}
