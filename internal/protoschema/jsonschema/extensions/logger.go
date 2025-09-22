// Copyright 2024-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package extensions

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// LogLevel represents the level of logging
type LogLevel int

const (
	LogLevelSilent LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
	LogLevelTrace
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelSilent:
		return "SILENT"
	case LogLevelError:
		return "ERROR"
	case LogLevelWarn:
		return "WARN"
	case LogLevelInfo:
		return "INFO"
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelTrace:
		return "TRACE"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging for extension processing
type Logger struct {
	level      LogLevel
	prefix     string
	enableTime bool
}

// NewLogger creates a new logger with the specified level and prefix
func NewLogger(level LogLevel, prefix string) *Logger {
	return &Logger{
		level:      level,
		prefix:     prefix,
		enableTime: true,
	}
}

// NewLoggerFromEnv creates a logger configured from environment variables
func NewLoggerFromEnv(prefix string) *Logger {
	level := LogLevelSilent

	// Check PROTOC_GEN_JSONSCHEMA_LOG_LEVEL environment variable
	if envLevel := os.Getenv("PROTOC_GEN_JSONSCHEMA_LOG_LEVEL"); envLevel != "" {
		switch strings.ToUpper(envLevel) {
		case "ERROR":
			level = LogLevelError
		case "WARN", "WARNING":
			level = LogLevelWarn
		case "INFO":
			level = LogLevelInfo
		case "DEBUG":
			level = LogLevelDebug
		case "TRACE":
			level = LogLevelTrace
		}
	}

	// Check legacy PROTOC_GEN_JSONSCHEMA_DEBUG for backward compatibility
	if os.Getenv("PROTOC_GEN_JSONSCHEMA_DEBUG") == "true" {
		level = LogLevelDebug
	}

	return NewLogger(level, prefix)
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// GetLevel returns the current logging level
func (l *Logger) GetLevel() LogLevel {
	return l.level
}

// IsEnabled returns true if the given level is enabled
func (l *Logger) IsEnabled(level LogLevel) bool {
	return level <= l.level
}

// Error logs an error message
func (l *Logger) Error(format string, args ...any) {
	l.log(LogLevelError, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...any) {
	l.log(LogLevelWarn, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...any) {
	l.log(LogLevelInfo, format, args...)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...any) {
	l.log(LogLevelDebug, format, args...)
}

// Trace logs a trace message
func (l *Logger) Trace(format string, args ...any) {
	l.log(LogLevelTrace, format, args...)
}

// log is the internal logging method
func (l *Logger) log(level LogLevel, format string, args ...any) {
	if !l.IsEnabled(level) {
		return
	}

	var prefix string
	if l.enableTime {
		prefix = fmt.Sprintf("[%s] [%s] %s: ",
			time.Now().Format("15:04:05.000"),
			level.String(),
			l.prefix)
	} else {
		prefix = fmt.Sprintf("[%s] %s: ", level.String(), l.prefix)
	}

	message := fmt.Sprintf(format, args...)
	log.Printf("%s%s", prefix, message)
}

// With creates a child logger with an additional prefix
func (l *Logger) With(additionalPrefix string) *Logger {
	newPrefix := l.prefix
	if newPrefix != "" {
		newPrefix += "." + additionalPrefix
	} else {
		newPrefix = additionalPrefix
	}

	return &Logger{
		level:      l.level,
		prefix:     newPrefix,
		enableTime: l.enableTime,
	}
}

// LogConfigLoad logs configuration loading events
func (l *Logger) LogConfigLoad(configPath string, phase string, details map[string]any) {
	if !l.IsEnabled(LogLevelInfo) {
		return
	}

	var detailsStr []string
	for key, value := range details {
		detailsStr = append(detailsStr, fmt.Sprintf("%s=%v", key, value))
	}

	l.Info("config_load phase=%s path=%s %s",
		phase, configPath, strings.Join(detailsStr, " "))
}

// LogProcessorMatch logs option processor matching events
func (l *Logger) LogProcessorMatch(processorName string, fieldName string, matched bool, reason string) {
	if !l.IsEnabled(LogLevelDebug) {
		return
	}

	status := "NOMATCH"
	if matched {
		status = "MATCH"
	}

	l.Debug("processor_match processor=%s field=%s status=%s reason=%s",
		processorName, fieldName, status, reason)
}

// LogSchemaProcessing logs schema processing events
func (l *Logger) LogSchemaProcessing(schemaName string, phase string, details map[string]any) {
	if !l.IsEnabled(LogLevelDebug) {
		return
	}

	var detailsStr []string
	for key, value := range details {
		detailsStr = append(detailsStr, fmt.Sprintf("%s=%v", key, value))
	}

	l.Debug("schema_processing schema=%s phase=%s %s",
		schemaName, phase, strings.Join(detailsStr, " "))
}

// LogOptionExtraction logs option extraction events
func (l *Logger) LogOptionExtraction(fieldName string, optionPath string, found bool, value any) {
	if !l.IsEnabled(LogLevelTrace) {
		return
	}

	status := "NOTFOUND"
	if found {
		status = "FOUND"
	}

	l.Trace("option_extraction field=%s option=%s status=%s value=%v",
		fieldName, optionPath, status, value)
}

// LogTemplateSubstitution logs template substitution events
func (l *Logger) LogTemplateSubstitution(template string, result string, variables map[string]any) {
	if !l.IsEnabled(LogLevelTrace) {
		return
	}

	var varStr []string
	for key, value := range variables {
		varStr = append(varStr, fmt.Sprintf("%s=%v", key, value))
	}

	l.Trace("template_substitution template=%s result=%s variables=[%s]",
		template, result, strings.Join(varStr, ", "))
}

// LogPerformance logs performance metrics
func (l *Logger) LogPerformance(operation string, duration time.Duration, details map[string]any) {
	if !l.IsEnabled(LogLevelInfo) {
		return
	}

	var detailsStr []string
	for key, value := range details {
		detailsStr = append(detailsStr, fmt.Sprintf("%s=%v", key, value))
	}

	l.Info("performance operation=%s duration=%s %s",
		operation, duration, strings.Join(detailsStr, " "))
}

// LogError logs structured error information
func (l *Logger) LogError(operation string, err error, context map[string]any) {
	if !l.IsEnabled(LogLevelError) {
		return
	}

	var contextStr []string
	for key, value := range context {
		contextStr = append(contextStr, fmt.Sprintf("%s=%v", key, value))
	}

	l.Error("error operation=%s error=%v context=[%s]",
		operation, err, strings.Join(contextStr, ", "))
}

// Global logger instance
var defaultLogger = NewLoggerFromEnv("extensions")

// SetGlobalLogLevel sets the global logging level
func SetGlobalLogLevel(level LogLevel) {
	defaultLogger.SetLevel(level)
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	return defaultLogger
}

// Global logging functions for convenience
func Error(format string, args ...any) {
	defaultLogger.Error(format, args...)
}

func Warn(format string, args ...any) {
	defaultLogger.Warn(format, args...)
}

func Info(format string, args ...any) {
	defaultLogger.Info(format, args...)
}

func Debug(format string, args ...any) {
	defaultLogger.Debug(format, args...)
}

func Trace(format string, args ...any) {
	defaultLogger.Trace(format, args...)
}