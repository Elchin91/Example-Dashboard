package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Elchin91/GoDashboard/db"
	"github.com/gorilla/mux"
)

// Example handler function (adapt to your specific handlers)
func dailyDataHandler(w http.ResponseWriter, r *http.Request) {
	// Extract parameters from the request (e.g., start_date, end_date, tab)
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")
	tab := mux.Vars(r)["tab"]

	// Use the new functions from the db package to fetch the data
	data, err := db.GetDailyData(startDate, endDate, tab)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching data: %v", err), http.StatusInternalServerError)
		return
	}

	// Marshal the data to JSON and write it to the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// Add other handler functions here, updating them to use the db package
