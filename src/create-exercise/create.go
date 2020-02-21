package create

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"
)

// ExerciseType Type of the Exercise
type ExerciseType string

var (
	// ErrMissingUserID Error when userId field is not received
	ErrMissingUserID = errors.New("Missing userId")
	// ErrMissingDescription Error when description field is not received
	ErrMissingDescription = errors.New("Missing description")
	// ErrInvalidDescription Error when description field is not received
	ErrInvalidDescription = errors.New("Invalid description not an alphanumeric string")
	// ErrMissingType Error when type field is not received
	ErrMissingType = errors.New("Missing type")
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
	// ErrExerciseOverlapping Error when a new exercise overlaps a saved one
	ErrExerciseOverlapping = errors.New("The exercise that you intended to create overlaps with an existing one")

	validTypes = map[ExerciseType]bool{
		RunningType:          true,
		SwimmingType:         true,
		StrenghtTrainingType: true,
		CircuitTrainingType:  true,
	}
)

const (
	// RunningType Exercise type for running
	RunningType ExerciseType = "RUNNING"
	// SwimmingType Exercise type for swimming
	SwimmingType ExerciseType = "SWIMMING"
	// StrenghtTrainingType Exercise type for strength training
	StrenghtTrainingType ExerciseType = "STRENGTH_TRAINING"
	// CircuitTrainingType Exercise type for circuit training
	CircuitTrainingType ExerciseType = "CIRCUIT_TRAINING"
)

// Exercise structure and Request structure
type Exercise struct {
	// ID field of Exercise
	ID int64 `json:"id"`
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
	Exercise *Exercise `json:"exercise,omitempty"`
	Error    string    `json:"error,omitempty"`
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

	dir, err := os.Getwd()
	if err != nil {
		return false, err
	}
	database, err := sql.Open("sqlite3", fmt.Sprintf("%s/egym.db", dir))
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

func (e *Exercise) validateCreateExerciseRequest() error {
	if e.UserID == 0 {
		return ErrMissingUserID
	}

	if e.Description == "" {
		return ErrMissingDescription
	}

	if !isAlphaNumericString(e.Description) {
		return ErrInvalidDescription
	}

	if e.ExerciseType == "" {
		return ErrMissingType
	}

	if !validTypes[e.ExerciseType] {
		return ErrInvalidType
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

func (e *Exercise) createExercise() error {
	finishDate := addDurationToDate(e.StartTime, e.Duration) // esto podria estar siendo redundante
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	database, err := sql.Open("sqlite3", fmt.Sprintf("%s/egym.db", dir))
	if err != nil {
		return err
	}

	statement, err := database.Prepare("INSERT INTO exercises (USER_ID, DESCRIPTION, TYPE, START_TIME, FINISH_TIME, DURATION, CALORIES) VALUES (?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}

	result, err := statement.Exec(e.UserID, e.Description, e.ExerciseType, e.StartTime, finishDate, e.Duration, e.Calories)
	if err != nil {
		return err
	}

	e.ID, err = result.LastInsertId()

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
	newResponse := &Response{}
	exercise := &Exercise{}

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(exercise); err != nil {
		response(w, http.StatusBadRequest, newResponse, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err := exercise.validateCreateExerciseRequest()
	if err != nil {
		response(w, http.StatusBadRequest, newResponse, err)
		return
	}

	err = exercise.createExercise()
	if err != nil {
		response(w, http.StatusInternalServerError, newResponse, err)
		return
	}

	newResponse.Exercise = exercise
	response(w, http.StatusCreated, newResponse, err)
}
