package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

// Конфигурация подключения к БД
const (
	dbUser     = "pashapay"
	dbPassword = "Q1w2e3r4!@#"
	dbHost     = "192.168.46.4"
	dbPort     = "3306"
	dbName     = "report"
)

// Глобальная переменная для пула подключений
var db *sql.DB

// Инициализация подключения к базе данных
func initDB() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPassword, dbHost, dbPort, dbName)
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	// Проверка подключения
	return db.Ping()
}

// executeQuery выполняет SQL‑запрос с параметрами и возвращает срез карт с данными.
func executeQuery(query string, params ...interface{}) ([]map[string]interface{}, error) {
	rows, err := db.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := make([]map[string]interface{}, 0)
	// Создаем срез для хранения указателей на значения
	for rows.Next() {
		// Создаем срез для хранения значений всех колонок
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		// Формируем карту для строки
		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := columns[i]
			// Если значение []byte, то преобразуем в строку
			b, ok := val.([]byte)
			if ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		results = append(results, rowMap)
	}

	return results, nil
}

// Функция для рендеринга JSON-ответа
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// Парсинг даты из строки (формат "2006-01-02")
func parseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

// ---------------------- Математические вспомогательные функции ----------------------

// percentile вычисляет p‑й процентиль для среза чисел.
func percentile(data []float64, p float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)
	pos := p / 100 * float64(len(sorted)-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return sorted[lower]
	}
	weight := pos - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// mean вычисляет среднее значение среза чисел.
func mean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

// min возвращает минимальное значение в срезе.
func min(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	m := data[0]
	for _, v := range data {
		if v < m {
			m = v
		}
	}
	return m
}

// toFloat64 пытается преобразовать значение к float64.
func toFloat64(val interface{}) float64 {
	switch v := val.(type) {
	case nil:
		return 0
	case int64:
		return float64(v)
	case int:
		return float64(v)
	case float64:
		return v
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0
		}
		return f
	default:
		return 0
	}
}

// ---------------------- HANDLER‑ФУНКЦИИ ----------------------

// Handler для главной страницы (рендеринг шаблона)
func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Путь к шаблонам
	tmplPath := filepath.Join("flask-dashboard", "app", "templates", "dashboard.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Передаем переменную column_select_js=true в шаблон
	data := map[string]interface{}{
		"column_select_js": true,
	}
	tmpl.Execute(w, data)
}

// Handler для daily данных (/daily/data/{tab})
func getDailyDataHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tab := vars["tab"]

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Please provide start_date and end_date"})
		return
	}

	log.Printf("Debug getDailyDataHandler: tab=%s, start_date=%s, end_date=%s", tab, startDate, endDate)

	queries := map[string]string{
		"calls": `
            SELECT DATE(c.enter_queue_date) AS report_date, COUNT(*) AS total_calls
            FROM call_report c
            WHERE DATE(c.enter_queue_date) BETWEEN ? AND ?
              AND (c.type = 'in' OR c.type = 'abandon')
            GROUP BY report_date
            ORDER BY report_date;`,
		"aht": `
            SELECT DATE(c.enter_queue_date) AS report_date, ROUND(AVG(c.call_duration), 2) AS avg_call_duration
            FROM call_report c
            WHERE DATE(c.enter_queue_date) BETWEEN ? AND ?
            GROUP BY report_date
            ORDER BY report_date;`,
		"chats": `
            SELECT DATE(assign_date) AS report_date, COUNT(*) AS total_chats
            FROM chat_report
            WHERE type = 'in'
              AND DATE(assign_date) BETWEEN ? AND ?
            GROUP BY report_date
            ORDER BY report_date;`,
		"frt": `
            SELECT DATE(c.assign_date) AS report_date, ROUND(AVG(c.chat_frt), 2) AS avg_chat_frt
            FROM chat_report c
            WHERE DATE(c.assign_date) BETWEEN ? AND ?
            GROUP BY report_date
            ORDER BY report_date;`,
		"sl": `
            SELECT DATE(c.enter_queue_date) AS report_date,
                   ROUND(SUM(CASE WHEN c.queue_wait_time <= 20 THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) AS sl
            FROM call_report c
            WHERE DATE(c.enter_queue_date) BETWEEN ? AND ?
              AND c.type = 'in'
              AND c.queue_name = 'm10'
            GROUP BY report_date
            ORDER BY report_date;`,
		"rt": `
            SELECT DATE(c.assign_date) AS report_date, ROUND(AVG(c.resolution_time_avg), 2) AS resolution_time_avg
            FROM chat_report c
            WHERE DATE(c.assign_date) BETWEEN ? AND ?
            GROUP BY report_date
            ORDER BY report_date;`,
		"abandoned": `
            SELECT DATE(c.enter_queue_date) AS report_date, COUNT(*) AS total_abandoned
            FROM call_report c
            WHERE DATE(c.enter_queue_date) BETWEEN ? AND ?
              AND c.type = 'abandon'
            GROUP BY report_date
            ORDER BY report_date;`,
		"topics_classif": `
            (SELECT 
                DATE(cr.answer_date) AS report_date,
                COALESCE(rt.category_name, '') AS category_or_name,
                'call' AS source,
                COUNT(*) AS total
            FROM call_report cr
            LEFT JOIN registered_topic rt ON cr.id = rt.call_report_id
            WHERE DATE(cr.answer_date) BETWEEN ? AND ?
            GROUP BY report_date, category_or_name)
            
            UNION ALL
            
            (SELECT 
                DATE(cr.first_agent_message_date) AS report_date,
                COALESCE(crt.name, '') AS category_or_name,
                'chat' AS source,
                COUNT(*) AS total
            FROM chat_report cr
            LEFT JOIN chat_registered_topic crt ON cr.id = crt.chat_report_id
            WHERE DATE(cr.first_agent_message_date) BETWEEN ? AND ?
            GROUP BY report_date, category_or_name)
            ORDER BY report_date, category_or_name;`,
		"subtopics_classif": `
            (SELECT 
                DATE(cr.answer_date) AS report_date,
                COALESCE(rt.category_name, '') AS category_or_name,
                'call' AS source,
                COUNT(*) AS total
            FROM call_report cr
            LEFT JOIN registered_topic rt ON cr.id = rt.call_report_id
            WHERE DATE(cr.answer_date) BETWEEN ? AND ?
            GROUP BY report_date, category_or_name)
            
            UNION ALL
            
            (SELECT 
                DATE(cr.first_agent_message_date) AS report_date,
                COALESCE(crt.name, '') AS category_or_name,
                'chat' AS source,
                COUNT(*) AS total
            FROM chat_report cr
            LEFT JOIN chat_registered_topic crt ON cr.id = crt.chat_report_id
            WHERE DATE(cr.first_agent_message_date) BETWEEN ? AND ?
            GROUP BY report_date, category_or_name)
            ORDER BY report_date, category_or_name;`,
	}

	query, ok := queries[tab]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid tab name"})
		return
	}

	log.Printf("[DEBUG] Starting daily data query for tab=%s with SQL:\n%s", tab, query)

	var params []interface{}
	// Для topics_classif и subtopics_classif параметры передаются дважды
	if tab == "topics_classif" || tab == "subtopics_classif" {
		params = []interface{}{startDate, endDate, startDate, endDate}
	} else {
		params = []interface{}{startDate, endDate}
	}

	data, err := executeQuery(query, params...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("Daily data result for tab=%s: %+v", tab, data)
	log.Printf("[DEBUG] Completed query for tab=%s, returned %d rows", tab, len(data))

	writeJSON(w, http.StatusOK, data)
}

