package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log" // See https://github.com/spf13/viper/issues/1152
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/automaxprocs/maxprocs"

	"github.com/trivago/go-bootstrap/logging"
)

const (
	// ArgLogLevel is the command line argument and viper key to set the
	// loglevel of the application.
	ArgLogLevel = "loglevel"
)

var (
	// SkipArgs defines the number of command line arguments to skip
	// during parameter parsing in the InitConfig function.
	SkipArgs = 0

	// FlagsName contains the executable name to display when using --help
	FlagsName = os.Args[0]

	// DefaultLogLevel defines the log level the system should run at by default
	DefaultLogLevel = "debug"

	// ExtraArgs contains the commandline flags left after parsing in the
	// InitConfig function.
	ExtraArgs = []string{}
)

// Read retrieves the configuration values from the environment, command line
// or config file. The envPrefix is used to prefix environment variables.
// ConfigFile can be empty to disable reading from a config file or must be of a
// fileformat supported by viper (e.g. ".yaml").
// Use viper.SetDefault to set default values for configuration parameters.
func Read(envPrefix, configFile string) {
	// Default values
	viper.SetDefault(ArgLogLevel, DefaultLogLevel)

	// Allow reading from config file
	if len(configFile) > 0 {
		directory := filepath.Dir(configFile)
		fileType := filepath.Ext(configFile)
		fileName := strings.TrimSuffix(filepath.Base(configFile), fileType)

		viper.SetConfigName(fileName)
		viper.SetConfigType(fileType)

		if len(directory) > 0 {
			viper.AddConfigPath(directory)
		} else {
			viper.AddConfigPath(".")
		}

		if err := viper.ReadInConfig(); err != nil {
			// Make sure this is logged _after_ the logging has been set up
			defer log.Info().Err(err).Msgf("Failed to read config file %s.", configFile)
		}
	}

	// Allow reading from environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()

	// Allow reading from command line flags
	err := viperAutomaticFlags()

	// Setup global loglevel
	logging.SetLogLevel(viper.GetString(ArgLogLevel))

	if err != nil {
		log.Error().Err(err).Msg("Failed to process command line arguments.")
	}

	// Make application cgroups aware
	// Needs to happen after the logger has been set up.
	_, err = maxprocs.Set(maxprocs.Logger(func(format string, a ...interface{}) {
		log.Info().Msgf(format, a...)
	}))

	if err != nil {
		log.Error().Err(err).Msg("Failed to configure maxprocs to match container CPU quota.")
	}
}

// viperAutomaticFlags converts all keys with a default value into command line
// flags. Each flag supports a shorthand form, using the first character. If
// two flags have the same first character, the first flag will have a short
// form, the second one will not.
func viperAutomaticFlags() error {
	usedShorts := map[string]struct{}{}
	getShort := func(k string) string {
		short := k[0:1]
		if _, taken := usedShorts[short]; !taken {
			usedShorts[short] = struct{}{}
			return short
		}
		return ""
	}

	flagSet := pflag.NewFlagSet(FlagsName, pflag.ExitOnError)
	flagSet.SetOutput(os.Stdout)

	for _, key := range viper.AllKeys() {
		switch v := viper.Get(key).(type) {
		case bool:
			flagSet.BoolP(key, getShort(key), v, "")

		case int:
			flagSet.IntP(key, getShort(key), v, "")

		case string:
			flagSet.StringP(key, getShort(key), v, "")

		case []string:
			flagSet.StringArrayP(key, getShort(key), v, "")

		case time.Duration:
			flagSet.DurationP(key, getShort(key), v, "")
		}

		log.Debug().Msgf("%s = %v", key, viper.Get(key))
	}

	defer func() {
		ExtraArgs = flagSet.Args()
	}()

	if err := flagSet.Parse(os.Args[1+SkipArgs:]); err != nil {
		return err
	}

	if err := viper.BindPFlags(flagSet); err != nil {
		return err
	}

	return nil
}
