
// Package xLog offers simple cross platform logging for Windows and Linux.
// Available logging endpoints are event log (Windows), syslog (Linux), and
// an io.Writer.
package xlog

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

type log_level int

// Logging levels.
const (
	eFatal log_level = iota
	eError
	eWarn
	eInfo
	eTrace
)

// Logging tags.
const (
	tagFatal    	= "[F] "
	tagError 	= "[E] "
	tagWarn   	= "[W] "
	tagInfo   	= "[I] "
	tagTrace	= "[T] "
)

const (
	flags    = log.Ldate | log.Lmicroseconds | log.Lshortfile
	initText = "Logging before logger.Init.\n"
)

var (
	logLock       sync.Mutex
	defaultLogger *Logger
)

func initialize() {
	defaultLogger = &Logger{
		logFatal:   	log.New(os.Stderr, initText+tagFatal, flags),
		logError:   	log.New(os.Stderr, initText+tagError, flags),
		logWarn: 	log.New(os.Stderr, initText+tagWarn, flags),
		logInfo:    	log.New(os.Stderr, initText+tagInfo, flags),
		logTrace: 	log.New(os.Stderr, initText+tagTrace, flags),
	}
}

/**
 * 初始化默认Logger实例。
 * 
 * @return {[type]} [description]
 */
func init() {
	initialize()
}

// Init sets up logging and should be called before log functions, usually in
// the caller's main(). Default log functions can be called before Init(), but log
// output will only go to stderr (along with a warning).
// The first call to Init populates the default logger and returns the
// generated logger, subsequent calls to Init will only return the generated
// logger.
// If the logFile passed in also satisfies io.Closer, logFile.Close will be called
// when closing the logger.
func Init(name string, verbose, systemLog bool, logFile io.Writer) *Logger {
	var il, wl, el io.Writer
	var syslogErr error
	if systemLog {
		il, wl, el, syslogErr = setup(name)
	}

	iLogs := []io.Writer{logFile}
	wLogs := []io.Writer{logFile}
	eLogs := []io.Writer{logFile}
	if il != nil {
		iLogs = append(iLogs, il)
	}
	if wl != nil {
		wLogs = append(wLogs, wl)
	}
	if el != nil {
		eLogs = append(eLogs, el)
	}
	// Windows services don't have stdout/stderr. Writes will fail, so try them last.
	eLogs = append(eLogs, os.Stderr)
	if verbose {
		iLogs = append(iLogs, os.Stdout)
		wLogs = append(wLogs, os.Stdout)
	}

	l := Logger{
		logFatal:   	log.New(io.MultiWriter(eLogs...), tagFatal, flags),
		logError:   	log.New(io.MultiWriter(eLogs...), tagError, flags),
		logWarn: 	log.New(io.MultiWriter(wLogs...), tagWarn, flags),
		logInfo:    	log.New(io.MultiWriter(iLogs...), tagInfo, flags),
		logTrace:    	log.New(io.MultiWriter(iLogs...), tagTrace, flags),
		
	}
	for _, w := range []io.Writer{logFile, il, wl, el} {
		if c, ok := w.(io.Closer); ok && c != nil {
			l.closers = append(l.closers, c)
		}
	}
	l.initialized = true

	logLock.Lock()
	defer logLock.Unlock()
	if !defaultLogger.initialized {
		defaultLogger = &l
	}

	if syslogErr != nil {
		Error(syslogErr)
	}

	return &l
}

// Close closes the default logger.
func Close() {
	defaultLogger.Close()
}

// A Logger represents an active logging object. Multiple loggers can be used
// simultaneously even if they are using the same same writers.
type Logger struct {
	logFatal    	*log.Logger
	logError    	*log.Logger
	logWarn  	*log.Logger
	logInfo     	*log.Logger
	logTrace    	*log.Logger
	closers     	[]io.Closer
	initialized 	bool
}

func (l *Logger) output(level log_level, depth int, txt string) {
	logLock.Lock()
	defer logLock.Unlock()
	switch level {
	case eTrace:
		l.logTrace.Output(3+depth, txt)
	case eInfo:
		l.logInfo.Output(3+depth, txt)
	case eWarn:
		l.logWarn.Output(3+depth, txt)
	case eError:
		l.logError.Output(3+depth, txt)
	case eFatal:
		l.logFatal.Output(3+depth, txt)
	default:
		panic(fmt.Sprintln("unrecognized log_level:", level))
	}
}

// Close closes all the underlying log writers, which will flush any cached logs.
// Any errors from closing the underlying log writers will be printed to stderr.
// Once Close is called, all future calls to the logger will panic.
func (l *Logger) Close() {
	logLock.Lock()
	defer logLock.Unlock()

	if !l.initialized {
		return
	}

	for _, c := range l.closers {
		if err := c.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close log %v: %v\n", c, err)
		}
	}
}

