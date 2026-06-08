package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/masterdatamate/masterdatamate/examples/templates/generated/constants"
	_ "modernc.org/sqlite"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var jsonDir string
	var schemaPath string
	var dbPath string
	var providerID string
	var dayText string
	flag.StringVar(&jsonDir, "json-dir", "runtime-json", "directory produced by masterdatamate export --format json")
	flag.StringVar(&schemaPath, "schema", "generated/sql/schema.sql", "SQL file produced by masterdatamate generate")
	flag.StringVar(&dbPath, "db", ":memory:", "SQLite database path")
	flag.StringVar(&providerID, "provider", "", "weather provider ID; defaults to the first enabled provider")
	flag.StringVar(&dayText, "day", "2026-06-08", "observation day in YYYY-MM-DD")
	flag.Parse()

	day, err := time.Parse("2006-01-02", dayText)
	if err != nil {
		return fmt.Errorf("parse --day: %w", err)
	}

	regions, stations, metrics, providers, err := loadRuntimeData(jsonDir)
	if err != nil {
		return err
	}
	provider, err := selectProvider(providers, providerID)
	if err != nil {
		return err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := applySchema(db, schemaPath); err != nil {
		return err
	}
	if err := seedMasterTables(context.Background(), db, regions, stations); err != nil {
		return err
	}

	fetcher := deterministicFetcher{day: day}
	if err := constants.RecordDailyWeather(context.Background(), db, fetcher, provider, stations, metrics); err != nil {
		return err
	}

	observationCount, err := countRows(db, "weather_observation")
	if err != nil {
		return err
	}
	stationCount, err := countRows(db, "weather_station")
	if err != nil {
		return err
	}

	fmt.Printf("Weather demo complete: %d region(s), %d station(s), %d metric(s), %d observation(s) written using %s.\n",
		len(regions), stationCount, len(metrics), observationCount, provider.ProviderID)
	return nil
}

func loadRuntimeData(jsonDir string) ([]constants.Region, []constants.WeatherStation, []constants.WeatherMetric, []constants.WeatherProvider, error) {
	var regions []constants.Region
	var stations []constants.WeatherStation
	var metrics []constants.WeatherMetric
	var providers []constants.WeatherProvider
	if err := readJSONFile(filepath.Join(jsonDir, "region.json"), &regions); err != nil {
		return nil, nil, nil, nil, err
	}
	if err := readJSONFile(filepath.Join(jsonDir, "weather_station.json"), &stations); err != nil {
		return nil, nil, nil, nil, err
	}
	if err := readJSONFile(filepath.Join(jsonDir, "weather_metric.json"), &metrics); err != nil {
		return nil, nil, nil, nil, err
	}
	if err := readJSONFile(filepath.Join(jsonDir, "weather_provider.json"), &providers); err != nil {
		return nil, nil, nil, nil, err
	}
	return regions, stations, metrics, providers, nil
}

func readJSONFile(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

func selectProvider(providers []constants.WeatherProvider, providerID string) (constants.WeatherProvider, error) {
	for _, provider := range providers {
		if !provider.Enabled {
			continue
		}
		if providerID == "" || provider.ProviderID == providerID {
			return provider, nil
		}
	}
	if providerID != "" {
		return constants.WeatherProvider{}, constants.NewWeatherError(constants.ErrorProviderUnavailable)
	}
	return constants.WeatherProvider{}, fmt.Errorf("no enabled weather provider found")
}

func applySchema(db *sql.DB, schemaPath string) error {
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read generated SQL %s: %w", schemaPath, err)
	}
	for _, stmt := range splitSQLStatements(string(data)) {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("apply generated SQL: %w\n%s", err, stmt)
		}
	}
	return nil
}

func splitSQLStatements(script string) []string {
	lines := strings.Split(script, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		cleaned = append(cleaned, line)
	}
	parts := strings.Split(strings.Join(cleaned, "\n"), ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		if stmt != "" {
			statements = append(statements, stmt)
		}
	}
	return statements
}

func seedMasterTables(ctx context.Context, db *sql.DB, regions []constants.Region, stations []constants.WeatherStation) error {
	for _, region := range regions {
		if _, err := db.ExecContext(ctx, `
INSERT INTO region (region_id, display_name, timezone, latitude, longitude)
VALUES (?, ?, ?, ?, ?)`,
			region.RegionID, region.DisplayName, region.Timezone, region.Latitude, region.Longitude,
		); err != nil {
			return err
		}
	}
	for _, station := range stations {
		if _, err := db.ExecContext(ctx, `
INSERT INTO weather_station (
  station_id, region_id, display_name, provider_station_id, latitude, longitude, enabled
) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			station.StationID, station.RegionID, station.DisplayName, station.ProviderStationID,
			station.Latitude, station.Longitude, station.Enabled,
		); err != nil {
			return err
		}
	}
	return nil
}

func countRows(db *sql.DB, table string) (int, error) {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

type deterministicFetcher struct {
	day time.Time
}

func (f deterministicFetcher) FetchDaily(ctx context.Context, station constants.WeatherStation, provider constants.WeatherProvider, metrics []constants.WeatherMetric) ([]constants.Observation, error) {
	observedAt := f.day.Add(9 * time.Hour)
	observations := make([]constants.Observation, 0, len(metrics))
	for _, metric := range metrics {
		observations = append(observations, constants.Observation{
			StationID:  station.StationID,
			MetricID:   metric.MetricID,
			ObservedAt: observedAt,
			Value:      deterministicValue(station.StationID, metric.MetricID),
			Unit:       metric.Unit,
			ProviderID: provider.ProviderID,
		})
	}
	return observations, nil
}

func deterministicValue(stationID string, metricID string) float64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(stationID + ":" + metricID))
	base := float64(h.Sum32()%1000) / 10
	switch metricID {
	case "temperature":
		return 5 + base/4
	case "humidity":
		return 40 + float64(int(base)%60)
	case "rainfall":
		return base / 20
	default:
		return base
	}
}
