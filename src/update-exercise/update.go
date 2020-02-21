package update

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// ExerciseType Type of the Exercise
type ExerciseType string

var (
	// ErrInvalidID Error when ID field is not valid
	ErrInvalidID = errors.New("Invalid exercise id")
	// ErrMissingID Error when ID field is not received
	ErrMissingID = errors.New("Missing exercise id")
	// ErrUnwantedUserID Error when userId field is received
	ErrUnwantedUserID = errors.New("Unwanted userId field received")
	// ErrMissingDescription Error when description field is not received
	ErrMissingDescription = errors.New("Missing description")
	// ErrInvalidDescription Error when description field is not received
	ErrInvalidDescription = errors.New("Invalid description not an alphanumeric string")
	// ErrUnwantedType Error when type field is received
	ErrUnwantedType = errors.New("Unwanted type field received")
	// ErrInvalidType Error when type field is invalid
	ErrInvalidType = errors.New("Invalid type")
	// ErrMissingStartTime Error when startTime field is not received
	ErrMissingStartTime = errors.New("Missing startTime")
	// ErrInvalidStartTime Error when startTime field is invalid
	ErrInvalidStartTime = errors.New("Invalid startTime format must be ISO8601")
	// ErrMissingDuration Error when duration field is not received
	ErrMissingDuration = errors.New("Missing duration")
	// ErrMissingCalories Error when calories field is not received
	ErrMissingCalories = errors.New("Missing calories")
	// ErrInvalidExercise Error when total points calculated for an exercise returns 0
	ErrInvalidExercise = errors.New("Invalid exercise as to total points calculation equals 0")
	// ErrExerciseOverlapping Error when a new exercise overlaps a saved one
	ErrExerciseOverlapping = errors.New("The exercise that you intended to create overlaps with an existing one")
	// ErrDatabaseError internal database error
	ErrDatabaseError = errors.New("Internal database error")
	// ErrNoExerciseFound The exercise you tried to update does not exists
	ErrNoExerciseFound = errors.New("The exercise you tried to update does not exists")
)

// Exercise structure and Request structure
type Exercise struct {
	// UserID id field of User
	UserID int64 `json:"userId"`
	// Description of the Exercise
	Description string `json:"description"`
	// ExerciseType type of the exercise
	ExerciseType ExerciseType `json:"type"`
	// StartTime time when exercise starts
	StartTime time.Time `json:"startTime"`
	// Duration duration of the exercise
	Duration int64 `json:"duration"`
	// Calories burnt on the exercise
	Calories int64 `json:"calories"`
}

// Response for /exercise
type Response struct {
	Exercise *Exercise `json:"exercise"`
	Error    string    `json:"error"`
}

func isAlphaNumericString(description string) bool {
	AlphaNumericStringRegex := `^[A-Za-z0-9\s]+$`
	AlphaNumericRegex := regexp.MustCompile(AlphaNumericStringRegex)

	return AlphaNumericRegex.MatchString(description)
}

func addDurationToDate(date time.Time, duration int64) time.Time {
	afterDurationSeconds := date.Add(time.Second * time.Duration(duration))
	return afterDurationSeconds
}

func checkExerciseOverlapping(userID int64, startDate time.Time, finishDate time.Time) (bool, error) {
	var totalExercisesCollatingOnStart int
	var totalExercisesCollatingOnFinish int

	database, err := sql.Open("sqlite3", "../egym.db")
	if err != nil {
		return true, err
	}

	sqlStatement := `SELECT COUNT(*) FROM exercises WHERE USER_ID=$1 AND START_TIME BETWEEN $2 AND $3;`
	_ = database.QueryRow(sqlStatement, userID, startDate, finishDate).Scan(&totalExercisesCollatingOnStart)

	sqlStatement = `SELECT COUNT(*) FROM exercises WHERE USER_ID=$1 AND FINISH_TIME BETWEEN $2 AND $3;`
	_ = database.QueryRow(sqlStatement, userID, startDate, finishDate).Scan(&totalExercisesCollatingOnFinish)

	if totalExercisesCollatingOnStart > 0 || totalExercisesCollatingOnFinish > 0 {
		return true, ErrExerciseOverlapping
	}

	return false, nil
}

func (e *Exercise) validateUpdateExerciseRequest(ID int64) error {
	if ID == 0 {
		return ErrMissingID
	}

	if e.UserID != 0 {
		return ErrUnwantedUserID
	}

	if e.Description == "" {
		return ErrMissingDescription
	}

	if !isAlphaNumericString(e.Description) {
		return ErrInvalidDescription
	}

	if e.ExerciseType != "" {
		return ErrUnwantedType
	}

	if e.StartTime.IsZero() {
		return ErrMissingStartTime
	}

	if e.Duration == 0 {
		return ErrMissingDuration
	}

	if e.Calories == 0 {
		return ErrMissingCalories
	}

	finishDate := addDurationToDate(e.StartTime, e.Duration)
	isOverlapping, err := checkExerciseOverlapping(e.UserID, e.StartTime, finishDate)
	if isOverlapping {
		return err
	}

	return nil
}

func (e *Exercise) updateExercise(ID int64) error {
	finishDate := addDurationToDate(e.StartTime, e.Duration)

	database, err := sql.Open("sqlite3", "../egym.db")
	if err != nil {
		return ErrDatabaseError
	}

	sqlStatement := `SELECT COUNT(*) exercises WHERE ID=$1;`
	var numberOfElements int64
	_ = database.QueryRow(sqlStatement, ID).Scan(&numberOfElements)
	if numberOfElements == 0 {
		return ErrNoExerciseFound
	}

	statement, err := database.Prepare("UPDATE exercises SET DESCRIPTION=$1, START_TIME=$2, FINISH_TIME=$3, DURATION=$4, CALORIES=$5 WHERE ID=$6")
	if err != nil {
		return ErrDatabaseError
	}

	_, err = statement.Exec(e.Description, e.StartTime, finishDate, e.Duration, e.Calories, ID)

	sqlStatement = `SELECT USER_ID, TYPE FROM exercises WHERE ID=$1;`
	var userID int64
	var exerciseType ExerciseType

	_ = database.QueryRow(sqlStatement, ID).Scan(&userID, &exerciseType)
	e.UserID = userID
	e.ExerciseType = exerciseType

	return err
}

func response(w http.ResponseWriter, httpStatus int, response *Response, err error) {
	if err != nil {
		response.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(response)
}

// ExerciseEndpoint function that handles request and response
func ExerciseEndpoint(w http.ResponseWriter, r *http.Request) {
	exercise := &Exercise{}
	newResponse := &Response{}
	params := mux.Vars(r)

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(exercise); err != nil {
		response(w, http.StatusBadRequest, newResponse, err)
		return
	}

	exerciseID, err := strconv.ParseInt(params["exerciseId"], 10, 64)
	if err != nil {
		response(w, http.StatusBadRequest, newResponse, err)
		return
	}

	err = exercise.validateUpdateExerciseRequest(exerciseID)
	if err != nil {
		response(w, http.StatusBadRequest, newResponse, err)
		return
	}

	err = exercise.updateExercise(exerciseID)
	if err != nil {
		response(w, http.StatusInternalServerError, newResponse, err)
		return
	}

	newResponse.Exercise = exercise
	response(w, http.StatusOK, newResponse, err)
}
