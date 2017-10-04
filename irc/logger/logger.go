// Copyright (c) 2017 Daniel Oaks <daniel@danieloaks.net>
// released under the MIT license

package logger

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"strings"

	"sync"

	colorable "github.com/mattn/go-colorable"
	"github.com/mgutz/ansi"
)

// Level represents the level to log messages at.
type Level int

const (
	// LogDebug represents debug messages.
	LogDebug Level = iota
	// LogInfo represents informational messages.
	LogInfo
	// LogWarning represents warnings.
	LogWarning
	// LogError represents errors.
	LogError
)

var (
	// LogLevelNames takes a config name and gives the real log level.
	LogLevelNames = map[string]Level{
		"debug":    LogDebug,
		"info":     LogInfo,
		"warn":     LogWarning,
		"warning":  LogWarning,
		"warnings": LogWarning,
		"error":    LogError,
		"errors":   LogError,
	}
	// LogLevelDisplayNames gives the display name to use for our log levels.
	LogLevelDisplayNames = map[Level]string{
		LogDebug:   "debug",
		LogInfo:    "info",
		LogWarning: "warning",
		LogError:   "error",
	}
)

// Manager is the main interface used to log debug/info/error messages.
type Manager struct {
	configMutex     sync.RWMutex
	loggers         []singleLogger
	stdoutWriteLock sync.Mutex // use one lock for both stdout and stderr
	fileWriteLock   sync.Mutex
	loggingRawIO    bool
}

// Config represents the configuration of a single logger.
type LoggingConfig struct {
	Method        string
	MethodStdout  bool
	MethodStderr  bool
	MethodFile    bool
	Filename      string
	TypeString    string   `yaml:"type"`
	Types         []string `yaml:"real-types"`
	ExcludedTypes []string `yaml:"real-excluded-types"`
	LevelString   string   `yaml:"level"`
	Level         Level    `yaml:"level-real"`
}

// NewManager returns a new log manager.
func NewManager(config []LoggingConfig) (*Manager, error) {
	var logger Manager

	if err := logger.ApplyConfig(config); err != nil {
		return nil, err
	}

	return &logger, nil
}

