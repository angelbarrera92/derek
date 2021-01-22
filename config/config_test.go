package config

import (
	"os"
	"testing"
)

func TestNewConfig_ValidSecretPath_WithApplicationID(t *testing.T) {
	privateWant := "private"
	secretWant := "secret"
	appIDWant := "321"

	os.Setenv("DEREK_SECRET_KEY", secretWant)
	os.Setenv("DEREK_PRIVATE_KEY", privateWant)
	os.Setenv("APPLICATION_ID", appIDWant)

	cfg, err := NewConfig()

	if err != nil {
		t.Errorf("%s", err.Error())
		t.Fail()
		return
	}

	if cfg.SecretKey != secretWant {
		t.Errorf("want %q, got %q", secretWant, cfg.SecretKey)
		t.Fail()
	}

	if cfg.PrivateKey != privateWant {
		t.Errorf("want %q, got %q", privateWant, cfg.PrivateKey)
		t.Fail()
	}

	if cfg.ApplicationID != appIDWant {
		t.Errorf("want %q, got %q", appIDWant, cfg.ApplicationID)
		t.Fail()
	}
}

func Test_getFirstLine(t *testing.T) {
	var exampleSecrets = []struct {
		secret       string
		expectedByte string
	}{
		{
			secret:       "New-line \n",
			expectedByte: "New-line ",
		},
		{
			secret: `Newline and text 
			`,
			expectedByte: "Newline and text ",
		},
		{
			secret:       `Example secret2 `,
			expectedByte: `Example secret2 `,
		},
		{
			secret:       "\n",
			expectedByte: "",
		},
		{
			secret:       "",
			expectedByte: "",
		},
	}
	for _, test := range exampleSecrets {

		t.Run(string(test.secret), func(t *testing.T) {
			stringNoLines := getFirstLine(test.secret)
			if test.expectedByte != string(stringNoLines) {
				t.Errorf("String after removal - wanted: \"%s\", got \"%s\"", test.expectedByte, test.secret)
			}
		})
	}
}