// Trace logs with the eTrace log_level.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Trace(v ...interface{}) {
	l.output(eTrace, 0, fmt.Sprint(v...))
}

// TraceDepth acts as Trace but uses depth to determine which call frame to log.
// TraceDepth(0, "msg") is the same as Trace("msg").
func (l *Logger) TraceDepth(depth int, v ...interface{}) {
	l.output(eTrace, depth, fmt.Sprint(v...))
}

// Traceln logs with the eTrace log_level.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Traceln(v ...interface{}) {
	l.output(eTrace, 0, fmt.Sprintln(v...))
}

// Tracef logs with the eTrace log_level.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Tracef(format string, v ...interface{}) {
	l.output(eTrace, 0, fmt.Sprintf(format, v...))
}

// Info logs with the Info log_level.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Info(v ...interface{}) {
	l.output(eInfo, 0, fmt.Sprint(v...))
}

// InfoDepth acts as Info but uses depth to determine which call frame to log.
// InfoDepth(0, "msg") is the same as Info("msg").
func (l *Logger) InfoDepth(depth int, v ...interface{}) {
	l.output(eInfo, depth, fmt.Sprint(v...))
}

// Infoln logs with the Info log_level.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Infoln(v ...interface{}) {
	l.output(eInfo, 0, fmt.Sprintln(v...))
}

// Infof logs with the Info log_level.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Infof(format string, v ...interface{}) {
	l.output(eInfo, 0, fmt.Sprintf(format, v...))
}

// Warning logs with the Warning log_level.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Warning(v ...interface{}) {
	l.output(eWarn, 0, fmt.Sprint(v...))
}

// WarningDepth acts as Warning but uses depth to determine which call frame to log.
// WarningDepth(0, "msg") is the same as Warning("msg").
func (l *Logger) WarningDepth(depth int, v ...interface{}) {
	l.output(eWarn, depth, fmt.Sprint(v...))
}

// Warningln logs with the Warning log_level.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Warningln(v ...interface{}) {
	l.output(eWarn, 0, fmt.Sprintln(v...))
}

// Warningf logs with the Warning log_level.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Warningf(format string, v ...interface{}) {
	l.output(eWarn, 0, fmt.Sprintf(format, v...))
}

// Error logs with the ERROR log_level.
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Error(v ...interface{}) {
	l.output(eError, 0, fmt.Sprint(v...))
}

// ErrorDepth acts as Error but uses depth to determine which call frame to log.
// ErrorDepth(0, "msg") is the same as Error("msg").
func (l *Logger) ErrorDepth(depth int, v ...interface{}) {
	l.output(eError, depth, fmt.Sprint(v...))
}

// Errorln logs with the ERROR log_level.
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Errorln(v ...interface{}) {
	l.output(eError, 0, fmt.Sprintln(v...))
}

// Errorf logs with the Error log_level.
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.output(eError, 0, fmt.Sprintf(format, v...))
}

// Fatal logs with the Fatal log_level, and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Print.
func (l *Logger) Fatal(v ...interface{}) {
	l.output(eFatal, 0, fmt.Sprint(v...))
	l.Close()
	os.Exit(1)
}

// FatalDepth acts as Fatal but uses depth to determine which call frame to log.
// FatalDepth(0, "msg") is the same as Fatal("msg").
func (l *Logger) FatalDepth(depth int, v ...interface{}) {
	l.output(eFatal, depth, fmt.Sprint(v...))
	l.Close()
	os.Exit(1)
}

// Fatalln logs with the Fatal log_level, and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Println.
func (l *Logger) Fatalln(v ...interface{}) {
	l.output(eFatal, 0, fmt.Sprintln(v...))
	l.Close()
	os.Exit(1)
}

// Fatalf logs with the Fatal log_level, and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Printf.
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.output(eFatal, 0, fmt.Sprintf(format, v...))
	l.Close()
	os.Exit(1)
}

// SetFlags sets the output flags for the logger.
func SetFlags(flag int) {
	defaultLogger.logTrace.SetFlags(flag)
	defaultLogger.logInfo.SetFlags(flag)
	defaultLogger.logWarn.SetFlags(flag)
	defaultLogger.logError.SetFlags(flag)
	defaultLogger.logFatal.SetFlags(flag)
}

// Trace uses the default logger and logs with the eTrace log_level.
// Arguments are handled in the manner of fmt.Print.
func Trace(v ...interface{}) {
	defaultLogger.output(eTrace, 0, fmt.Sprint(v...))
}

// TraceDepth acts as Trace but uses depth to determine which call frame to log.
// TraceDepth(0, "msg") is the same as Trace("msg").
func TraceDepth(depth int, v ...interface{}) {
	defaultLogger.output(eTrace, depth, fmt.Sprint(v...))
}

