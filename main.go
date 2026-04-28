package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/goccy/go-yaml"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

var (
	singleQuote = flag.Bool("single-quote", false, "Use single/double quotes the output.")

	awsRegion                   = os.Getenv("SELECT_TUNNEL_AWS_REGION")
	awsPostgresPasswordSecretID = os.Getenv("SELECT_TUNNEL_AWS_SECRET_POSTGRES_PASSWORD")

	pgUsername = os.Getenv("SECRET_TUNNEL_POSTGRES_USERNAME")
	pgHost     = os.Getenv("SECRET_TUNNEL_POSTGRES_HOST")
	pgPort     = os.Getenv("SECRET_TUNNEL_POSTGRES_PORT")
	pgDatabase = os.Getenv("SECRET_TUNNEL_POSTGRES_DATABASE")
)

type (

	// Sensor represents an input YAML file.
	Sensor struct {
		FullnameOverride string `json:"fullnameOverride"`
		Configmap        struct {
			Data sensorConfigmapData `json:"data"`
		} `json:"configmap"`
		Secret struct {
			Data sensorSecretData `json:"data"`
		} `json:"secret"`
	}

	sensorConfigmapData struct {
		StadiumDeviceEndpoint   *string `json:"STADIUM_DEVICE_ENDPOINT,omitempty"`
		StadiumDeviceSensorUUID string  `yaml:"STADIUM_DEVICE_SENSOR_UUID" json:"-"`
		StadiumDeviceType       string  `json:"STADIUM_DEVICE_TYPE,omitempty"`
	}

	sensorSecretData struct {
		StadiumDeviceAPIToken  *string `json:"STADIUM_DEVICE_API_TOKEN,omitempty"`
		StadiumDeviceUsername  *string `json:"STADIUM_DEVICE_USERNAME,omitempty"`
		StadiumDevicePassword  *string `json:"STADIUM_DEVICE_PASSWORD,omitempty"`
		StadiumDeviceAPIKey    *string `json:"STADIUM_DEVICE_API_KEY,omitempty"`
		StadiumDeviceAccountID *string `json:"STADIUM_DEVICE_ACCOUNT_ID,omitempty"`
		StadiumDeviceID        *string `json:"STADIUM_DEVICE_ID,omitempty"`
	}
)

type (

	// Output represents the output YAML file.
	Output struct {
		Secrets []outputSecret `json:"secrets"`
	}

	outputSecret struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		KVPairs string `json:"kvpairs"`
	}
)

func main() {
	flag.Parse()
	ctx := context.Background()

	postgresPassword, err := getAWSSecrets(ctx)
	if err != nil {
		slog.Error("Cannot get AWS secrets", "err", err.Error())
		return
	}

	// Get enabled sensor UUIDs.
	enabledSensorUUIDs, err := getEnabledSensors(ctx, postgresPassword)
	if err != nil {
		slog.Error("Cannot get enabled sensor UUIDs", "err", err.Error())
		return
	}

	var output Output

	// Process each file.
	for _, filePath := range flag.Args() {
		log := slog.With("file", filePath)

		// Open the file.
		file, err := os.Open(filePath)
		if err != nil {
			log.Error("Cannot open file", "err", err.Error())
			return
		}

		// Read the file.
		sensorYAML, err := io.ReadAll(file)
		if err != nil {
			log.Error("Cannot read file", "err", err.Error())
			return
		}

		// YAML decode the file.
		var sensor Sensor
		if err := yaml.Unmarshal(sensorYAML, &sensor); err != nil {
			log.Error("Cannot YAML decode file", "err", err.Error())
			return
		}

		// Check if the sensor is enabled.
		if !slices.Contains(enabledSensorUUIDs, sensor.Configmap.Data.StadiumDeviceSensorUUID) {
			continue
		}

		// JSON encode the sensor's data objects.
		kvpairsJSON, err := json.Marshal(struct {
			sensorConfigmapData
			sensorSecretData
		}{sensor.Configmap.Data, sensor.Secret.Data})

		if err != nil {
			log.Error("Cannot JSON encode kvpairs", "err", err.Error())
			return
		}

		// Add a new secret to output.
		output.Secrets = append(output.Secrets, outputSecret{
			Name:    sensor.FullnameOverride,
			Type:    sensor.Configmap.Data.StadiumDeviceType,
			KVPairs: string(kvpairsJSON),
		})
	}

	// YAML encode output into stdout.
	if err := yaml.NewEncoder(os.Stdout, yaml.UseSingleQuote(*singleQuote)).Encode(output); err != nil {
		slog.Error("Cannot YAML encode output", "err", err.Error())
	}
}

// getEnabledSensors returns the UUIDs of all enabled sensors from the database.
func getEnabledSensors(ctx context.Context, postgresPassword string) ([]string, error) {
	pgURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", pgUsername, postgresPassword, pgHost, pgPort, pgDatabase)

	// Create connection to the database.
	conn, err := pgx.Connect(ctx, pgURL)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create postgres connection")
	}
	defer conn.Close(ctx)

	// Get UUIDs of all enabled rows.
	rows, err := conn.Query(ctx, `
		SELECT sensor_uuid
		FROM collections.sensors
		WHERE enabled_flag = true
	`)

	if err != nil {
		return nil, errors.Wrap(err, "cannot select enabled sensors")
	}

	defer rows.Close()

	// Add each row's UUID to a slice.
	uuids := []string{}
	for rows.Next() {
		var uuid string
		if err := rows.Scan(&uuid); err != nil {
			return nil, errors.Wrap(err, "cannot scan sensor")
		}
		uuids = append(uuids, uuid)
	}

	if rows.Err() != nil {
		return nil, errors.Wrap(err, "cannot scan sensors")
	}

	return uuids, nil
}

func getAWSSecrets(ctx context.Context) (postgresPassword string, err error) {

	// Load AWS configuration.
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(awsRegion))
	if err != nil {
		slog.Error("Cannot load AWS config", "err", err.Error())
		return
	}

	secrets := secretsmanager.NewFromConfig(cfg)

	// Get the postgres password.
	postgresPasswordResult, err := secrets.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(awsPostgresPasswordSecretID)})
	if err != nil {
		return "", errors.Wrap(err, "cannot get postgres password from AWS")
	}

	return *postgresPasswordResult.SecretString, nil
}
