package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Client management commands",
}

var bindCmd = &cobra.Command{
	Use:   "bind [token]",
	Short: "Bind this client to a Hub account using a token",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		token := args[0]

		fmt.Printf("Saving bind token: %s\n", token)

		// Set token in config
		viper.Set("bind_token", token)

		// If no config file was loaded, set a default
		if viper.ConfigFileUsed() == "" {
			viper.SetConfigFile("config.yaml")
		}

		// Persist config
		if err := viper.WriteConfig(); err != nil {
			// If file not found (or not created yet), WriteConfig might fail.
			// Check if we need to create it.
			if _, ok := err.(viper.ConfigFileNotFoundError); ok || os.IsNotExist(err) || err.Error() == "missing configuration for 'configPath'" {
				if err := viper.SafeWriteConfig(); err != nil {
					// Fallback to WriteConfigAs just to be sure.
					if err := viper.WriteConfigAs("config.yaml"); err != nil {
						fmt.Printf("Error writing config: %v\n", err)
						os.Exit(1)
					}
				}
			} else {
				// Just in case, try WriteConfigAs
				if err := viper.WriteConfigAs("config.yaml"); err != nil {
					fmt.Printf("Error updating config file: %v\n", err)
					os.Exit(1)
				}
			}
		}

		fmt.Println("Bind token saved successfully.")
		fmt.Println("Please restart the application to complete the binding process.")
	},
}

func init() {
	rootCmd.AddCommand(clientCmd)
	clientCmd.AddCommand(bindCmd)
}
