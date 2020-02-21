package main

import (
	"database/sql"
	"log"
	"net/http"

	create "./create-exercise"
	rank "./get-ranking"
	update "./update-exercise"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

func createTable() error {
	database, err := sql.Open("sqlite3", "../egym.db")
	if err != nil {
		return err
	}

	statement, err := database.Prepare("CREATE TABLE IF NOT EXISTS exercises (ID INTEGER PRIMARY KEY AUTOINCREMENT, USER_ID INTEGER NOT NULL, DESCRIPTION TEXT NOT NULL, TYPE TEXT NOT NULL, START_TIME DATE NOT NULL, FINISH_TIME DATE NOT NULL, DURATION INTEGER NOT NULL, CALORIES INTEGER NOT NULL)")
	if err != nil {
		return err
	}

	statement.Exec()

	return nil
}

func main() {
	err := createTable()
	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/exercise", create.ExerciseEndpoint).Methods("POST")
	r.HandleFunc("/exercise/{exerciseId}", update.ExerciseEndpoint).Methods("PUT")
	r.HandleFunc("/ranking", rank.RankingEndpoint).Methods("GET")

	log.Fatal(http.ListenAndServe(":8080", r))
}