// Traceln uses the default logger and logs with the eTrace log_level.
// Arguments are handled in the manner of fmt.Println.
func Traceln(v ...interface{}) {
	defaultLogger.output(eTrace, 0, fmt.Sprintln(v...))
}

// Tracef uses the default logger and logs with the eTrace log_level.
// Arguments are handled in the manner of fmt.Printf.
func Tracef(format string, v ...interface{}) {
	defaultLogger.output(eTrace, 0, fmt.Sprintf(format, v...))
}

// Info uses the default logger and logs with the Info log_level.
// Arguments are handled in the manner of fmt.Print.
func Info(v ...interface{}) {
	defaultLogger.output(eInfo, 0, fmt.Sprint(v...))
}

// InfoDepth acts as Info but uses depth to determine which call frame to log.
// InfoDepth(0, "msg") is the same as Info("msg").
func InfoDepth(depth int, v ...interface{}) {
	defaultLogger.output(eInfo, depth, fmt.Sprint(v...))
}

// Infoln uses the default logger and logs with the Info log_level.
// Arguments are handled in the manner of fmt.Println.
func Infoln(v ...interface{}) {
	defaultLogger.output(eInfo, 0, fmt.Sprintln(v...))
}

// Infof uses the default logger and logs with the Info log_level.
// Arguments are handled in the manner of fmt.Printf.
func Infof(format string, v ...interface{}) {
	defaultLogger.output(eInfo, 0, fmt.Sprintf(format, v...))
}

// Warning uses the default logger and logs with the Warning log_level.
// Arguments are handled in the manner of fmt.Print.
func Warning(v ...interface{}) {
	defaultLogger.output(eWarn, 0, fmt.Sprint(v...))
}

// WarningDepth acts as Warning but uses depth to determine which call frame to log.
// WarningDepth(0, "msg") is the same as Warning("msg").
func WarningDepth(depth int, v ...interface{}) {
	defaultLogger.output(eWarn, depth, fmt.Sprint(v...))
}

// Warningln uses the default logger and logs with the Warning log_level.
// Arguments are handled in the manner of fmt.Println.
func Warningln(v ...interface{}) {
	defaultLogger.output(eWarn, 0, fmt.Sprintln(v...))
}

// Warningf uses the default logger and logs with the Warning log_level.
// Arguments are handled in the manner of fmt.Printf.
func Warningf(format string, v ...interface{}) {
	defaultLogger.output(eWarn, 0, fmt.Sprintf(format, v...))
}

// Error uses the default logger and logs with the Error log_level.
// Arguments are handled in the manner of fmt.Print.
func Error(v ...interface{}) {
	defaultLogger.output(eError, 0, fmt.Sprint(v...))
}

// ErrorDepth acts as Error but uses depth to determine which call frame to log.
// ErrorDepth(0, "msg") is the same as Error("msg").
func ErrorDepth(depth int, v ...interface{}) {
	defaultLogger.output(eError, depth, fmt.Sprint(v...))
}

// Errorln uses the default logger and logs with the Error log_level.
// Arguments are handled in the manner of fmt.Println.
func Errorln(v ...interface{}) {
	defaultLogger.output(eError, 0, fmt.Sprintln(v...))
}

// Errorf uses the default logger and logs with the Error log_level.
// Arguments are handled in the manner of fmt.Printf.
func Errorf(format string, v ...interface{}) {
	defaultLogger.output(eError, 0, fmt.Sprintf(format, v...))
}

// Fatalln uses the default logger, logs with the Fatal log_level,
// and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Print.
func Fatal(v ...interface{}) {
	defaultLogger.output(eFatal, 0, fmt.Sprint(v...))
	defaultLogger.Close()
	os.Exit(1)
}

// FatalDepth acts as Fatal but uses depth to determine which call frame to log.
// FatalDepth(0, "msg") is the same as Fatal("msg").
func FatalDepth(depth int, v ...interface{}) {
	defaultLogger.output(eFatal, depth, fmt.Sprint(v...))
	defaultLogger.Close()
	os.Exit(1)
}

// Fatalln uses the default logger, logs with the Fatal log_level,
// and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Println.
func Fatalln(v ...interface{}) {
	defaultLogger.output(eFatal, 0, fmt.Sprintln(v...))
	defaultLogger.Close()
	os.Exit(1)
}

// Fatalf uses the default logger, logs with the Fatal log_level,
// and ends with os.Exit(1).
// Arguments are handled in the manner of fmt.Printf.
func Fatalf(format string, v ...interface{}) {
	defaultLogger.output(eFatal, 0, fmt.Sprintf(format, v...))
	defaultLogger.Close()
	os.Exit(1)
}
