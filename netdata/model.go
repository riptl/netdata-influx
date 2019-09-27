package netdata

import "encoding/json"

type Response struct {
	API            int           `json:"api"`
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	FirstEntry     int64         `json:"first_entry"`
	LastEntry      int64         `json:"last_entry"`
	DimensionNames []string      `json:"dimension_names"`
	DimensionIDs   []string      `json:"dimensions_ids"`
	LatestValues   []json.Number `json:"latest_values"`
	ViewLatest     []json.Number `json:"view_latest"`
	Dimensions     int           `json:"dimensions"`
	Points         int           `json:"points"`
	Format         string        `json:"format"`
	Result         Result        `json:"result"`
	Min            json.Number   `json:"min"`
	Max            json.Number   `json:"max"`
}

type Result struct {
	Labels []string `json:"labels"`
	Data   [][]json.Number `json:"data"`
}
