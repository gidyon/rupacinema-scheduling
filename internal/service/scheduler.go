package service

import (
	"context"
	"github.com/gidyon/rupacinema/account/pkg/api"
	"github.com/gidyon/rupacinema/movie/pkg/api"
	"github.com/gidyon/rupacinema/scheduling/pkg/api"
	"github.com/golang/protobuf/ptypes/empty"
	"strings"
	"sync"
)

// days, screens, timeOfDay, seats, details
var (
	weekDays         = []int32{1, 2, 3, 4, 5, 6, 7}
	screensAvailable = []string{"Screen 1"}
	maxMoviesVoted   = 4
)

var showsPerDay = []struct {
	playtime string
	showID   int32
}{
	{"11am", 1},
	{"3pm", 2},
	{"6pm", 3},
	{"9pm", 4},
}

type scheduleAPIServer struct {
	ctx            context.Context
	muSchedule     sync.Mutex // guards weeklySchedule
	weeklySchedule scheduler.DaysSchedule
	// Remote Services
	accountServiceClient account.AccountAPIClient
	movieAPIClient       movie.MovieAPIClient
}

// NewShowScheduler creates a new show scheduler service
func NewShowScheduler(
	ctx context.Context,
	accountServiceClient account.AccountAPIClient,
	movieAPIClient movie.MovieAPIClient,
) (scheduler.ShowSchedulerServer, error) {
	scheduleAPI := &scheduleAPIServer{
		ctx:        ctx,
		muSchedule: sync.Mutex{},
		weeklySchedule: scheduler.DaysSchedule{
			DaysSchedule: make(map[int32]*scheduler.ScreensSchedule),
		},
		// Remote Services
		accountServiceClient: accountServiceClient,
		movieAPIClient:       movieAPIClient,
	}

	err := scheduleAPI.initializeSchedule()
	if err != nil {
		return nil, err
	}

	// saves the current schedule in a file after 5 minutes
	go scheduleAPI.saveScheduleWorker()

	// worker that updates movies resource
	go scheduleAPI.updateMovies()

	return &scheduleAPIServer{}, nil
}

// FTW!
func (scheduleAPI *scheduleAPIServer) initializeSchedule() error {
	for _, weekDay := range weekDays {
		scheduleAPI.weeklySchedule.DaysSchedule[weekDay] = &scheduler.ScreensSchedule{
			ScreensSchedule: make(map[string]*scheduler.ShowsSchedule),
		}
		screensSchedule := scheduleAPI.weeklySchedule.DaysSchedule[weekDay].ScreensSchedule
		for _, screen := range screensAvailable {
			screensSchedule[screen] = &scheduler.ShowsSchedule{
				ShowsSchedule: make(map[int32]*scheduler.ShowSchedule),
			}
			showsSchedule := scheduleAPI.weeklySchedule.DaysSchedule[weekDay].ScreensSchedule[screen].ShowsSchedule
			for _, show := range showsPerDay {
				showsSchedule[show.showID] = &scheduler.ShowSchedule{
					PlayTime:    show.playtime,
					Movie:       &movie.Movie{},
					VotedMovies: make([]*movie.Movie, 0),
				}
			}
		}
	}
	return nil
}

// Assumes that the mutex gurading weeklySchedule is locked
func (scheduleAPI *scheduleAPIServer) getDaySchedule(
	weekDay int32,
) (*scheduler.ScreensSchedule, error) {
	daySchedule, ok := scheduleAPI.weeklySchedule.DaysSchedule[weekDay]
	if !ok {
		return nil, errNoMovieScheduleForWeekday(weekDay)
	}
	return daySchedule, nil
}

// Assumes that the mutex gurading weeklySchedule is locked
func (scheduleAPI *scheduleAPIServer) getShowSchedule(
	weekDay, show int32, screen string,
) (*scheduler.ShowSchedule, error) {
	daySchedule, err := scheduleAPI.getDaySchedule(weekDay)
	// Get day schedule
	if err != nil {
		return nil, err
	}

	// Get screen schedule
	screenSchedule, ok := daySchedule.ScreensSchedule[screen]
	if !ok {
		return nil, errNoMovieScheduleForScreen(screen)
	}

	// Get show schedule
	showSchedule, ok := screenSchedule.ShowsSchedule[show]
	if !ok {
		return nil, errNoMovieScheduleForShow(show)
	}

	return showSchedule, nil
}

