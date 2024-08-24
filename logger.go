package logger

import (
	"context"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

var (
	_logger *logger
)

type Event struct {
	instance *zerolog.Logger
}

type logger struct {
	instance *zerolog.Logger
	options loggerOptions
}

func (l *Event) Infof(format string, args ...interface{}) {
	l.instance.Info().Msgf(format, args...)
}

func (l *Event) Debugf(format string, args ...interface{}) {
	l.instance.Debug().Msgf(format, args...)
}

func (l *Event) Errorf(format string, args ...interface{}) {
	l.instance.Error().Msgf(format, args...)
}

func (l *Event) Warnf(format string, args ...interface{}) {
	l.instance.Warn().Msgf(format, args...)
}

func (l *Event) Panicf(format string, args ...interface{}) {
	l.instance.Panic().Msgf(format, args...)
}

func (l *Event) Printf(format string, args ...interface{}) {
	l.instance.Printf(format, args...)
}

// Starts and configures the log instance
// and should be called at the start of the application.
//
// # Notice
//
// At this point in time the underlying logger used is zero log https://github.com/rs/zerolog
func Init(opt ...LoggerOption) {
	var (
		l            = zerolog.New(os.Stdout)
		opts = defaultLoggerOptions
	)

	for _, o := range opt {
		o.apply(&opts)
	}

	switch opts.env {
	case string(test):
		l.Output(io.Discard)
	case string(development):
		l = l.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		l.Level(zerolog.DebugLevel)
	case string(production), string(staging):
		l.Level(zerolog.InfoLevel)
	default:
		l.Level(zerolog.DebugLevel)
	}

	_logger = &logger{instance: &l, options: opts}
}

// Gets a basic logger instance with no tracing.
//
// # Note
//  - This function is only recommended for simple logging, like application startup and configuration.
//  - For more in-depth logging, please refer to the Trace method.
func Get() *Event {
	var (
		l zerolog.Logger
		_, fileName, _, _ = runtime.Caller(1)
	)

	l = _logger.instance.With().Str("file", fileName).Logger()
	return &Event{instance: &l}
}

// Gets a logger instance with a trace ID.
//
// This function is recomended for a more in-depth logging.
//
// # Note
//
// The trace id key name in the log is set by the Option TraceKey. The default value is "cid".
func Trace(pctx context.Context) (log *Event, c context.Context) {
	return trace(pctx)
}

func trace(pctx context.Context) (log *Event, ctx context.Context) {
	var (
		current, fileName, _, _ = runtime.Caller(2)
		callerName              = runtime.FuncForPC(current).Name()
		fileDetail              = strings.Split(callerName, "/")
		pkg                     = fileDetail[len(fileDetail)-2]
		method                  = fileDetail[len(fileDetail)-1]
		methods                 = strings.Split(method, ".")
		ctxKey 					= _logger.options.contextKey
		traceKey 				= _logger.options.traceKey
		traceId 				string
	)
	if len(methods) > 0 {
		method = methods[len(methods)-1]
	}

	if existingKey, ok := pctx.Value(ctxKey).(string); ok && existingKey != "" {
		traceId = existingKey
	} else {
		traceId = uuid.New().String()
	}

	ctx = context.WithValue(pctx, contextKey(ctxKey), traceId)
	
	// This cleans up a bit our logging in a development setting
	if _logger.options.env == string(development) {
		l := _logger.
			instance.
			With().
			Ctx(pctx).
			Str(traceKey, traceId).
			Logger()
		log = &Event{instance: &l}
		return log, ctx
	}
	
	l := _logger.
		instance.
		With().
		Ctx(pctx).
		Str(traceKey, traceId).
		Str("file", fileName).
		Str("method", method).
		Str("pkg", pkg).
		Logger()

	log = &Event{instance: &l}

	return log, ctx
}
