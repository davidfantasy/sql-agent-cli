package credentials

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type HelperRequest struct {
	Version    int    `json:"version"`
	Action     string `json:"action"`
	Connection string `json:"connection"`
	Driver     string `json:"driver"`
}

type HelperCredentials struct {
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Database string `json:"database,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type HelperResponse struct {
	OK          bool              `json:"ok"`
	Credentials HelperCredentials `json:"credentials"`
}

func ParseHelperResponse(data []byte) (HelperResponse, error) {
	var response HelperResponse
	err := json.Unmarshal(data, &response)
	return response, err
}

func RunCredentialHelper(connectionName, driver, command string) (HelperCredentials, error) {
	request := HelperRequest{
		Version:    1,
		Action:     "resolve",
		Connection: connectionName,
		Driver:     driver,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return HelperCredentials{}, fmt.Errorf("marshal helper request: %w", err)
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return HelperCredentials{}, errors.New("credential helper command is empty")
	}

	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = bytes.NewReader(requestBody)

	stdout, err := cmd.Output()
	if err != nil {
		return HelperCredentials{}, fmt.Errorf("run credential helper: %w", err)
	}

	response, err := ParseHelperResponse(stdout)
	if err != nil {
		return HelperCredentials{}, fmt.Errorf("parse helper response: %w", err)
	}

	if !response.OK {
		return HelperCredentials{}, errors.New("credential helper returned ok=false")
	}

	return response.Credentials, nil
}
