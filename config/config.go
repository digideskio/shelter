package config

import (
	"encoding/json"
	"io/ioutil"
	"time"
)

var (
	ShelterConfig Config
)

type Config struct {
	BasePath    string
	LogFilename string

	Database struct {
		Name string
		URI  string
	}

	Scan struct {
		Enabled           bool
		NumberOfQueriers  int
		DomainsBufferSize int
		ErrorsBufferSize  int
		UDPMaxSize        uint16
		SaveAtOnce        int
		ConnectionRetries int

		Timeouts struct {
			DialSeconds  time.Duration
			ReadSeconds  time.Duration
			WriteSeconds time.Duration
		}

		VerificationIntervals struct {
			MaxOKDays              int
			MaxErrorDays           int
			MaxExpirationAlertDays int
		}
	}

	RESTServer struct {
		Enabled            bool
		LanguageConfigPath string

		TLS struct {
			CertificatePath string
			PrivateKeyPath  string
		}

		Listeners []struct {
			IP   string
			Port int
			TLS  bool
		}

		Secrets map[string]string
	}
}

func LoadConfig(path string) error {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytes, &ShelterConfig); err != nil {
		return err
	}

	return nil
}