// checks whether a movie exists in schedule and is not nil
// Assumes that the mutex gurading weeklySchedule is locked
func (scheduleAPI *scheduleAPIServer) existInSchedule(
	weekDay, show int32, screen, movieID string,
) (bool, error) {
	showSchedule, err := scheduleAPI.getShowSchedule(weekDay, show, screen)
	if err != nil {
		return false, err
	}

	if showSchedule.Movie == nil {
		return false, nil
	}

	if showSchedule.Movie.Id == movieID {
		return true, nil
	}

	return false, errMovieScheduleExist(movieID)
}

// Assumes that the mutex gurading weeklySchedule is locked
func (scheduleAPI *scheduleAPIServer) updateMovieInfo(
	weekDay, show int32, screen string, movieItem *movie.Movie,
) {
	showShedule, err := scheduleAPI.getShowSchedule(weekDay, show, screen)
	if err != nil {
		return
	}
	// Find the movie
	for _, movieSchedule := range append(showShedule.VotedMovies, showShedule.Movie) {
		// If the Id match, update the movie info
		if movieSchedule.Id == movieItem.Id {
			*movieSchedule = *movieItem
		}
	}
}

// The Pseudocode for voting up a movie:
// 1. Validate fields from request
// 2. Lock the mutex, and defer Unlock of the mutex
// 3. Get show for the day and screen based on the input fields
// 4. If the movie in show matches the Id of movie in request, increment current votes it and return
// 5. Otherwise range over the voted movies
// 6. When a match pf movie id is found, increment current votes
// 7. Swap the movies if necessary
// 8. Return the updated movie
func (scheduleAPI *scheduleAPIServer) VoteUpMovie(
	ctx context.Context, voteReq *scheduler.VoteUpMovieRequest,
) (*movie.Movie, error) {
	// Authenticate the request
	_, err := scheduleAPI.accountServiceClient.AuthenticateRequest(
		ctx, &empty.Empty{},
	)
	if err != nil {
		return nil, err
	}

	movieID := voteReq.GetMovieId()
	userID := voteReq.GetUserId()
	screen := voteReq.GetScreen()
	weekDay := voteReq.GetWeekDay()
	showNumber := voteReq.GetShowNumber()

	// Validate the input
	err := func() error {
		var err error
		switch {
		case strings.Trim(screen, " ") == "":
			err = errMissingCredential("Screen")
		case strings.Trim(userID, " ") == "":
			err = errMissingCredential("User Id")
		case strings.Trim(movieID, " ") == "":
			err = errMissingCredential("Movie Id")
		case weekDay <= 0 || weekDay > 7:
			err = errIncorrectVal("Week day")
		case showNumber <= 0:
			err = errIncorrectVal("Show number")
		}
		return err
	}()
	if err != nil {
		return nil, err
	}

	// lock the muSchedule mutex and defer unlock
	scheduleAPI.muSchedule.Lock()
	defer scheduleAPI.muSchedule.Unlock()

	// Get the show
	showSchedule, err := scheduleAPI.getShowSchedule(weekDay, showNumber, screen)
	if err != nil {
		return nil, err
	}

	// Increment the votes of the currently selected show
	if showSchedule.Movie.Id == movieID {
		showSchedule.Movie.CurrentVotes++
		return showSchedule.Movie, nil
	}

	// Increment the vote in voted movies section and swap the result if necessary
	for _, movieItem := range showSchedule.VotedMovies {
		if movieItem.Id == movieID {
			movieItem.CurrentVotes++
			break
		}
	}

	// Change the movie in show depending on the votes between display movie and the voted movies
	swapMovies(showSchedule.Movie, showSchedule.VotedMovies)

	// Vote up the movie
	return showSchedule.Movie, nil
}

