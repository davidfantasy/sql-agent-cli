package cli

import (
	"errors"

	"github.com/davidfantasy/sql-agent-cli/internal/config"
	"github.com/spf13/cobra"
)

func newConnectCommand() *cobra.Command {
	var (
		driver           string
		host             string
		port             int
		database         string
		username         string
		passwordEnv      string
		credentialHelper string
		readOnly         bool
	)

	cmd := &cobra.Command{
		Use:   "connect <name>",
		Short: "Store a named connection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if driver == "" {
				return errors.New("--driver is required")
			}
			if !supportedDriver(driver) {
				return errors.New("--driver must be one of: mysql, postgres")
			}
			if database == "" {
				return errors.New("--database is required")
			}

			return config.SaveConnection(config.Connection{
				Name:             args[0],
				Driver:           driver,
				Host:             host,
				Port:             port,
				Database:         database,
				Username:         username,
				PasswordEnv:      passwordEnv,
				CredentialHelper: credentialHelper,
				ReadOnly:         readOnly,
			})
		},
	}

	cmd.Flags().StringVar(&driver, "driver", "", "Database driver: mysql or postgres")
	cmd.Flags().StringVar(&host, "host", "", "Database host")
	cmd.Flags().IntVar(&port, "port", 0, "Database port")
	cmd.Flags().StringVar(&database, "database", "", "Database name")
	cmd.Flags().StringVar(&username, "username", "", "Database username")
	cmd.Flags().StringVar(&passwordEnv, "password-env", "", "Environment variable containing the password")
	cmd.Flags().StringVar(&credentialHelper, "credential-helper", "", "External helper command for credentials")
	cmd.Flags().BoolVar(&readOnly, "read-only", false, "Mark connection as read-only")

	return cmd
}
