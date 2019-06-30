package service

import (
	"context"
	"github.com/Sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// checks whether a given context has been cancelled
func cancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
	}
	return false
}

// contextError wraps context error to a gRPC error
func contextError(ctx context.Context, operation string) error {
	if _, ok := ctx.Err().(interface{ Timeout() bool }); ok {
		// Should retry the request
		return status.Errorf(codes.DeadlineExceeded, "couldn't complete %s operation: %v", operation, ctx.Err())
	}
	return status.Errorf(codes.Canceled, "couldn't complete CreateProduct operation: %v", ctx.Err())
}

func logWarn(format string, args ...interface{}) {
	logrus.Warnf(format, args...)
}

func logInfo(format string, args ...interface{}) {
	logrus.Infof(format, args...)
}

func logError(format string, args ...interface{}) {
	logrus.Errorf(format, args...)
}

func errCheckingCreds(err error) error {
	return status.Errorf(codes.Internal, "failed while checking credentials: %v", err)
}

func errPermissionDenied(op string) error {
	return status.Errorf(codes.PermissionDenied, "not authorised to perform %s operation", op)
}

func errFromJSONMarshal(err error, obj string) error {
	return status.Errorf(codes.Internal, "failed to marshal %s: %v", obj, err)
}

func errFromJSONUnMarshal(err error, obj string) error {
	return status.Errorf(codes.Internal, "failed to unmarshal %s: %v", obj, err)
}

func errFromProtoMarshal(err error, obj string) error {
	return status.Errorf(codes.Internal, "failed to proto marshal %s: %v", obj, err)
}

func errFromProtoUnMarshal(err error, obj string) error {
	return status.Errorf(codes.Internal, "failed to proto unmarshal %s: %v", obj, err)
}

func errQueryFailed(err error, queryType string) error {
	return status.Errorf(codes.Internal, "failed to execute %s query: %v", queryType, err)
}

func errMissingCredential(cred string) error {
	return status.Errorf(codes.FailedPrecondition, "missing credentials: %v", cred)
}

func errNilObject(obj string) error {
	return status.Errorf(codes.InvalidArgument, "cannot accept nil %s value", obj)
}

func errRedisCmdFailed(err error, queryType string) error {
	return status.Errorf(codes.Internal, "failed to execute %s command: %v", queryType, err)
}

func errConvertingType(err error, from, to string) error {
	return status.Errorf(codes.Internal, "couldn't convert from %s to %s: %v", from, to, err)
}

func errNonExistentKey(key string) error {
	return status.Errorf(codes.InvalidArgument, "key %s doesn't exist in cache", key)
}

func errFailedTypeConversion(from, to string) error {
	return status.Errorf(codes.Internal, "type conversion from: %q to %q failed", from, to)
}

func errIncorrectVal(val string) error {
	return status.Errorf(codes.InvalidArgument, "incorrect value for %q", val)
}

func errNoMovieScheduleForWeekday(weekday int32) error {
	return status.Errorf(codes.Unknown, "no movie schedule for week day: %d", weekday)
}

func errNoMovieScheduleForScreen(screen string) error {
	return status.Errorf(codes.Unknown, "no movie schedule for screen: %q", screen)
}

func errNoMovieScheduleForShow(showNumber int32) error {
	return status.Errorf(codes.Unknown, "no show schedule for show number %q", showNumber)
}

func errNoMovieScheduleExist(movieID string) error {
	return status.Errorf(codes.Unknown, "no schedule for movie with %q", movieID)
}

func errMovieScheduleExist(movieID string) error {
	return status.Errorf(codes.Unknown, "schedule for movie with %q exists", movieID)
}

func errNoVotedMovieRoom() error {
	return status.Error(codes.ResourceExhausted, "no room to add voted movie")
}
