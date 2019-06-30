package service

import (
	"github.com/gidyon/rupacinema/scheduling/pkg/logger"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
	"os"
	"time"
)

// saves the current weekly schedule periodically in a file
func (scheduleAPI *scheduleAPIServer) saveScheduleWorker() {
	for {
		select {
		case <-scheduleAPI.ctx.Done():
			return
		case <-time.After(time.Duration(5 * time.Minute)):
			func() {
				f, err := os.OpenFile("snapshot", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 066)
				if err != nil {
					logger.Log.Warn("error while saving file", zap.Error(err))
					return
				}

				// Lock the mutex
				scheduleAPI.muSchedule.Lock()

				bs, err := proto.Marshal(&scheduleAPI.weeklySchedule)
				if err != nil {
					logger.Log.Error("error while marshaling file", zap.Error(err))
					return
				}

				// Unlock the mutex
				scheduleAPI.muSchedule.Unlock()

				_, err = f.Write(bs)
				if err != nil {
					if err != nil {
						logger.Log.Error("error while writing to file", zap.Error(err))
						return
					}
				}

			}()
		}
	}
}
