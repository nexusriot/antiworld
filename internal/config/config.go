package config

import (
	"encoding/json"
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/nexusriot/antiworld/internal/crypto"
)

const (
	filename = "config.json"
)

type Proxy struct {
	Address  string `json:"address"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Config struct {
	DownloadFolder string `json:"download_folder"`
	BaseUrl        string `json:"base_url"`
	Proxy          *Proxy `json:"proxy"`
}

func LoadConfiguration() *Config {
	var config Config
	configFile, err := os.Open(filename)
	defer configFile.Close()
	if err != nil {
		log.Fatalf("failed to open config file: %s", filename)
	}
	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	if err != nil {
		log.Fatalf("failed to parse config file: %s", filename)
	}
	if config.Proxy != nil && config.Proxy.Password != "" {
		decryptedPass, err := crypto.Decrypt(config.Proxy.Password)
		if err != nil {
			log.Fatalf("failed to decrypt password")
		}
		config.Proxy.Password = decryptedPass
	}
	log.Infof("config file loaded")
	return &config
}