// Handler для hourly данных (/hourly/data/{tab})
func getHourlyDataHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tab := vars["tab"]

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Please provide start_date and end_date"})
		return
	}

	log.Printf("Debug getHourlyDataHandler: tab=%s, start_date=%s, end_date=%s", tab, startDate, endDate)

	queries := map[string]string{
		"calls": `
            SELECT HOUR(c.enter_queue_date) AS hour, COUNT(*) AS total_calls
            FROM call_report c
            WHERE DATE(c.enter_queue_date) BETWEEN ? AND ?
              AND (c.type = 'in' OR c.type = 'abandon')
            GROUP BY hour
            ORDER BY hour;`,
		"aht": `
            SELECT HOUR(answer_date) AS hour, ROUND(AVG(call_duration), 2) AS avg_call_duration
            FROM call_report
            WHERE DATE(answer_date) BETWEEN ? AND ?
            GROUP BY hour
            ORDER BY hour;`,
		"chats": `
            SELECT HOUR(assign_date) AS hour, COUNT(*) AS total_chats
            FROM chat_report
            WHERE type = 'in'
              AND DATE(assign_date) BETWEEN ? AND ?
            GROUP BY hour
            ORDER BY hour;`,
		"frt": `
            SELECT HOUR(assign_date) AS hour, ROUND(AVG(chat_frt), 2) AS avg_chat_frt
            FROM chat_report
            WHERE DATE(assign_date) BETWEEN ? AND ?
            GROUP BY hour
            ORDER BY hour;`,
		"sl": `
            SELECT HOUR(enter_queue_date) AS hour,
                   ROUND(SUM(CASE WHEN queue_wait_time <= 20 THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) AS sl
            FROM call_report
            WHERE DATE(enter_queue_date) BETWEEN ? AND ?
              AND type = 'in'
              AND queue_name = 'm10'
            GROUP BY hour
            ORDER BY hour;`,
		"rt": `
            SELECT HOUR(assign_date) AS hour, ROUND(AVG(resolution_time_avg), 2) AS resolution_time_avg
            FROM chat_report
            WHERE DATE(assign_date) BETWEEN ? AND ?
            GROUP BY hour
            ORDER BY hour;`,
		"abandoned": `
            SELECT HOUR(c.enter_queue_date) AS hour, COUNT(*) AS total_abandoned
            FROM call_report c
            WHERE DATE(c.enter_queue_date) BETWEEN ? AND ?
              AND c.type = 'abandon'
            GROUP BY hour
            ORDER BY hour;`,
	}

	query, ok := queries[tab]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid tab name"})
		return
	}

	data, err := executeQuery(query, startDate, endDate)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	log.Printf("Hourly data result for tab=%s: %+v", tab, data)

	writeJSON(w, http.StatusOK, data)
}