// The Pseudocode:
// 1. Validate input fields from the request
// 2. Get the movie resource remotely
// 3. Lock the mutex and defer unlock
// 4. Check that the movie does not exist in schedule, return an error if so
// 5. Only then add the movie in schedule
// 6. Return success
func (scheduleAPI *scheduleAPIServer) CreateMovieDaySchedule(
	ctx context.Context, makeReq *scheduler.CreateMovieDayScheduleRequest,
) (*empty.Empty, error) {
	// Authenticate the request
	_, err := scheduleAPI.accountServiceClient.AuthenticateRequest(
		ctx, &empty.Empty{},
	)
	if err != nil {
		return nil, err
	}

	weekDay := makeReq.GetWeekDay()
	screen := makeReq.GetScreen()
	showNumber := makeReq.GetShow()
	movieID := makeReq.GetMovieId()

	// Validate the input
	err := func() error {
		var err error
		switch {
		case weekDay <= 0 || weekDay > 7:
			err = errIncorrectVal("Week day")
		case strings.Trim(screen, " ") == "":
			err = errMissingCredential("Screen")
		case strings.Trim(movieID, " ") == "":
			err = errMissingCredential("Movie Id")
		case showNumber <= 0:
			err = errIncorrectVal("Show number")
		}
		return err
	}()
	if err != nil {
		return nil, err
	}

	// Get the movie resource
	movieItem, err := scheduleAPI.movieAPIClient.GetMovie(
		ctx,
		&movie.GetMovieRequest{
			MovieId: movieID,
		},
	)
	if err != nil {
		return nil, err
	}

	// lock the muSchedule mutex and defer unlock
	scheduleAPI.muSchedule.Lock()
	defer scheduleAPI.muSchedule.Unlock()

	// Ensure the movie exists does not exist in schedule
	ok, err := scheduleAPI.existInSchedule(weekDay, showNumber, screen, movieItem.Id)
	if err != nil {
		return nil, err
	}

	// Return err if it exist in schedule
	if ok {
		return nil, errMovieScheduleExist(movieItem.Id)
	}

	// Add the movie in schedule
	showSchedule, _ := scheduleAPI.getShowSchedule(weekDay, showNumber, screen)
	showSchedule.Movie = movieItem

	return &empty.Empty{}, nil
}

// The Pseudocode:
// 1. Validate the input fields from the request
// 2. Get the remote movie
// 3. Check that there is room to add the voted movie, if not we fail early
// 4. Lock the mutex and defer unlock
// 5. Check that the movie hasn't already been voted
// 6. Only after step 5, do we add the movie in voted movies section
// 7. Return successful
func (scheduleAPI *scheduleAPIServer) AddVotedMovie(
	ctx context.Context, addReq *scheduler.AddVotedMovieRequest,
) (*empty.Empty, error) {
	// Authenticate the request
	_, err := scheduleAPI.accountServiceClient.AuthenticateRequest(
		ctx, &empty.Empty{},
	)
	if err != nil {
		return nil, err
	}

	weekDay := addReq.GetWeekDay()
	screen := addReq.GetScreen()
	showNumber := addReq.GetShow()
	movieID := addReq.GetMovieId()

	// Validate the input fields from request
	err := func() error {
		var err error
		switch {
		case weekDay <= 0 || weekDay > 7:
			err = errIncorrectVal("Week day")
		case strings.Trim(screen, " ") == "":
			err = errMissingCredential("Screen")
		case showNumber <= 0:
			err = errIncorrectVal("Show number")
		case strings.Trim(movieID, " ") == "":
			err = errMissingCredential("Movie ID")
		}
		return err
	}()
	if err != nil {
		return nil, err
	}

	// Get the movie resource
	movieItem, err := scheduleAPI.movieAPIClient.GetMovie(
		ctx,
		&movie.GetMovieRequest{
			MovieId: movieID,
		},
	)
	if err != nil {
		return nil, err
	}

	// lock the muSchedule mutex and defer unlock
	scheduleAPI.muSchedule.Lock()
	defer scheduleAPI.muSchedule.Unlock()

	// Ensure the movie exists does not exist in schedule
	ok, err := scheduleAPI.existInSchedule(weekDay, showNumber, screen, movieItem.Id)
	if err != nil {
		return nil, err
	}

	// Return err if it exist in schedule
	if ok {
		return nil, errMovieScheduleExist(movieItem.Id)
	}

	// Check there is room to add to voted movie
	showSchedule, _ := scheduleAPI.getShowSchedule(weekDay, showNumber, screen)
	if len(showSchedule.VotedMovies) >= maxMoviesVoted {
		return nil, errNoVotedMovieRoom()
	}

	// Add movie to voted movies section
	showSchedule.VotedMovies = append(showSchedule.VotedMovies, movieItem)

	return &empty.Empty{}, nil
}

