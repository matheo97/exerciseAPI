package rank

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// ExerciseType Type of the Exercise
type ExerciseType string

var (
	// ErrInvalidUserIDs Error when userIDs params is invalid
	ErrInvalidUserIDs = errors.New("Invalid params userIDs")

	exerciseTypes = map[ExerciseType]ExerciseType{
		RunningType:          RunningType,
		SwimmingType:         SwimmingType,
		StrenghtTrainingType: StrenghtTrainingType,
		CircuitTrainingType:  CircuitTrainingType,
	}

	getMultiplicationFactor = map[ExerciseType]int{
		RunningType:          2,
		SwimmingType:         3,
		StrenghtTrainingType: 3,
		CircuitTrainingType:  4,
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

// Row is a user struct
type Row struct {
	ExerciseType string
	Duration     int64
	Calories     int64
	FinishTime   time.Time
}

// PointsByType points of user by type
type PointsByType struct {
	UserID           string
	ExerciseType     ExerciseType
	Points           float64
	LastExerciseDate time.Time
}

// User is a user struct
type User struct {
	UserID           string
	Points           float64
	LastExerciseDate time.Time
}

// Response for /exercise
type Response struct {
	Ranking []*User `json:"ranking"` // use struct []*User inside []*PointsByType
	Error   string  `json:"error"`
}

// ByPoints implements sort.Interface based on the points field
type ByPoints []*User

func (p ByPoints) Len() int { return len(p) }
func (p ByPoints) Less(i, j int) bool {
	if p[i].Points == p[j].Points {
		diff := p[i].LastExerciseDate.Sub(p[j].LastExerciseDate)
		if diff > 0 {
			return true
		}
		return false
	}

	return p[i].Points > p[j].Points
}
func (p ByPoints) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func totalPointsByUser(userID string, pointsByUser []*PointsByType) (*User, error) {
	totalPointsByUser := &User{
		UserID: userID,
	}

	LastExerciseDone, err := time.Parse(time.RFC3339, "1990-10-30T12:34:23Z")
	if err != nil {
		return nil, err
	}

	for _, pointsPerType := range pointsByUser {
		totalPointsByUser.Points += pointsPerType.Points

		diff := LastExerciseDone.Sub(pointsPerType.LastExerciseDate)
		if diff < 0 {
			totalPointsByUser.LastExerciseDate = pointsPerType.LastExerciseDate
		}
	}

	return totalPointsByUser, nil
}

func calculatePointsByExerciseType(userID string, exerciseType ExerciseType, exercises []Row) *PointsByType {
	pointsByType := &PointsByType{
		UserID:       userID,
		ExerciseType: exerciseType,
	}

	if len(exercises) > 0 {
		pointsByType.LastExerciseDate = exercises[0].FinishTime
	}

	multiplicationFactor := getMultiplicationFactor[exerciseType]
	percent := 100.0

	for _, exercise := range exercises {
		basePoints := (int64((exercise.Duration+59)/60) + exercise.Calories) * int64(multiplicationFactor)

		if percent <= 0 {
			break
		}

		pointsByType.Points += float64(basePoints) * (percent / 100.0)
		percent -= 10.0
	}

	return pointsByType
}

func setResult(result *sql.Rows) ([]Row, error) {
	var userExercises []Row
	for result.Next() {
		var row Row
		var finishTime string

		if err := result.Scan(&row.ExerciseType, &row.Duration, &row.Calories, &finishTime); err != nil {
			return nil, err
		}

		time, err := time.Parse(time.RFC3339, finishTime)
		if err != nil {
			return nil, err
		}

		row.FinishTime = time

		userExercises = append(userExercises, row)
	}

	result.Close()

	return userExercises, nil
}

func getExercisesByType(exerciseType ExerciseType, userID string) ([]Row, error) {
	database, err := sql.Open("sqlite3", "../egym.db")
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`%s%s%s%s%s`, `SELECT TYPE, DURATION, CALORIES, FINISH_TIME FROM exercises WHERE TYPE="`, exerciseType, `" AND USER_ID=`, userID, ` AND START_TIME BETWEEN DATE("NOW", "-29 days") AND DATE("NOW", "-1 day") ORDER BY START_TIME DESC`)

	result, err := database.Query(query)
	if err != nil {
		return nil, err
	}

	userExercises, err := setResult(result)

	return userExercises, nil
}

func getTotalPointsByUser(userID string) (*User, error) {
	pointsByUser := []*PointsByType{}
	for i := range exerciseTypes {
		userExercises, err := getExercisesByType(exerciseTypes[i], userID)
		if err != nil {
			return nil, err
		}

		pointsByType := calculatePointsByExerciseType(userID, exerciseTypes[i], userExercises)
		pointsByUser = append(pointsByUser, pointsByType)
	}

	totalPointsByUser, err := totalPointsByUser(userID, pointsByUser)
	if err != nil {
		return nil, err
	}

	return totalPointsByUser, err
}

func getTotalPoints(users []string) ([]*User, error) {
	totalPoints := []*User{}
	for _, userID := range users {
		totalPointsByUser, err := getTotalPointsByUser(userID)
		if err != nil {
			return nil, err
		}

		totalPoints = append(totalPoints, totalPointsByUser)
	}

	return totalPoints, nil
}

func response(w http.ResponseWriter, httpStatus int, response *Response, err error) {
	if err != nil {
		response.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(response)
}

// RankingEndpoint function that handles request and response
func RankingEndpoint(w http.ResponseWriter, r *http.Request) {
	newResponse := &Response{}

	users, ok := r.URL.Query()["userIds"]
	if !ok || len(users[0]) < 1 {
		response(w, http.StatusBadRequest, newResponse, ErrInvalidUserIDs)
		return
	}

	totalPoints, err := getTotalPoints(users)
	if err != nil {
		response(w, http.StatusInternalServerError, newResponse, err)
		return
	}

	sort.Sort(ByPoints(totalPoints)) // sort points of users by points

	newResponse.Ranking = totalPoints
	response(w, http.StatusOK, newResponse, err)
}