// Handler для hourly table данных (/hourly/table/data/{tab})
func getHourlyTableDataHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tab := vars["tab"]

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Please provide start_date and end_date"})
		return
	}

	queries := map[string]string{
		"calls": `
            SELECT DATE(c.enter_queue_date) AS report_date, HOUR(c.enter_queue_date) AS hour, COUNT(*) AS total_calls
            FROM call_report c
            WHERE DATE(c.enter_queue_date) BETWEEN ? AND ?
              AND (c.type = 'in' OR c.type = 'abandon')
            GROUP BY report_date, hour
            ORDER BY report_date, hour;`,
		"aht": `
            SELECT DATE(c.enter_queue_date) AS report_date, HOUR(c.enter_queue_date) AS hour, ROUND(AVG(c.call_duration), 2) AS avg_call_duration
            FROM call_report c
            WHERE DATE(c.enter_queue_date) BETWEEN ? AND ?
            GROUP BY report_date, hour
            ORDER BY report_date, hour;`,
		"chats": `
            SELECT DATE(assign_date) AS report_date, HOUR(assign_date) AS hour, COUNT(*) AS total_chats
            FROM chat_report
            WHERE type = 'in'
              AND DATE(assign_date) BETWEEN ? AND ?
            GROUP BY report_date, hour
            ORDER BY report_date, hour;`,
		"frt": `
            SELECT DATE(c.assign_date) AS report_date, HOUR(c.assign_date) AS hour, ROUND(AVG(c.chat_frt), 2) AS avg_chat_frt
            FROM chat_report c
            WHERE DATE(c.assign_date) BETWEEN ? AND ?
            GROUP BY report_date, hour
            ORDER BY report_date, hour;`,
		"sl": `
            SELECT DATE(c.enter_queue_date) AS report_date, HOUR(c.enter_queue_date) AS hour,
                   ROUND(SUM(CASE WHEN c.queue_wait_time <= 20 THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) AS sl
            FROM call_report c
            WHERE DATE(c.enter_queue_date) BETWEEN ? AND ?
              AND c.type = 'in'
              AND c.queue_name = 'm10'
            GROUP BY report_date, hour
            ORDER BY report_date, hour;`,
		"rt": `
            SELECT DATE(c.assign_date) AS report_date, HOUR(c.assign_date) AS hour, ROUND(AVG(c.resolution_time_avg), 2) AS resolution_time_avg
            FROM chat_report c
            WHERE DATE(c.assign_date) BETWEEN ? AND ?
            GROUP BY report_date, hour
            ORDER BY report_date, hour;`,
		"abandoned": `
            SELECT DATE(c.enter_queue_date) AS report_date, HOUR(c.enter_queue_date) AS hour, COUNT(*) AS total_abandoned
            FROM call_report c
            WHERE DATE(c.enter_queue_date) BETWEEN ? AND ?
              AND c.type = 'abandon'
            GROUP BY report_date, hour
            ORDER BY report_date, hour;`,
	}

	query, ok := queries[tab]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid tab name"})
		return
	}

	data, err := executeQuery(query, startDate, endDate)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// Handler для онлайн данных (/online/data)
func onlineDataHandler(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	type result struct {
		total int
		err   error
	}

	var totalCalls, totalAbandoned, waitingCalls, totalChats, activeChats int
	var avgAHT, slValue, avgFRT, avgRT float64

	// Функция для запроса, использующая QueryRow
	queryInt := func(query string, param interface{}) (int, error) {
		var res sql.NullInt64
		err := db.QueryRow(query, param).Scan(&res)
		if err != nil {
			return 0, err
		}
		if res.Valid {
			return int(res.Int64), nil
		}
		return 0, nil
	}

	queryFloat := func(query string, param interface{}) (float64, error) {
		var res sql.NullFloat64
		err := db.QueryRow(query, param).Scan(&res)
		if err != nil {
			return 0, err
		}
		if res.Valid {
			return res.Float64, nil
		}
		return 0, nil
	}

	var err error
	totalCalls, err = queryInt(`
        SELECT COUNT(*) AS total_calls
        FROM call_report
        WHERE (type = 'in' OR type='abandon')
          AND enter_queue_date >= ?`, startOfDay)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	totalAbandoned, err = queryInt(`
        SELECT COUNT(*) AS total_abandoned
        FROM call_report
        WHERE type='abandon'
          AND enter_queue_date >= ?`, startOfDay)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	waitingCalls, err = queryInt(`
        SELECT COUNT(*) AS waiting_calls
        FROM call_report
        WHERE type='in'
          AND enter_queue_date >= ?
          AND answer_date IS NULL
          AND queue_wait_time IS NULL`, startOfDay)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	avgAHT, err = queryFloat(`
        SELECT ROUND(AVG(call_duration), 2) AS avg_call_duration
        FROM call_report
        WHERE enter_queue_date >= ?
          AND type='in'`, startOfDay)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	slValue, err = queryFloat(`
        SELECT ROUND(SUM(CASE WHEN queue_wait_time <= 20 THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) AS sl
        FROM call_report
        WHERE enter_queue_date >= ?
          AND type='in'
          AND queue_name='m10'`, startOfDay)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	totalChats, err = queryInt(`
        SELECT COUNT(*) AS total_chats
        FROM chat_report
        WHERE type='in'
          AND assign_date >= ?`, startOfDay)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	activeChats, err = queryInt(`
        SELECT COUNT(*) AS active_chats
        FROM summary_request
        WHERE type='CHAT'
          AND status='in'
          AND created_date >= ?
          AND queue_duration_time = ''`, startOfDay)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	avgFRT, err = queryFloat(`
        SELECT ROUND(AVG(chat_frt), 2) AS avg_chat_frt
        FROM chat_report
        WHERE assign_date >= ?`, startOfDay)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	avgRT, err = queryFloat(`
        SELECT ROUND(AVG(resolution_time_avg), 2) AS resolution_time_avg
        FROM chat_report
        WHERE assign_date >= ?`, startOfDay)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	resp := map[string]interface{}{
		"calls":         totalCalls,
		"abandoned":     totalAbandoned,
		"waiting_calls": waitingCalls,
		"aht":           avgAHT,
		"sl":            slValue,
		"chats":         totalChats,
		"active_chats":  activeChats,
		"frt":           avgFRT,
		"rt":            avgRT,
	}
	writeJSON(w, http.StatusOK, resp)
}

