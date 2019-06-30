package service

import (
	"github.com/gidyon/rupacinema/movie/pkg/api"
	"github.com/gidyon/rupacinema/scheduling/pkg/logger"
	"go.uber.org/zap"
	"sync"
	"time"
)

// retrieves updated movie resource for a all days schedule after 10 minutes
func (scheduleAPI *scheduleAPIServer) updateMovies() {
	type updatedMovie struct {
		weekDay       int32
		showNumber    int32
		screen        string
		movieResource *movie.Movie
		err           error
	}
	resChan := make(chan updatedMovie, maxMoviesVoted+1)

	for {
		select {
		case <-scheduleAPI.ctx.Done():
			return
		case <-time.After(time.Duration(20 * time.Minute)):
			// Send requests to retrieve movie resource concurrently
			// Use channel to send the result of the goroutine fetching the movie
			// Lock the mutex, update the movie and unlock it

			func() {
				// Waitgroup to wait for goroutines to finish
				wg := &sync.WaitGroup{}

				// lock the muSchedule mutex
				scheduleAPI.muSchedule.Lock()
				for _, weekDay := range weekDays {
					weekDay := weekDay
					for _, screen := range screensAvailable {
						screen := screen
						for _, show := range showsPerDay {
							showNumber := show.showID
							showSchedule, err := scheduleAPI.getShowSchedule(
								weekDay, showNumber, screen,
							)
							if err != nil {
								logger.Log.Warn(
									"error while getting schedule",
									zap.Error(err),
									zap.String("Operation", "updateMovies"),
								)
								continue
							}
							// Range voted movies
							for _, votedMovie := range showSchedule.VotedMovies {
								wg.Add(1)
								go func(movieID string) {
									// Get the movie resource
									movieItem, err := scheduleAPI.movieAPIClient.GetMovie(
										scheduleAPI.ctx,
										&movie.GetMovieRequest{
											MovieId: movieID,
										},
									)
									// Send the movie resource to the channel
									select {
									case <-scheduleAPI.ctx.Done():
										return
									case resChan <- updatedMovie{
										weekDay:       weekDay,
										showNumber:    showNumber,
										screen:        screen,
										movieResource: movieItem,
										err:           err,
									}:
									}
								}(votedMovie.Id)
							}

							// Get the movie in display
							wg.Add(1)
							movieID := showSchedule.Movie.Id
							go func() {
								// Get the movie resource
								movieItem, err := scheduleAPI.movieAPIClient.GetMovie(
									scheduleAPI.ctx,
									&movie.GetMovieRequest{
										MovieId: movieID,
									},
								)
								// Send the movie resource to the channel
								select {
								case <-scheduleAPI.ctx.Done():
									return
								case resChan <- updatedMovie{
									weekDay:       weekDay,
									showNumber:    showNumber,
									screen:        screen,
									movieResource: movieItem,
									err:           err,
								}:
								}

							}()
						}
					}
				}
				// Unlock the mutex
				scheduleAPI.muSchedule.Unlock()

				go func() {
					// Wait for all goroutines to complete
					wg.Wait()
					// Then close the result channel
					close(resChan)
				}()

				// Range over the results
				for res := range resChan {

					// Lock the mutex
					scheduleAPI.muSchedule.Lock()

					if res.err != nil {
						logger.Log.Error(
							"Error while fetching result",
							zap.Error(res.err),
						)
						continue
					}
					scheduleAPI.updateMovieInfo(
						res.weekDay, res.showNumber, res.screen, res.movieResource,
					)

					// Unlock the mutex
					scheduleAPI.muSchedule.Unlock()
				}
			}()
		}
	}
}
