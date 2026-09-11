package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Approach adapted from https://github.com/carolynvs/stingoftheviper

const envPrefix = "cc2md"
const configFilename = "cc2md"

func initializeConfig(cmd *cobra.Command, args []string) error {
	v := viper.New()

	v.SetConfigName(configFilename)

	// Acceptable paths for config file
	v.AddConfigPath("/etc/cc2md")
	v.AddConfigPath("$HOME/.config/cc2md")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	bindFlags(cmd, v)

	return nil
}

func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			if err := cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val)); err != nil {
				fmt.Fprintf(os.Stderr, "cc2md: invalid config value for %q: %s\n", f.Name, err)
			}
		}
	})
}