// Handler для custom query (/custom_query/data)
func customQueryDataHandler(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	tab := r.URL.Query().Get("tab")
	if tab == "" {
		tab = "overall_classif"
	}
	if startDate == "" || endDate == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Please provide start_date and end_date"})
		return
	}

	var query string
	var params []interface{}
	if tab == "call_classif" {
		query = `
            SELECT 
                COALESCE(rt.category_name, '') AS category_or_name,
                DATE(cr.answer_date) AS report_date,
                cr.number AS Number,
                'call' AS source,
                COUNT(*) AS total
            FROM call_report cr
            LEFT JOIN registered_topic rt ON cr.id = rt.call_report_id
            WHERE cr.answer_date >= ? AND cr.answer_date < ?
            GROUP BY category_or_name, report_date, Number`
		params = []interface{}{startDate, endDate}
	} else if tab == "chat_classif" {
		query = `
            SELECT 
                COALESCE(crt.name, '') AS category_or_name,
                DATE(cr.first_agent_message_date) AS report_date,
                cr.channel_id AS Number,
                'chat' AS source,
                COUNT(*) AS total
            FROM chat_report cr
            LEFT JOIN chat_registered_topic crt ON cr.id = crt.chat_report_id
            WHERE cr.first_agent_message_date >= ? AND cr.first_agent_message_date < ?
            GROUP BY category_or_name, report_date, Number`
		params = []interface{}{startDate, endDate}
	} else { // overall_classif или default
		query = `
            SELECT 
                COALESCE(rt.category_name, '') AS category_or_name,
                DATE(cr.answer_date) AS report_date,
                cr.number AS Number,
                'call' AS source,
                COUNT(*) AS total
            FROM call_report cr
            LEFT JOIN registered_topic rt ON cr.id = rt.call_report_id
            WHERE cr.answer_date >= ? AND cr.answer_date < ?
            GROUP BY category_or_name, report_date, Number

            UNION ALL

            SELECT 
                COALESCE(crt.name, '') AS category_or_name,
                DATE(cr.first_agent_message_date) AS report_date,
                cr.channel_id AS Number,
                'chat' AS source,
                COUNT(*) AS total
            FROM chat_report cr
            LEFT JOIN chat_registered_topic crt ON cr.id = crt.chat_report_id
            WHERE cr.first_agent_message_date >= ? AND cr.first_agent_message_date < ?
            GROUP BY category_or_name, report_date, Number`
		params = []interface{}{startDate, endDate, startDate, endDate}
	}

	data, err := executeQuery(query, params...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func detailedDailyDataHandler(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	if startDate == "" || endDate == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Please provide start_date and end_date"})
		return
	}

	// dataMap: map[date]string -> map[hour]int -> метрики
	dataMap := make(map[string]map[int]map[string]interface{})

	// Функция, которая гарантирует, что для заданной даты и часа в dataMap созданы все необходимые поля
	ensureDateHour := func(dateStr string, hour int) {
		if _, ok := dataMap[dateStr]; !ok {
			dataMap[dateStr] = make(map[int]map[string]interface{})
		}
		if _, ok := dataMap[dateStr][hour]; !ok {
			dataMap[dateStr][hour] = map[string]interface{}{
				"calls":     0,
				"aht":       0,
				"sl":        0,
				"abandoned": 0,
				"chats":     0,
				"frt":       0,
				"rt":        0,
				"agents":    0,
			}
		}
	}

	// Список обычных запросов с 2 плейсхолдерами
	normalQueries := []struct {
		query   string
		colName string
		conv    func(interface{}) interface{}
	}{
		{
			// Запрос для звонков (calls)
			query: `
                SELECT DATE(enter_queue_date) AS report_date,
                       HOUR(enter_queue_date) AS hour,
                       COUNT(*) AS total_calls
                FROM call_report
                WHERE DATE(enter_queue_date) BETWEEN ? AND ?
                  AND (type = 'in' OR type = 'abandon')
                GROUP BY report_date, hour
                ORDER BY report_date, hour;`,
			colName: "calls",
			conv:    func(v interface{}) interface{} { return toFloat64(v) },
		},
		{
			// Запрос для среднего времени разговора (aht)
			query: `
                SELECT DATE(enter_queue_date) AS report_date,
                       HOUR(enter_queue_date) AS hour,
                       ROUND(AVG(call_duration), 2) AS avg_call_duration
                FROM call_report
                WHERE DATE(enter_queue_date) BETWEEN ? AND ?
                  AND type = 'in'
                GROUP BY report_date, hour
                ORDER BY report_date, hour;`,
			colName: "aht",
			conv:    func(v interface{}) interface{} { return toFloat64(v) },
		},
		{
			// Запрос для SL
			query: `
                SELECT DATE(enter_queue_date) AS report_date,
                       HOUR(enter_queue_date) AS hour,
                       ROUND(SUM(CASE WHEN queue_wait_time <= 20 THEN 1 ELSE 0 END) / COUNT(*) * 100, 2) AS sl
                FROM call_report
                WHERE DATE(enter_queue_date) BETWEEN ? AND ?
                  AND type = 'in'
                  AND queue_name = 'm10'
                GROUP BY report_date, hour
                ORDER BY report_date, hour;`,
			colName: "sl",
			conv:    func(v interface{}) interface{} { return toFloat64(v) },
		},
		{
			// Запрос для Abandoned
			query: `
                SELECT DATE(enter_queue_date) AS report_date,
                       HOUR(enter_queue_date) AS hour,
                       COUNT(*) AS total_abandoned
                FROM call_report
                WHERE DATE(enter_queue_date) BETWEEN ? AND ?
                  AND type = 'abandon'
                GROUP BY report_date, hour
                ORDER BY report_date, hour;`,
			colName: "abandoned",
			conv:    func(v interface{}) interface{} { return toFloat64(v) },
		},
		{
			// Запрос для чатов (chats)
			query: `
                SELECT DATE(assign_date) AS report_date,
                       HOUR(assign_date) AS hour,
                       COUNT(*) AS total_chats
                FROM chat_report
                WHERE type = 'in'
                  AND DATE(assign_date) BETWEEN ? AND ?
                GROUP BY report_date, hour
                ORDER BY report_date, hour;`,
			colName: "chats",
			conv:    func(v interface{}) interface{} { return toFloat64(v) },
		},
		{
			// Запрос для FRT
			query: `
                SELECT DATE(assign_date) AS report_date,
                       HOUR(assign_date) AS hour,
                       AVG(chat_frt) AS avg_chat_frt
                FROM chat_report
                WHERE DATE(assign_date) BETWEEN ? AND ?
                GROUP BY report_date, hour
                ORDER BY report_date, hour;`,
			colName: "frt",
			conv:    func(v interface{}) interface{} { return toFloat64(v) },
		},
		{
			// Запрос для RT
			query: `
                SELECT DATE(assign_date) AS report_date,
                       HOUR(assign_date) AS hour,
                       AVG(resolution_time_avg) AS resolution_time_avg
                FROM chat_report
                WHERE DATE(assign_date) BETWEEN ? AND ?
                GROUP BY report_date, hour
                ORDER BY report_date, hour;`,
			colName: "rt",
			conv:    func(v interface{}) interface{} { return toFloat64(v) },
		},
	}

	// Выполняем обычные запросы (с 2 плейсхолдерами)
	for _, nq := range normalQueries {
		if err := execAndFill2(nq.query, nq.colName, nq.conv, startDate, endDate, dataMap, ensureDateHour); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	// UNION‑запрос для агентов (agents) – с 4 плейсхолдерами:
	unionQuery := `
		SELECT DATE(created_at) AS report_date,
			   HOUR(created_at) AS hour,
			   COUNT(DISTINCT user_id) AS distinct_agents
		FROM call_report
		WHERE created_at >= ? AND created_at < ?
		  AND type = 'in'
		GROUP BY report_date, hour
		UNION ALL
		SELECT DATE(assign_date) AS report_date,
			   HOUR(assign_date) AS hour,
			   COUNT(DISTINCT user_id) AS distinct_agents
		FROM chat_report
		WHERE assign_date >= ? AND assign_date < ?
		  AND type = 'in'
		GROUP BY report_date, hour
	`

	// Create a wrapper function that returns interface{} instead of float64
	toFloat64Interface := func(val interface{}) interface{} {
		return toFloat64(val)
	}

	if err := execAndFill4(unionQuery, "agents", toFloat64Interface, startDate, endDate, dataMap, ensureDateHour); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, dataMap)
}
func execAndFill2(query, colName string, conv func(interface{}) interface{},
	startDate, endDate string,
	dataMap map[string]map[int]map[string]interface{},
	ensureDateHour func(string, int)) error {

	rows, err := db.Query(query, startDate, endDate)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var reportDate time.Time
		var hour int
		var val interface{}
		if err := rows.Scan(&reportDate, &hour, &val); err != nil {
			return err
		}
		dateStr := reportDate.Format("2006-01-02")
		ensureDateHour(dateStr, hour)
		dataMap[dateStr][hour][colName] = conv(val)
	}
	return nil
}

