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

	awsBaseEndpoint       = os.Getenv("ST_AWS_BASE_ENDPOINT") // Primarily for local testing.
	awsDatabaseSecretName = os.Getenv("ST_AWS_DB_SECRET_NAME")
)

type (
	Sensor struct {
		FullnameOverride string `json:"fullnameOverride"`
		Configmap        struct {
			Data SensorConfigmapData `json:"data"`
		} `json:"configmap"`
		Secret struct {
			Data SensorSecretData `json:"data"`
		} `json:"secret"`
	}

	SensorConfigmapData struct {
		StadiumDeviceEndpoint   *string `json:"STADIUM_DEVICE_ENDPOINT,omitempty"`
		StadiumDeviceSensorUUID string  `yaml:"STADIUM_DEVICE_SENSOR_UUID" json:"-"`
		StadiumDeviceType       string  `json:"STADIUM_DEVICE_TYPE,omitempty"`
	}

	SensorSecretData struct {
		StadiumDeviceAPIToken  *string `json:"STADIUM_DEVICE_API_TOKEN,omitempty"`
		StadiumDeviceUsername  *string `json:"STADIUM_DEVICE_USERNAME,omitempty"`
		StadiumDevicePassword  *string `json:"STADIUM_DEVICE_PASSWORD,omitempty"`
		StadiumDeviceAPIKey    *string `json:"STADIUM_DEVICE_API_KEY,omitempty"`
		StadiumDeviceAccountID *string `json:"STADIUM_DEVICE_ACCOUNT_ID,omitempty"`
		StadiumDeviceID        *string `json:"STADIUM_DEVICE_ID,omitempty"`
	}
)

type (
	Output struct {
		Secrets []SensorSecret `json:"secrets"`
	}

	SensorSecret struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		KVPairs string `json:"kvpairs"`
	}
)

var secretsManagerClient *secretsmanager.Client

func main() {
	flag.Parse()
	ctx := context.Background()

	// Load AWS config and create a Secrets-Manager client.
	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		slog.Error("cannot load AWS config", "err", err.Error())
	}

	secretsManagerClient = secretsmanager.NewFromConfig(awsConfig,
		func(o *secretsmanager.Options) {
			if awsBaseEndpoint != "" {
				o.BaseEndpoint = aws.String(awsBaseEndpoint)
			}
		})

	// Get the UUIDs of only enabled sensors.
	enabledSensorUUIDs, err := getEnabledSensors(ctx)
	if err != nil {
		slog.Error("Cannot get enabled sensor UUIDs", "err", err.Error())
		return
	}

	output := Output{
		Secrets: make([]SensorSecret, 0, flag.NArg()),
	}

	// Create a sensor-secret from each sensor-file.
	for _, sensorFilePath := range flag.Args() {
		log := slog.With("sensor_file", sensorFilePath)

		// Decode the sensor file.
		sensor, err := decodeSensorFile(sensorFilePath)
		if err != nil {
			log.Error("Cannot decode sensor-file", "err", err.Error())
			return
		}

		// Skip this sensor if it's disabled.
		if !slices.Contains(enabledSensorUUIDs, sensor.Configmap.Data.StadiumDeviceSensorUUID) {
			continue
		}

		// Create the sensor-secret.
		sensorSecret, err := createSensorSecret(sensor)
		if err != nil {
			log.Error("Cannot create sensor-secret", "err", err.Error())
			return
		}

		output.Secrets = append(output.Secrets, *sensorSecret)
	}

	// Encode the output into stdout.
	if err := yaml.NewEncoder(os.Stdout, yaml.UseSingleQuote(*singleQuote)).Encode(output); err != nil {
		slog.Error("Cannot encode output", "err", err.Error())
		return
	}
}

// decodeSensorFile returns the content of a sensor-file.
func decodeSensorFile(sensorFilePath string) (*Sensor, error) {

	// Open the sensor-file.
	sensorFile, err := os.Open(sensorFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "cannot open file")
	}
	defer sensorFile.Close()

	// Read the sensor-file.
	sensorYAML, err := io.ReadAll(sensorFile)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read file")
	}

	// Decode the sensor-YAML.
	var sensor Sensor
	if err := yaml.Unmarshal(sensorYAML, &sensor); err != nil {
		return nil, errors.Wrap(err, "cannot decode sensor")
	}

	return &sensor, nil
}

// createSensorSecret returns a new sensor-secret.
func createSensorSecret(sensor *Sensor) (*SensorSecret, error) {

	// Encode the sensor's data objects into one JSON object.
	kvPairsJSON, err := json.Marshal(struct {
		SensorConfigmapData
		SensorSecretData
	}{sensor.Configmap.Data, sensor.Secret.Data})

	if err != nil {
		return nil, errors.Wrap(err, "cannot encode kvpairs")
	}

	return &SensorSecret{
		// Name:    sensor.FullnameOverride,
		Name:    sensor.Configmap.Data.StadiumDeviceSensorUUID,
		Type:    sensor.Configmap.Data.StadiumDeviceType,
		KVPairs: string(kvPairsJSON),
	}, nil
}

// getEnabledSensors returns the UUIDs of all enabled sensors from the database.
func getEnabledSensors(ctx context.Context) ([]string, error) {

	// Get the database-secret from Secrets-Manager.
	secretResult, err := secretsManagerClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &awsDatabaseSecretName})
	if err != nil {
		return nil, errors.Wrap(err, "cannot get database-secret")
	}

	// Decode the database-secret.
	var secret map[string]any
	if err := json.Unmarshal([]byte(*secretResult.SecretString), &secret); err != nil {
		return nil, errors.Wrap(err, "cannot decode database-secret")
	}

	pgURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		secret["PG_USERNAME"],
		secret["PG_PASSWORD"],
		secret["PG_HOST"],
		secret["PG_PORT"],
		secret["PG_DATABASE"])

	// Create connection to the database.
	conn, err := pgx.Connect(ctx, pgURL)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create postgres connection")
	}
	defer conn.Close(ctx)

	// Select all enabled sensors.
	rows, err := conn.Query(ctx, `
		SELECT sensor_uuid
		FROM collections.sensors
		WHERE enabled_flag = true
	`)

	if err != nil {
		return nil, errors.Wrap(err, "cannot select enabled sensors")
	}

	defer rows.Close()

	// Get each sensor's UUID.
	uuids := []string{}
	for rows.Next() {
		var uuid string
		if err := rows.Scan(&uuid); err != nil {
			return nil, errors.Wrap(err, "cannot scan database row")
		}
		uuids = append(uuids, uuid)
	}

	if rows.Err() != nil {
		return nil, errors.Wrap(err, "cannot scan database rows")
	}

	return uuids, nil
}
