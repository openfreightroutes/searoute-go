package searoute

import "encoding/json"

type Point struct {
	Lon, Lat float64
}

type Route struct {
	Coordinates       [][2]float64
	MultiCoordinates  [][][2]float64
	Length            float64
	Units             Units
	BBox              [4]float64
	GreatCircleLength float64
	DetourRatio       float64
	OriginSnapKm      float64
	DestinationSnapKm float64
	DurationHours     *float64
	Passages          []string
	ECAKm             *float64
	ECAFraction       *float64
	CO2eTonnes        *float64
	Variant           string
}

func (r *Route) MarshalJSON() ([]byte, error) {
	type properties struct {
		Length            float64    `json:"length"`
		Units             Units      `json:"units"`
		BBox              [4]float64 `json:"bbox"`
		GreatCircleLength float64    `json:"greatCircleLength"`
		DetourRatio       float64    `json:"detourRatio"`
		OriginSnapKm      float64    `json:"originSnapKm"`
		DestinationSnapKm float64    `json:"destinationSnapKm"`
		DurationHours     *float64   `json:"durationHours,omitempty"`
		Passages          *[]string  `json:"passages,omitempty"`
		ECAKm             *float64   `json:"ecaKm,omitempty"`
		ECAFraction       *float64   `json:"ecaFraction,omitempty"`
		CO2eTonnes        *float64   `json:"co2eTonnes,omitempty"`
		Variant           string     `json:"variant,omitempty"`
	}
	type geometry struct {
		Type        string `json:"type"`
		Coordinates any    `json:"coordinates"`
	}
	type feature struct {
		Type       string     `json:"type"`
		Properties properties `json:"properties"`
		Geometry   geometry   `json:"geometry"`
	}

	// A requested but empty passage list must still marshal as [], which omitempty on a plain slice would drop.
	var passages *[]string
	if r.Passages != nil {
		passages = &r.Passages
	}
	g := geometry{"LineString", r.Coordinates}
	if r.MultiCoordinates != nil {
		g = geometry{"MultiLineString", r.MultiCoordinates}
	}
	return json.Marshal(feature{
		Type: "Feature",
		Properties: properties{
			Length:            r.Length,
			Units:             r.Units,
			BBox:              r.BBox,
			GreatCircleLength: r.GreatCircleLength,
			DetourRatio:       r.DetourRatio,
			OriginSnapKm:      r.OriginSnapKm,
			DestinationSnapKm: r.DestinationSnapKm,
			DurationHours:     r.DurationHours,
			Passages:          passages,
			ECAKm:             r.ECAKm,
			ECAFraction:       r.ECAFraction,
			CO2eTonnes:        r.CO2eTonnes,
			Variant:           r.Variant,
		},
		Geometry: g,
	})
}
