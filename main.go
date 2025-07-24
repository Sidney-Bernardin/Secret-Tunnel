package main

import (
	"encoding/json"
	"flag"
	"io"
	"log/slog"
	"os"

	"github.com/goccy/go-yaml"
)

type Target struct {
	FullnameOverride string `json:"fullnameOverride"`
	Configmap        struct {
		Data struct {
			StadiumDeviceEndpoint  string `json:"STADIUM_DEVICE_ENDPOINT,omitempty"`
			StadiumDeviceApiToken  string `json:"STADIUM_DEVICE_API_TOKEN,omitempty"`
			StadiumDeviceUsername  string `json:"STADIUM_DEVICE_USERNAME,omitempty"`
			StadiumDevicePassword  string `json:"STADIUM_DEVICE_PASSWORD,omitempty"`
			StadiumDeviceApiKey    string `json:"STADIUM_DEVICE_API_KEY,omitempty"`
			StadiumDeviceAccountId string `json:"STADIUM_DEVICE_ACCOUNT_ID,omitempty"`
			StadiumDeviceId        string `json:"STADIUM_DEVICE_ID,omitempty"`
			StadiumDeviceType      string `json:"STADIUM_DEVICE_TYPE,omitempty"`
		} `json:"data"`
	} `json:"configmap"`
}

type Secret struct {
	Name    string `json:"name"`
	KVPairs string `json:"kvpairs"`
}

var (
	singleQuote = flag.Bool("single-quote", false, "Single or double quotes for strings.")
)

func main() {
	flag.Parse()

	var output struct {
		Secrets []Secret `json:"secrets"`
	}

	for _, path := range flag.Args() {
		log := slog.With("file", path)

		// Open path's file.
		file, err := os.Open(path)
		if err != nil {
			log.Error("Cannot open file", "err", err.Error())
			return
		}

		// Read the file.
		targetYAML, err := io.ReadAll(file)
		if err != nil {
			log.Error("Cannot read file", "err", err.Error())
			return
		}

		// YAML decode the target.
		var target Target
		if err := yaml.Unmarshal(targetYAML, &target); err != nil {
			log.Error("Cannot YAML decode file", "err", err.Error())
			return
		}

		// JSON encode the target's configmap data.
		targetConfigmapDataJSON, err := json.Marshal(target.Configmap.Data)
		if err != nil {
			log.Error("Cannot JSON encode configmap data", "err", err.Error())
			return
		}

		// Add a new secret to the output.
		output.Secrets = append(output.Secrets, Secret{
			Name:    target.FullnameOverride,
			KVPairs: string(targetConfigmapDataJSON),
		})
	}

	// YAML encode the output to stdout.
	if err := yaml.NewEncoder(os.Stdout, yaml.UseSingleQuote(*singleQuote)).Encode(output); err != nil {
		slog.Error("Cannot YAML encode output", "err", err.Error())
	}
}