func (logger *Manager) ApplyConfig(config []LoggingConfig) error {
	logger.configMutex.Lock()
	defer logger.configMutex.Unlock()

	for _, logger := range logger.loggers {
		logger.Close()
	}

	logger.loggers = nil
	logger.loggingRawIO = false

	// for safety, this deep-copies all mutable data in `config`
	// XXX let's keep it that way
	var lastErr error
	for _, logConfig := range config {
		typeMap := make(map[string]bool)
		for _, name := range logConfig.Types {
			typeMap[name] = true
		}
		excludedTypeMap := make(map[string]bool)
		for _, name := range logConfig.ExcludedTypes {
			excludedTypeMap[name] = true
		}

		sLogger := singleLogger{
			MethodSTDOUT: logConfig.MethodStdout,
			MethodSTDERR: logConfig.MethodStderr,
			MethodFile: fileMethod{
				Enabled:  logConfig.MethodFile,
				Filename: logConfig.Filename,
			},
			Level:           logConfig.Level,
			Types:           typeMap,
			ExcludedTypes:   excludedTypeMap,
			stdoutWriteLock: &logger.stdoutWriteLock,
			fileWriteLock:   &logger.fileWriteLock,
		}
		if typeMap["userinput"] || typeMap["useroutput"] || (typeMap["*"] && !(excludedTypeMap["userinput"] && excludedTypeMap["useroutput"])) {
			logger.loggingRawIO = true
		}
		if sLogger.MethodFile.Enabled {
			file, err := os.OpenFile(sLogger.MethodFile.Filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
			if err != nil {
				lastErr = fmt.Errorf("Could not open log file %s [%s]", sLogger.MethodFile.Filename, err.Error())
			}
			writer := bufio.NewWriter(file)
			sLogger.MethodFile.File = file
			sLogger.MethodFile.Writer = writer
		}
		logger.loggers = append(logger.loggers, sLogger)
	}

	return lastErr
}

func (logger *Manager) IsLoggingRawIO() bool {
	logger.configMutex.RLock()
	defer logger.configMutex.RUnlock()
	return logger.loggingRawIO
}

// Log logs the given message with the given details.
func (logger *Manager) Log(level Level, logType string, messageParts ...string) {
	logger.configMutex.RLock()
	defer logger.configMutex.RUnlock()

	for _, singleLogger := range logger.loggers {
		singleLogger.Log(level, logType, messageParts...)
	}
}

// Debug logs the given message as a debug message.
func (logger *Manager) Debug(logType string, messageParts ...string) {
	logger.Log(LogDebug, logType, messageParts...)
}

// Info logs the given message as an info message.
func (logger *Manager) Info(logType string, messageParts ...string) {
	logger.Log(LogInfo, logType, messageParts...)
}

// Warning logs the given message as a warning message.
func (logger *Manager) Warning(logType string, messageParts ...string) {
	logger.Log(LogWarning, logType, messageParts...)
}

// Error logs the given message as an error message.
func (logger *Manager) Error(logType string, messageParts ...string) {
	logger.Log(LogError, logType, messageParts...)
}

// Fatal logs the given message as an error message, then exits.
func (logger *Manager) Fatal(logType string, messageParts ...string) {
	logger.Error(logType, messageParts...)
	logger.Error("FATAL", "Fatal error encountered, application exiting")
	os.Exit(1)
}

type fileMethod struct {
	Enabled  bool
	Filename string
	File     *os.File
	Writer   *bufio.Writer
}

// singleLogger represents a single logger instance.
type singleLogger struct {
	stdoutWriteLock *sync.Mutex
	fileWriteLock   *sync.Mutex
	MethodSTDOUT    bool
	MethodSTDERR    bool
	MethodFile      fileMethod
	Level           Level
	Types           map[string]bool
	ExcludedTypes   map[string]bool
}

func (logger *singleLogger) Close() error {
	if logger.MethodFile.Enabled {
		flushErr := logger.MethodFile.Writer.Flush()
		closeErr := logger.MethodFile.File.Close()
		if flushErr != nil {
			return flushErr
		}
		return closeErr
	}
	return nil
}

// Log logs the given message with the given details.
func (logger *singleLogger) Log(level Level, logType string, messageParts ...string) {
	// no logging enabled
	if !(logger.MethodSTDOUT || logger.MethodSTDERR || logger.MethodFile.Enabled) {
		return
	}

	// ensure we're logging to the given level
	if level < logger.Level {
		return
	}

	// ensure we're capturing this logType
	logTypeCleaned := strings.ToLower(strings.TrimSpace(logType))
	capturing := (logger.Types["*"] || logger.Types[logTypeCleaned]) && !logger.ExcludedTypes["*"] && !logger.ExcludedTypes[logTypeCleaned]
	if !capturing {
		return
	}

	// assemble full line
	timeGrey := ansi.ColorFunc("243")
	grey := ansi.ColorFunc("8")
	alert := ansi.ColorFunc("232+b:red")
	warn := ansi.ColorFunc("black:214")
	info := ansi.ColorFunc("117")
	debug := ansi.ColorFunc("78")
	section := ansi.ColorFunc("229")

	levelDisplay := LogLevelDisplayNames[level]
	if level == LogError {
		levelDisplay = alert(levelDisplay)
	} else if level == LogWarning {
		levelDisplay = warn(levelDisplay)
	} else if level == LogInfo {
		levelDisplay = info(levelDisplay)
	} else if level == LogDebug {
		levelDisplay = debug(levelDisplay)
	}

	sep := grey(":")
	fullStringFormatted := fmt.Sprintf("%s %s %s %s %s %s ", timeGrey(time.Now().UTC().Format("2006-01-02T15:04:05Z")), sep, levelDisplay, sep, section(logType), sep)
	fullStringRaw := fmt.Sprintf("%s : %s : %s : ", time.Now().UTC().Format("2006-01-02T15:04:05Z"), LogLevelDisplayNames[level], logType)
	for i, p := range messageParts {
		fullStringFormatted += p
		fullStringRaw += p
		if i != len(messageParts)-1 {
			fullStringFormatted += " " + sep + " "
			fullStringRaw += " : "
		}
	}

	// output
	if logger.MethodSTDOUT {
		logger.stdoutWriteLock.Lock()
		fmt.Fprintln(colorable.NewColorableStdout(), fullStringFormatted)
		logger.stdoutWriteLock.Unlock()
	}
	if logger.MethodSTDERR {
		logger.stdoutWriteLock.Lock()
		fmt.Fprintln(colorable.NewColorableStderr(), fullStringFormatted)
		logger.stdoutWriteLock.Unlock()
	}
	if logger.MethodFile.Enabled {
		logger.fileWriteLock.Lock()
		logger.MethodFile.Writer.WriteString(fullStringRaw + "\n")
		logger.MethodFile.Writer.Flush()
		logger.fileWriteLock.Unlock()
	}
}
