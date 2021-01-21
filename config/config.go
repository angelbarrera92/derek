// Package config loads configuration from files and environment
// for Derek to use
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	derekSecretKeyFile = "derek-secret-key"
	privateKeyFile     = "derek-private-key"
)

// Config to run Derek
type Config struct {
	SecretKey       string
	PrivateKey      string
	ApplicationID   string
	DCOStatusChecks bool
}

// NewConfig populates configuration from known-locations and gives
// an error if configuration is missing from disk or environmental variables
func NewConfig() (Config, error) {
	config := Config{}

	// keyPath, pathErr := getSecretPath()
	// if pathErr != nil {
	// 	return config, pathErr
	// }

	secretKeyBytes, exists := os.LookupEnv("DEREK_SECRET_KEY")
	// ioutil.ReadFile(path.Join(keyPath, derekSecretKeyFile))

	if !exists {
		msg := errors.New("unable to read GitHub symmetrical secret")
		return config, msg
	}

	secretKeyBytes = getFirstLine(secretKeyBytes)
	config.SecretKey = string(secretKeyBytes)

	// privateKeyPath := path.Join(keyPath, privateKeyFile)

	keyBytes, exists := os.LookupEnv("DEREK_PRIVATE_KEY")
	// := ioutil.ReadFile(privateKeyPath)
	if !exists {
		return config, fmt.Errorf("unable to read private key")
	}

	config.PrivateKey = string(keyBytes)

	if val, ok := os.LookupEnv("APPLICATION_ID"); ok && len(val) > 0 {
		config.ApplicationID = val
	} else {
		return config, fmt.Errorf("APPLICATION_ID must be given")
	}

	if val, ok := os.LookupEnv("DCO_STATUS_CHECKS"); ok && len(val) > 0 {
		v, err := strconv.ParseBool(val)
		if err == nil {
			config.DCOStatusChecks = v
		}
	}

	// debug, _ := json.Marshal(config)
	// fmt.Printf("Config:\n%s\n", debug)

	return config, nil
}

func getSecretPath() (string, error) {
	secretPath := os.Getenv("SECRET_PATH")

	if len(secretPath) == 0 {
		return "", fmt.Errorf("secret_path env-var not set, this should be /var/openfaas/secrets or /run/secrets")
	}

	return secretPath, nil
}

func getFirstLine(secret string) string {
	stringSecret := secret
	if newLine := strings.Index(stringSecret, "\n"); newLine != -1 {
		secret = secret[:newLine]
	}
	return secret
}
