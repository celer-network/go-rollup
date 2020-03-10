/*
Package log is a global and configurable logger pkg, based on zerolog (https://github.com/rs/zerolog)

You can configure this logger using a toml configuration file. Here is an example configuration that has all available fields.
However fields are optional. Even if you don't set some of them, logger will work well with a default value.

 # A default log level for all sub modules
 # must be one of this; debug/info/warn/error/fatal/panic
 level = "info"

 # A log output formatter
 # can be chosen among this; console, console_no_color, json
 formatter = "json"

 # Enabling source file and line printer
 caller = false

 # A time stamp field format.
 # e.g. in a time/format.go file
 # ANSIC       = "Mon Jan _2 15:04:05 2006"
 # RFC822      = "02 Jan 06 15:04 MST"
 # RFC1123     = "Mon, 02 Jan 2006 15:04:05 MST"
 # Kitchen     = "3:04PM"
 # Stamp       = "Jan _2 15:04:05"
 timefieldformat = "3:04 PM"

 # If there is a sub module and it has deferent options from defaults,
 # sub modules can be configured using a map struct of toml
 # currently, only setting sub modules's level is allowed
 [sub_module_name]
 level = "error"

 [can_have_multiple_module]
 level = "debug"

After creating a log configuration file, you must locate that to a same directory where binary file is.
Or you can register the config file path at an environment variable 'arglib_logconfig'.
Because this pkg is initialized at very early stage, faster than init() func, there is no way to get an arguments.
*/
package log

import (
	"errors"
	"os"
	"strings"
	"sync"

	colorable "github.com/mattn/go-colorable"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

var baseLogger = zerolog.New(os.Stderr)
var baseLevel = zerolog.InfoLevel
var logInitLock sync.Mutex
var isLogInit = false
var viperConf = viper.New()

var confFilePathKey = "LOGCONFIG"
var confEnvPrefix = "ARGLIB"
var defaultConfFileName = "arglog"

var defaultConfStr = ""

func loadConfigFile() *viper.Viper {
	// init viper
	viperConf.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viperConf.SetEnvPrefix(confEnvPrefix)
	viperConf.AutomaticEnv()

	// search a default conf file
	viperConf.SetConfigType("toml")
	viperConf.SetConfigName(defaultConfFileName)
	viperConf.AddConfigPath(".")

	// set the config file if path exist at environment
	if viperConf.GetString(confFilePathKey) != "" {
		confFilePath := viperConf.GetString(confFilePathKey)
		viperConf.SetConfigFile(confFilePath)
		baseLogger.Info().Str("file", confFilePath).Msg("Init Logger using a configuration file")
	}

	// try to read the configuration file
	err := viperConf.ReadInConfig()
	if err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			if defaultConfStr != "" {
				viperConf.ReadConfig(strings.NewReader(defaultConfStr))
			}
		default:
			baseLogger.Error().Err(err).Msg("Fail to read a logger's config file")
		}
	}

	return viperConf
}

func initLog() {

	// set timestamp format
	// there is a nice example in time/format.go
	// ANSIC       = "Mon Jan _2 15:04:05 2006"
	// UnixDate    = "Mon Jan _2 15:04:05 MST 2006"
	// RubyDate    = "Mon Jan 02 15:04:05 -0700 2006"
	// RFC822      = "02 Jan 06 15:04 MST"
	// RFC822Z     = "02 Jan 06 15:04 -0700" // RFC822 with numeric zone
	// RFC850      = "Monday, 02-Jan-06 15:04:05 MST"
	// RFC1123     = "Mon, 02 Jan 2006 15:04:05 MST"
	// RFC1123Z    = "Mon, 02 Jan 2006 15:04:05 -0700" // RFC1123 with numeric zone
	// RFC3339     = "2006-01-02T15:04:05Z07:00"
	// RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
	// Kitchen     = "3:04PM"
	// Stamp      = "Jan _2 15:04:05"
	// StampMilli = "Jan _2 15:04:05.000"
	// StampMicro = "Jan _2 15:04:05.000000"
	// StampNano  = "Jan _2 15:04:05.000000000"

	if viperConf.GetString("timefieldformat") != "" {
		zerolog.TimeFieldFormat = viperConf.GetString("timefieldformat")
	}

	out := os.Stderr
	outputName := viperConf.GetString("out")
	if outputName != "" {
		o, err := getOutput(outputName)
		if err == nil {
			out = o
			baseLogger = baseLogger.Output(out)
		} else {
			baseLogger.Warn().Err(err).Str("outputName", outputName).Msg("failed to open output writer. set to base out instead")
		}
	}

	// set output writer
	outputWriter := viperConf.GetString("formatter")
	if outputWriter != "" {
		switch strings.ToLower(outputWriter) {
		case "json":
			baseLogger = baseLogger.Output(out)
		case "console":
			baseLogger = baseLogger.Output(
				zerolog.ConsoleWriter{Out: colorable.NewColorable(out), NoColor: false, TimeFormat: zerolog.TimeFieldFormat})
		case "console_no_color":
			baseLogger = baseLogger.Output(
				zerolog.ConsoleWriter{Out: out, NoColor: true, TimeFormat: zerolog.TimeFieldFormat})
		default:
			baseLogger.Warn().Str("formatter", outputWriter).Msg("Invalid Message Formatter. Only allowed; console/console_no_color/json")
			baseLogger = baseLogger.Output(out)
		}
	}

	// set a caller print option
	if viperConf.GetBool("caller") {
		baseLogger = baseLogger.With().Caller().Logger()
	}

	// set a base log level
	level := viperConf.GetString("level")
	var zLevel zerolog.Level
	if level == "" {
		zLevel = zerolog.InfoLevel
	} else {
		var err error
		if zLevel, err = zerolog.ParseLevel(level); err != nil {
			baseLogger.Warn().Err(err).Msg("Fail to parse and set a default log level. set the level as info")
			zLevel = zerolog.InfoLevel
		}
	}

	// create logger by attaching a timestamp and setting a level
	baseLogger = baseLogger.With().Timestamp().Logger().Level(zLevel)
	baseLevel = zLevel
}

