package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "://github.com"
)

// Resource описывает учебный материал или сетевое устройство
type Resource struct {
	ID        int
	Name      string // Название (например, "Документация по BGP" или "Коммутатор Cisco Core")
	Category  string // DevOps, NetOps, SysAdmin
	TargetURL string // Куда перенаправлять (URL документации или веб-морды устройства)
	Clicks    int    // Сколько раз переходили (Счетчик / Учет ресурса)
	LastClick string // Дата последнего перехода
}

var db *sql.DB
var tpl *template.Template

func main() {
	// 1. Инициализация SQLite
	var err error
	db, err = sql.Open("sqlite3", "./devnet_sprint.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Создаем таблицу ресурсов, если её нет
	createTable := `CREATE TABLE IF NOT EXISTS resources (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		category TEXT,
		target_url TEXT,
		clicks INTEGER DEFAULT 0,
		last_click TEXT DEFAULT '-'
	);`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal(err)
	}

	// Заполняем начальными данными для DevOps и NetOps
	var count int
	db.QueryRow("SELECT COUNT(*) FROM resources").Scan(&count)
	if count == 0 {
		db.Exec("INSERT INTO resources (name, category, target_url) VALUES (?, ?, ?)", 
			"Kubernetes Networking Guide", "DevOps", "https://kubernetes.io")
		db.Exec("INSERT INTO resources (name, category, target_url) VALUES (?, ?, ?)", 
			"Cisco Ansible Automation", "NetOps", "https://cisco.com")
	}

	// 2. Загрузка HTML-шаблона
	tpl = template.Must(template.ParseFiles("index.html"))

	// 3. Маршруты
	http.HandleFunc("/", handleIndex)       // Главная страница со списком
	http.HandleFunc("/add", handleAdd)     // Добавление новой ссылки/устройства
	http.HandleFunc("/go/", handleRedirect) // Умный редирект с фиксацией прогресса

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Подсветка текущего стенда в логах (DevOps практика)
	envType := os.Getenv("ENV_TYPE")
	if envType == "" {
		envType = "local"
	}

	log.Printf("🚀 [%s] Приложение запущенно на порту %s...", envType, port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// Отображение главной страницы
func handleIndex(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, category, target_url, clicks, last_click FROM resources")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var list []Resource
	for rows.Next() {
		var res Resource
		if err := rows.Scan(&res.ID, &res.Name, &res.Category, &res.TargetURL, &res.Clicks, &res.LastClick); err != nil {
			log.Println(res)
			continue
		}
		list = append(list, res)
	}

	// Передаем в шаблон тип стенда, чтобы красиво вывести его на экран
	data := map[string]interface{}{
		"Env":       os.Getenv("ENV_TYPE"),
		"Resources": list,
	}

	tpl.Execute(w, data)
}

// Добавление нового ресурса (Задание для студентов)
func handleAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		name := r.FormValue("name")
		category := r.FormValue("category")
		url := r.FormValue("url")

		if name != "" && url != "" {
			_, err := db.Exec("INSERT INTO resources (name, category, target_url) VALUES (?, ?, ?)", name, category, url)
			if err != nil {
				log.Println("Ошибка записи в БД:", err)
			}
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Умный редирект: увеличиваем счетчик, пишем дату и перенаправляем
func handleRedirect(w http.ResponseWriter, r *http.Request) {
	// Получаем ID из URL (например, из /go/1 получаем "1")
	idStr := r.URL.Path[len("/go/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var targetURL string
	// Ищем оригинальную ссылку в БД
	err = db.QueryRow("SELECT target_url FROM resources WHERE id = ?", id).Scan(&targetURL)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Фиксируем прогресс в БД: инкрементируем счетчик и пишем текущее время
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	_, err = db.Exec("UPDATE resources SET clicks = clicks + 1, last_click = ? WHERE id = ?", currentTime, id)
	if err != nil {
		log.Println("Ошибка обновления прогресса:", err)
	}

	// Делаем HTTP-редирект пользователя на целевой ресурс
	http.Redirect(w, r, targetURL, http.StatusSeeOther)
}