// The Pseudocode:
// 1. Validate the input fields from the request
// 2. Lock mutex and defer Unlock defer
// 3. Endure that the movie exists in schedule, if it isn't return
// 4. Set the CurrentVotes to be -ve and swap the movie with voted movies
// NB: This will ensure the swapping is successful
// 5. Delete the movie that has been swapped to voted movies
// 6. Return success
func (scheduleAPI *scheduleAPIServer) DeleteMovieDaySchedule(
	ctx context.Context, delReq *scheduler.DeleteMovieDayScheduleRequest,
) (*empty.Empty, error) {
	// Authenticate the request
	_, err := scheduleAPI.accountServiceClient.AuthenticateRequest(
		ctx, &empty.Empty{},
	)
	if err != nil {
		return nil, err
	}

	weekDay := delReq.GetWeekDay()
	screen := delReq.GetScreen()
	showNumber := delReq.GetShow()
	movieID := delReq.GetMovieId()

	// Validate the input fields from request
	err := func() error {
		var err error
		switch {
		case weekDay <= 0 || weekDay > 7:
			err = errIncorrectVal("Week day")
		case strings.Trim(screen, " ") == "":
			err = errMissingCredential("Screen")
		case showNumber <= 0:
			err = errIncorrectVal("Show number")
		case strings.Trim(movieID, " ") == "":
			err = errMissingCredential("Movie ID")
		}
		return err
	}()
	if err != nil {
		return nil, err
	}

	// lock the muSchedule mutex and defer unlock
	scheduleAPI.muSchedule.Lock()
	defer scheduleAPI.muSchedule.Unlock()

	// Ensure the movie exists in schedule
	ok, err := scheduleAPI.existInSchedule(weekDay, showNumber, screen, movieID)
	if err != nil {
		return nil, err
	}

	// Return err if it doesn't exist in schedule
	if !ok {
		return nil, errNoMovieScheduleExist(movieID)
	}

	// Ok will be true
	showSchedule, _ := scheduleAPI.getShowSchedule(weekDay, showNumber, screen)

	// So that the swapping succeeds
	showSchedule.Movie.CurrentVotes = -1

	// Swap the movie to be deleted to go into voted movies
	index := swapMovies(showSchedule.Movie, showSchedule.VotedMovies)

	// Remove the movie that has been swapped to voted movies
	showSchedule.VotedMovies = append(
		showSchedule.VotedMovies[:index], showSchedule.VotedMovies[index+1:]...,
	)

	return &empty.Empty{}, nil
}

func (scheduleAPI *scheduleAPIServer) GetDaySchedule(
	ctx context.Context, getReq *scheduler.GetDayScheduleRequest,
) (*scheduler.ScreensSchedule, error) {
	// Check that weekday is provided
	weekDay := getReq.GetWeekDay()
	if weekDay <= 0 || weekDay > 7 {
		return nil, errIncorrectVal("Week Day")
	}

	// lock the muSchedule mutex and defer unlock
	scheduleAPI.muSchedule.Lock()
	defer scheduleAPI.muSchedule.Unlock()

	daySchedule, err := scheduleAPI.getDaySchedule(weekDay)
	if err != nil {
		return nil, err
	}

	return daySchedule, nil
}

func (scheduleAPI *scheduleAPIServer) GetShowSchedule(
	ctx context.Context, getReq *scheduler.GetShowScheduleRequest,
) (*scheduler.ShowSchedule, error) {
	weekDay := getReq.GetWeekDay()
	screen := getReq.GetScreen()
	showNumber := getReq.GetShow()

	err := func() error {
		var err error
		switch {
		case weekDay <= 0 || weekDay > 7:
			err = errIncorrectVal("Week day")
		case strings.Trim(screen, " ") == "":
			err = errMissingCredential("Screen")
		case showNumber <= 0:
			err = errIncorrectVal("Show number")
		}
		return err
	}()
	if err != nil {
		return nil, err
	}

	// lock the muSchedule mutex and defer unlock
	scheduleAPI.muSchedule.Lock()
	defer scheduleAPI.muSchedule.Unlock()

	showSchedule, err := scheduleAPI.getShowSchedule(weekDay, showNumber, screen)
	if err != nil {
		return nil, err
	}

	return showSchedule, nil
}

func higherVotesIndex(movies []*movie.Movie) int {
	votes := movies[0].CurrentVotes
	index := 0

	for i, movieItem := range movies {
		if movieItem.CurrentVotes > votes {
			votes = movieItem.CurrentVotes
			index = i
		}
	}

	return index
}

func swapMovies(movieItem *movie.Movie, votedMovies []*movie.Movie) int {
	index := higherVotesIndex(votedMovies)

	if movieItem.CurrentVotes < votedMovies[index].CurrentVotes {
		temp := *movieItem
		*movieItem = *votedMovies[index]
		*votedMovies[index] = temp
	}

	return index
}
