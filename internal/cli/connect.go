package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/davidfantasy/sql-agent-cli/internal/config"
	"github.com/davidfantasy/sql-agent-cli/internal/credentials"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	openConnectionForValidation = validateConnection
	openWizardTerminal          = func() (*os.File, error) { return os.OpenFile("/dev/tty", os.O_RDWR, 0) }
	readPasswordInput           = term.ReadPassword
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
		wizard           bool
	)

	cmd := &cobra.Command{
		Use:   "connect <name>",
		Short: "Create, verify, and store a named connection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			connection := config.Connection{
				Name:             args[0],
				Driver:           driver,
				Host:             host,
				Port:             port,
				Database:         database,
				Username:         username,
				PasswordEnv:      passwordEnv,
				CredentialHelper: credentialHelper,
				ReadOnly:         readOnly,
			}

			if wizard {
				var err error
				connection, err = promptForConnection(connection)
				if err != nil {
					return err
				}
			} else {
				if err := validateConnectFlags(connection); err != nil {
					return err
				}
			}

			applyDefaultPort(&connection)

			if err := openConnectionForValidation(connection); err != nil {
				return err
			}

			return config.SaveConnection(connection)
		},
	}

	cmd.Flags().StringVar(&driver, "driver", "", "Database driver: mysql or postgres")
	cmd.Flags().StringVar(&host, "host", "", "Database host")
	cmd.Flags().IntVar(&port, "port", 0, "Database port")
	cmd.Flags().StringVar(&database, "database", "", "Database name")
	cmd.Flags().StringVar(&username, "username", "", "Database username")
	cmd.Flags().StringVar(&passwordEnv, "password-env", "", "Environment variable containing the password (for example: export DB_PASSWORD=... && sql-agent connect ... --password-env DB_PASSWORD)")
	cmd.Flags().StringVar(&credentialHelper, "credential-helper", "", "External helper command for credentials")
	cmd.Flags().BoolVar(&readOnly, "read-only", false, "Mark connection as read-only")
	cmd.Flags().BoolVar(&wizard, "wizard", false, "Prompt for connection fields in the local terminal and hide password input")
	cmd.Flags().BoolVarP(&wizard, "interactive", "i", false, "Alias for --wizard")

	return cmd
}

func validateConnectFlags(connection config.Connection) error {
	if connection.Driver == "" {
		return errors.New("--driver is required")
	}
	if !supportedDriver(connection.Driver) {
		return errors.New("--driver must be one of: mysql, postgres")
	}
	if connection.Database == "" {
		return errors.New("--database is required")
	}
	if connection.PasswordEnv == "" && connection.CredentialHelper == "" {
		return errors.New("non-interactive connect requires --password-env or --credential-helper; if you need to enter a password locally, use --wizard")
	}
	return nil
}

func applyDefaultPort(connection *config.Connection) {
	if connection.Port != 0 {
		return
	}

	switch strings.ToLower(connection.Driver) {
	case "mysql":
		connection.Port = 3306
	case "postgres":
		connection.Port = 5432
	}
}

func validateConnection(connection config.Connection) error {
	resolved, err := credentials.Resolve(connection, func(cfg config.Connection) (credentials.HelperCredentials, error) {
		return credentials.RunCredentialHelper(cfg.Name, cfg.Driver, cfg.CredentialHelper)
	})
	if err != nil {
		return err
	}

	driver, err := openDriver(resolved)
	if err != nil {
		return err
	}
	defer driver.Close()

	if err := driver.Ping(); err != nil {
		return fmt.Errorf("connection verification failed: %w", err)
	}

	return nil
}

func promptForConnection(connection config.Connection) (config.Connection, error) {
	tty, err := openWizardTerminal()
	if err != nil {
		return config.Connection{}, fmt.Errorf("open terminal for wizard: %w", err)
	}
	defer tty.Close()

	scanner := bufio.NewScanner(tty)

	if connection.Driver, err = promptString(tty, scanner, "Driver (mysql/postgres)", connection.Driver); err != nil {
		return config.Connection{}, err
	}
	if !supportedDriver(connection.Driver) {
		return config.Connection{}, errors.New("--driver must be one of: mysql, postgres")
	}
	applyDefaultPort(&connection)
	if connection.Host, err = promptString(tty, scanner, "Host", defaultString(connection.Host, "127.0.0.1")); err != nil {
		return config.Connection{}, err
	}
	portText, err := promptString(tty, scanner, "Port", strconv.Itoa(connection.Port))
	if err != nil {
		return config.Connection{}, err
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return config.Connection{}, fmt.Errorf("invalid port %q", portText)
	}
	connection.Port = port
	if connection.Database, err = promptString(tty, scanner, "Database", connection.Database); err != nil {
		return config.Connection{}, err
	}
	if connection.Database == "" {
		return config.Connection{}, errors.New("--database is required")
	}
	if connection.Username, err = promptString(tty, scanner, "Username", connection.Username); err != nil {
		return config.Connection{}, err
	}
	password, err := promptPassword(tty, "Password")
	if err != nil {
		return config.Connection{}, err
	}
	connection.PasswordEnv = ""
	connection.CredentialHelper = inlineCredentialHelperCommand(password)
	readOnlyText, err := promptString(tty, scanner, "Read-only (y/N)", boolDefault(connection.ReadOnly))
	if err != nil {
		return config.Connection{}, err
	}
	connection.ReadOnly = parseYes(readOnlyText)

	return connection, nil
}

func promptString(tty *os.File, scanner *bufio.Scanner, label, defaultValue string) (string, error) {
	if defaultValue != "" {
		if _, err := fmt.Fprintf(tty, "%s [%s]: ", label, defaultValue); err != nil {
			return "", err
		}
	} else {
		if _, err := fmt.Fprintf(tty, "%s: ", label); err != nil {
			return "", err
		}
	}

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", err
		}
		return "", io.EOF
	}

	text := strings.TrimSpace(scanner.Text())
	if text == "" {
		return defaultValue, nil
	}
	return text, nil
}

func promptPassword(tty *os.File, label string) (string, error) {
	if _, err := fmt.Fprintf(tty, "%s: ", label); err != nil {
		return "", err
	}
	secret, err := readPasswordInput(int(tty.Fd()))
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	if _, err := fmt.Fprintln(tty); err != nil {
		return "", err
	}
	if len(secret) == 0 {
		return "", errors.New("password is required in wizard mode")
	}
	return string(secret), nil
}

func inlineCredentialHelperCommand(password string) string {
	return "sql-agent-inline://" + password
}

func defaultString(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func boolDefault(value bool) string {
	if value {
		return "y"
	}
	return "n"
}

func parseYes(value string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	return trimmed == "y" || trimmed == "yes"
}
