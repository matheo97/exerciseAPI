package main

import (
	"database/sql"
	"log"
	"net/http"

	create "./exercise"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

func updateExercise(w http.ResponseWriter, r *http.Request) {

}
func getRanking(w http.ResponseWriter, r *http.Request) {

}
func main() {

	database, _ := sql.Open("sqlite3", "../egym.db")
	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS exercises (ID INTEGER PRIMARY KEY AUTOINCREMENT, USER_ID INTEGER NOT NULL, DESCRIPTION TEXT NOT NULL, TYPE TEXT NOT NULL, START_TIME TEXT NOT NULL, FINISH_TIME TEXT NOT NULL, DURATION INTEGER NOT NULL, CALORIES INTEGER NOT NULL, POINTS INTEGER NOT NULL)")
	statement.Exec()

	r := mux.NewRouter()

	r.HandleFunc("/exercise", create.ExerciseEndpoint).Methods("POST")
	r.HandleFunc("/exercise/{exerciseId}", updateExercise).Methods("PUT")
	r.HandleFunc("/ranking", getRanking).Methods("GET")

	log.Fatal(http.ListenAndServe(":8080", r))
}