// NewLogger creates and returns new logger using a current setting.
// To classify and debug easily, this gets moduleName and
// makes all co-responding sources have a same tag 'module'
func NewLogger(moduleName string) *Logger {
	logInitLock.Lock()
	defer logInitLock.Unlock()

	// init logger only once at a start
	if !isLogInit {
		loadConfigFile()
		initLog()
		isLogInit = true
	}

	// create sub logger
	zLogger := baseLogger.With().Str("module", moduleName).Logger()

	// try to load sub config
	var zLevel zerolog.Level
	zLevel = baseLevel
	subViperConf := viperConf.Sub(moduleName)
	if subViperConf != nil {
		outputName := subViperConf.GetString("out")
		if outputName != "" {
			if out, err := getOutput(outputName); err == nil {
				zLogger = zLogger.Output(out)
			} else {
				baseLogger.Warn().Err(err).Str("outputName", outputName).Str("module", moduleName).Msg("failed to open output writer. set to base out instead")
			}
		}

		level := subViperConf.GetString("level")
		var err error

		if level != "" {
			if zLevel, err = zerolog.ParseLevel(level); err != nil {
				zLevel = zerolog.InfoLevel
			}

			// set sub logger's level
			zLogger = zLogger.Level(zLevel)
		}
	}

	return &Logger{
		Logger: &zLogger,
		name:   moduleName,
		level:  zLevel,
	}
}

var errEmptyName = errors.New("not really error. just placeholder")

// getOutput return prefer io.Writer matching to outName.
// outName is preserved keywords stdout and stderr, or file path
// it return error if name is empty or can't open file.
//
// Note: There can be argument about thread safe, because Writer of os.File is not
// guaranteed in thread safe golang runtime, also in stdout and stderr.
// But it looks practically less dangerous in POSIX compatible system.
//    https://stackoverflow.com/questions/29981050/concurrent-writing-to-a-file
func getOutput(outName string) (*os.File, error) {
	switch outName {
	case "":
		return nil, errEmptyName
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	default:
		out, err := os.OpenFile(outName, os.O_WRONLY|os.O_CREATE|os.O_APPEND|os.O_SYNC, 0644)
		if err != nil {
			return nil, err
		} else {
			return out, nil
		}
	}
}

// Default returns a defulat logger. this logger does not have a module name.
func Default() *Logger {
	logInitLock.Lock()
	defer logInitLock.Unlock()

	// init logger only once at a start
	if !isLogInit {
		initLog()
		isLogInit = true
	}

	return &Logger{
		Logger: &baseLogger,
		name:   "",
		level:  baseLevel,
	}
}

// IsDebugEnabled is used to check whether this logger's level is debug or not.
// This helps to prevent heavy computation to generate debug log statements.
func (logger *Logger) IsDebugEnabled() bool {
	return logger.level == zerolog.DebugLevel
}

// Level returns current logger level
func (logger *Logger) Level() string {
	return logger.level.String()
}

// Logger keeps configrations, and provides a funcs to print logs.
type Logger struct {
	*zerolog.Logger
	name  string
	level zerolog.Level
}
