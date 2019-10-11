package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	"github.com/spf13/viper"
	"github.com/terorie/netdata-influx/netdata"
)

var influx client.Client

const (
	ConfLogTimestamps = "log_timestamps"
	ConfInfluxAddr = "influx_addr"
	ConfInfluxDB = "influx_db"
	ConfRefresh = "refresh"
	ConfNetdata = "netdata_api"
	ConfHostTag = "host_tag"
	ConfCharts = "charts"
	ConfPoints = "points"
)

func main() {
	viper.SetDefault(ConfRefresh, 10 * time.Second)
	viper.SetDefault(ConfLogTimestamps, true)
	viper.SetDefault(ConfPoints, 0)
	viper.SetDefault(ConfCharts, []string{
		"system.cpu",
		"system.net",
		"system.pgpgio",
	})
	viper.SetEnvPrefix("ni")
	viper.AutomaticEnv()
	viper.SetDefault(ConfHostTag, viper.GetString(ConfNetdata))

	if !viper.GetBool(ConfLogTimestamps) {
		log.SetFlags(0)
	}
	if viper.GetString(ConfInfluxAddr) == "" {
		log.Fatal("$NI_INFLUX_ADDR not set")
	}
	if viper.GetString(ConfInfluxDB) == "" {
		log.Fatal("$NI_INFLUX_DB not set")
	}
	if viper.GetString(ConfNetdata) == "" {
		log.Fatal("$NI_NETDATA_API not set")
	}

	var err error
	influx, err = client.NewHTTPClient(client.HTTPConfig{
		Addr: viper.GetString(ConfInfluxAddr),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer closeInflux()

	for range time.Tick(viper.GetDuration(ConfRefresh)) {
		charts := viper.GetStringSlice(ConfCharts)
		for _, chart := range charts {
			res, err := getChart(chart)
			if err != nil {
				log.Printf("Failed to get chart %s: %s", chart, err)
				continue
			}
			if err := pushChart(res); err != nil {
				log.Printf("Failed to push chart %s: %s", chart, err)
			}
		}
	}
}

func getChart(chart string) (*netdata.Response, error) {
	builder := netdata.RequestBuilder{
		BaseURL: viper.GetString(ConfNetdata),
		Chart:   chart,
		Points:  viper.GetInt(ConfPoints),
		After:   -viper.GetInt(ConfPoints),
	}
	req, err := builder.Build()
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %s", res.Status)
	}
	dec := json.NewDecoder(res.Body)
	ndRes := new(netdata.Response)
	err = dec.Decode(&ndRes)
	if err != nil {
		return nil, err
	}
	if ndRes.API != 1 {
		return nil, fmt.Errorf("unexpected data at %s", req.URL.String())
	}
	return ndRes, err
}

func pushChart(res *netdata.Response) error {
	// Get number of columns
	nCols := len(res.Result.Labels)
	if nCols == 0 {
		return fmt.Errorf("no columns in result")
	}

	// Find index of time field
	timeIndex := -1
	for i, label := range res.Result.Labels {
		if label == "time" {
			timeIndex = i
		}
	}
	if timeIndex < 0 {
		return fmt.Errorf("no time column in result")
	}

	// Create batch
	batch, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  viper.GetString(ConfInfluxDB),
		Precision: "s",
	})
	if err != nil {
		return err
	}

	host := viper.GetString(ConfHostTag)

	// Fill batch with data
	for rowNr, row := range res.Result.Data {
		if len(row) != nCols {
			return fmt.Errorf("malformed row")
		}

		// Read timestamp from "time" column
		unix, err := row[timeIndex].Int64()
		if err != nil { return err }
		timestamp := time.Unix(unix, 0)

		// Create a point for each dimension
		for i, col := range row {
			if i == timeIndex {
				continue
			}
			var val float64
			if col == "" {
				val = 0
			} else {
				val, err = col.Float64()
				if err != nil {
					log.Printf("Chart %s: invalid data value at row %d: %s", res.ID, rowNr, err)
					continue
				}
			}
			tags := map[string]string{
				"host": host,
				"dimension": res.Result.Labels[i],
			}
			fields := map[string]interface{}{
				"value": val,
			}
			point, err := client.NewPoint(res.ID, tags, fields, timestamp)
			if err != nil { return err }
			batch.AddPoint(point)
		}
	}

	// Write batch
	return influx.Write(batch)
}

func closeInflux() {
	if err := influx.Close(); err != nil {
		log.Println("Error closing influx client", err)
	}
}