func execAndFill4(query, colName string, conv func(interface{}) interface{},
	startDate, endDate string,
	dataMap map[string]map[int]map[string]interface{},
	ensureDateHour func(string, int)) error {

	// Передаём 4 параметра: по два для каждой части UNION
	rows, err := db.Query(query, startDate, endDate, startDate, endDate)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var reportDate time.Time
		var hour int
		var val interface{}
		if err := rows.Scan(&reportDate, &hour, &val); err != nil {
			return err
		}
		dateStr := reportDate.Format("2006-01-02")
		ensureDateHour(dateStr, hour)
		dataMap[dateStr][hour][colName] = conv(val)
	}
	return nil
}

// Handler для прогноза по часу (/forecast/data)
func forecastDataHandler(w http.ResponseWriter, r *http.Request) {
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "optimal"
	}
	if startDateStr == "" || endDateStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Please provide start_date and end_date"})
		return
	}

	historicalEnd, err := parseDate(startDateStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid start_date"})
		return
	}
	historicalStart := historicalEnd.AddDate(0, 0, -30)

	// Запрос исторических данных для звонков
	callQuery := `
        SELECT 
            DATE(enter_queue_date) as report_date,
            HOUR(enter_queue_date) as hour,
            COUNT(CASE WHEN type = 'in' OR type = 'abandon' THEN 1 END) as total_calls,
            COUNT(CASE WHEN type = 'abandon' THEN 1 END) as total_abandoned,
            AVG(call_duration) as aht,
            ROUND(SUM(CASE WHEN queue_wait_time <= 20 AND type = 'in' THEN 1 ELSE 0 END) / 
                  NULLIF(SUM(CASE WHEN type = 'in' THEN 1 END), 0) * 100, 2) as sl
        FROM call_report
        WHERE enter_queue_date BETWEEN ? AND ?
        GROUP BY report_date, hour`
	callData, err := executeQuery(callQuery, historicalStart.Format("2006-01-02"), historicalEnd.Format("2006-01-02"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Запрос исторических данных для чатов
	chatQuery := `
        SELECT 
            DATE(assign_date) as report_date,
            HOUR(assign_date) as hour,
            COUNT(*) as total_chats,
            AVG(chat_frt) as frt,
            AVG(resolution_time_avg) as rt
        FROM chat_report
        WHERE assign_date BETWEEN ? AND ?
        GROUP BY report_date, hour`
	chatData, err := executeQuery(chatQuery, historicalStart.Format("2006-01-02"), historicalEnd.Format("2006-01-02"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Функции для безопасного получения чисел
	safeFloat := func(val interface{}) float64 {
		return toFloat64(val)
	}

	forecast := make(map[string]map[int]map[string]int)
	startDate, _ := parseDate(startDateStr)
	endDate, _ := parseDate(endDateStr)
	currentDate := startDate

	// Для каждого часа (0-23) вычисляем прогноз на основе исторических данных (фильтруем по hour)
	for !currentDate.After(endDate) {
		dateStr := currentDate.Format("2006-01-02")
		forecast[dateStr] = make(map[int]map[string]int)
		for hour := 0; hour < 24; hour++ {
			var hourCalls, hourAbandoned, hourChats, hourAHT, hourSL, hourFRT, hourRT float64

			// Собираем данные по каждому часу
			var callsSlice, abandonedSlice, chatsSlice, ahtSlice, slSlice, frtSlice, rtSlice []float64
			for _, row := range callData {
				if int(toFloat64(row["hour"])) == hour {
					callsSlice = append(callsSlice, safeFloat(row["total_calls"]))
					abandonedSlice = append(abandonedSlice, safeFloat(row["total_abandoned"]))
					ahtSlice = append(ahtSlice, safeFloat(row["aht"]))
					slSlice = append(slSlice, safeFloat(row["sl"]))
				}
			}
			for _, row := range chatData {
				if int(toFloat64(row["hour"])) == hour {
					chatsSlice = append(chatsSlice, safeFloat(row["total_chats"]))
					frtSlice = append(frtSlice, safeFloat(row["frt"]))
					rtSlice = append(rtSlice, safeFloat(row["rt"]))
				}
			}

			switch mode {
			case "optimal":
				hourCalls = percentile(callsSlice, 75)
				hourAbandoned = percentile(abandonedSlice, 25)
				hourChats = percentile(chatsSlice, 75)
				hourAHT = percentile(ahtSlice, 25)
				hourSL = percentile(slSlice, 75)
				hourFRT = percentile(frtSlice, 25)
				hourRT = percentile(rtSlice, 25)
			case "aggressive":
				if len(callsSlice) > 0 {
					hourCalls = callsSlice[len(callsSlice)-1]
				}
				if len(abandonedSlice) > 0 {
					hourAbandoned = min(abandonedSlice)
				}
				if len(chatsSlice) > 0 {
					hourChats = chatsSlice[len(chatsSlice)-1]
				}
				if len(ahtSlice) > 0 {
					hourAHT = min(ahtSlice)
				}
				hourSL = 98
				if len(frtSlice) > 0 {
					hourFRT = min(frtSlice)
				}
				if len(rtSlice) > 0 {
					hourRT = min(rtSlice)
				}
			default: // conservative
				hourCalls = mean(callsSlice)
				hourAbandoned = mean(abandonedSlice)
				hourChats = mean(chatsSlice)
				hourAHT = mean(ahtSlice)
				hourSL = mean(slSlice)
				hourFRT = mean(frtSlice)
				hourRT = mean(rtSlice)
			}

			// Округляем значения до целых
			callsInt := int(math.Round(hourCalls))
			abandonedInt := int(math.Round(hourAbandoned))
			chatsInt := int(math.Round(hourChats))
			ahtInt := int(math.Round(hourAHT))
			slInt := int(math.Round(hourSL))
			frtInt := int(math.Round(hourFRT))
			rtInt := int(math.Round(hourRT))
			totalContacts := callsInt + chatsInt
			requiredAgents := int(math.Max(1, math.Round(float64(totalContacts)/15.0)))

			forecast[dateStr][hour] = map[string]int{
				"calls":           callsInt,
				"abandoned":       abandonedInt,
				"chats":           chatsInt,
				"aht":             ahtInt,
				"sl":              slInt,
				"frt":             frtInt,
				"rt":              rtInt,
				"required_agents": requiredAgents,
			}
		}
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	// Для исторической статистики
	var callsForHist []float64
	var chatsForHist []float64
	var ahtForHist []float64
	var slForHist []float64

	for _, row := range callData {
		callsForHist = append(callsForHist, safeFloat(row["total_calls"]))
		ahtForHist = append(ahtForHist, safeFloat(row["aht"]))
		slForHist = append(slForHist, safeFloat(row["sl"]))
	}
	for _, row := range chatData {
		chatsForHist = append(chatsForHist, safeFloat(row["total_chats"]))
	}

	historicalStats := map[string]interface{}{
		"calls": map[string]float64{
			"avg": mean(callsForHist),
		},
		"chats": map[string]float64{
			"avg": mean(chatsForHist),
		},
		"aht": map[string]float64{
			"avg": mean(ahtForHist),
			"min": min(ahtForHist),
		},
		"sl": map[string]float64{
			"avg": mean(slForHist),
		},
	}

	resp := map[string]interface{}{
		"forecast":         forecast,
		"historical_stats": historicalStats,
	}
	writeJSON(w, http.StatusOK, resp)
}

// Handler для ежедневного прогноза (/forecast/daily/data)
func forecastDailyDataHandler(w http.ResponseWriter, r *http.Request) {
	startDateStr := r.URL.Query().Get("start_date")
	endDateStr := r.URL.Query().Get("end_date")
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "optimal"
	}
	if startDateStr == "" || endDateStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Please provide start_date and end_date"})
		return
	}

	historicalEnd, err := parseDate(startDateStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid start_date"})
		return
	}
	historicalStart := historicalEnd.AddDate(0, 0, -30)

	// Запрос исторических данных для звонков по дням
	callQuery := `
        SELECT 
            DATE(enter_queue_date) as report_date,
            COUNT(CASE WHEN type = 'in' OR type = 'abandon' THEN 1 END) as total_calls,
            COUNT(CASE WHEN type = 'abandon' THEN 1 END) as total_abandoned,
            AVG(call_duration) as aht,
            ROUND(SUM(CASE WHEN queue_wait_time <= 20 AND type = 'in' THEN 1 ELSE 0 END) / 
                  NULLIF(SUM(CASE WHEN type = 'in' THEN 1 END), 0) * 100, 2) as sl
        FROM call_report
        WHERE enter_queue_date BETWEEN ? AND ?
        GROUP BY report_date`
	callData, err := executeQuery(callQuery, historicalStart.Format("2006-01-02"), historicalEnd.Format("2006-01-02"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Запрос исторических данных для чатов по дням
	chatQuery := `
        SELECT 
            DATE(assign_date) as report_date,
            COUNT(*) as total_chats,
            AVG(chat_frt) as frt,
            AVG(resolution_time_avg) as rt
        FROM chat_report
        WHERE assign_date BETWEEN ? AND ?
        GROUP BY report_date`
	chatData, err := executeQuery(chatQuery, historicalStart.Format("2006-01-02"), historicalEnd.Format("2006-01-02"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Преобразуем результаты в карты по датам
	callsByDay := make(map[string]float64)
	abandonedByDay := make(map[string]float64)
	ahtByDay := make(map[string]float64)
	slByDay := make(map[string]float64)
	for _, row := range callData {
		dateStr := fmt.Sprintf("%v", row["report_date"])
		callsByDay[dateStr] = toFloat64(row["total_calls"])
		abandonedByDay[dateStr] = toFloat64(row["total_abandoned"])
		ahtByDay[dateStr] = toFloat64(row["aht"])
		slByDay[dateStr] = toFloat64(row["sl"])
	}

	chatsByDay := make(map[string]float64)
	frtByDay := make(map[string]float64)
	rtByDay := make(map[string]float64)
	for _, row := range chatData {
		dateStr := fmt.Sprintf("%v", row["report_date"])
		chatsByDay[dateStr] = toFloat64(row["total_chats"])
		frtByDay[dateStr] = toFloat64(row["frt"])
		rtByDay[dateStr] = toFloat64(row["rt"])
	}

	// Собираем срезы для вычислений
	var callsDaily, abandonedDaily, chatsDaily, ahtDaily, slDaily, frtDaily, rtDaily []float64
	for _, v := range callsByDay {
		callsDaily = append(callsDaily, v)
	}
	for _, v := range abandonedByDay {
		abandonedDaily = append(abandonedDaily, v)
	}
	for _, v := range chatsByDay {
		chatsDaily = append(chatsDaily, v)
	}
	for _, v := range ahtByDay {
		ahtDaily = append(ahtDaily, v)
	}
	for _, v := range slByDay {
		slDaily = append(slDaily, v)
	}
	for _, v := range frtByDay {
		frtDaily = append(frtDaily, v)
	}
	for _, v := range rtByDay {
		rtDaily = append(rtDaily, v)
	}

	forecast := make(map[string]map[string]int)
	startDate, _ := parseDate(startDateStr)
	endDate, _ := parseDate(endDateStr)
	currentDate := startDate

	for !currentDate.After(endDate) {
		dateStr := currentDate.Format("2006-01-02")
		var fcalls, fabandoned, fchats, faht, fsl, ffrt, frtVal float64
		switch mode {
		case "optimal":
			fcalls = percentile(callsDaily, 75)
			fabandoned = percentile(abandonedDaily, 25)
			fchats = percentile(chatsDaily, 75)
			faht = percentile(ahtDaily, 25)
			fsl = percentile(slDaily, 75)
			ffrt = percentile(frtDaily, 25)
			frtVal = percentile(rtDaily, 25)
		case "aggressive":
			if len(callsDaily) > 0 {
				fcalls = callsDaily[len(callsDaily)-1]
			}
			if len(abandonedDaily) > 0 {
				fabandoned = min(abandonedDaily)
			}
			if len(chatsDaily) > 0 {
				fchats = chatsDaily[len(chatsDaily)-1]
			}
			if len(ahtDaily) > 0 {
				faht = min(ahtDaily)
			}
			fsl = 98
			if len(frtDaily) > 0 {
				ffrt = min(frtDaily)
			}
			if len(rtDaily) > 0 {
				frtVal = min(rtDaily)
			}
		default: // conservative
			fcalls = mean(callsDaily)
			fabandoned = mean(abandonedDaily)
			fchats = mean(chatsDaily)
			faht = mean(ahtDaily)
			fsl = mean(slDaily)
			ffrt = mean(frtDaily)
			frtVal = mean(rtDaily)
		}

		// Приводим к целым
		callsInt := int(math.Round(fcalls))
		abandonedInt := int(math.Round(fabandoned))
		chatsInt := int(math.Round(fchats))
		ahtInt := int(math.Round(faht))
		slint := int(math.Round(fsl))
		frtInt := int(math.Round(ffrt))
		rtInt := int(math.Round(frtVal))

		// Применяем поправки по дням недели
		weekday := currentDate.Weekday() // 0=Sunday, 1=Monday,...
		if weekday == time.Monday {
			callsInt = int(float64(callsInt) * 1.15)
			chatsInt = int(float64(chatsInt) * 1.1)
		} else if weekday == time.Friday {
			callsInt = int(float64(callsInt) * 0.95)
		} else if weekday == time.Saturday || weekday == time.Sunday {
			callsInt = int(float64(callsInt) * 0.6)
			chatsInt = int(float64(chatsInt) * 0.5)
			if slint+3 < 99 {
				slint = slint + 3
			} else {
				slint = 99
			}
		}
		totalContacts := callsInt + chatsInt
		requiredAgents := int(math.Max(1, math.Round(float64(totalContacts)/120.0)))

		forecast[dateStr] = map[string]int{
			"calls":           callsInt,
			"abandoned":       abandonedInt,
			"chats":           chatsInt,
			"aht":             ahtInt,
			"sl":              slint,
			"frt":             frtInt,
			"rt":              rtInt,
			"required_agents": requiredAgents,
		}
		currentDate = currentDate.AddDate(0, 0, 1)
	}

	historicalStats := map[string]interface{}{
		"calls": map[string]float64{
			"avg": mean(callsDaily),
		},
		"chats": map[string]float64{
			"avg": mean(chatsDaily),
		},
		"aht": map[string]float64{
			"avg": mean(ahtDaily),
			"min": min(ahtDaily),
		},
		"sl": map[string]float64{
			"avg": mean(slDaily),
		},
	}

	resp := map[string]interface{}{
		"forecast":         forecast,
		"historical_stats": historicalStats,
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---------------------- MAIN ----------------------

func main() {
	// Инициализируем БД
	if err := initDB(); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	// Создаем роутер
	r := mux.NewRouter()

	// Маршруты для шаблонов и API
	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/daily/data/{tab}", getDailyDataHandler).Methods("GET")
	r.HandleFunc("/hourly/data/{tab}", getHourlyDataHandler).Methods("GET")
	r.HandleFunc("/hourly/table/data/{tab}", getHourlyTableDataHandler).Methods("GET")
	r.HandleFunc("/online/data", onlineDataHandler).Methods("GET")
	r.HandleFunc("/custom_query/data", customQueryDataHandler).Methods("GET")
	r.HandleFunc("/detailed/daily/data", detailedDailyDataHandler).Methods("GET")
	r.HandleFunc("/forecast/data", forecastDataHandler).Methods("GET")
	r.HandleFunc("/forecast/daily/data", forecastDailyDataHandler).Methods("GET")

	// Статические файлы для JavaScript
	jsDir := http.Dir(filepath.Join("flask-dashboard", "app", "static", "js"))
	r.PathPrefix("/js/").Handler(http.StripPrefix("/js/", http.FileServer(jsDir)))

	// Запуск сервера
	addr := ":5000"
	fmt.Println("Server is running on", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
