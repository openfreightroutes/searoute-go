package eca

import (
	"github.com/openfreightroutes/searoute-data/eca"
	searoute "github.com/openfreightroutes/searoute-go"
)

var Zones []searoute.ECAZone

func init() {
	for _, z := range eca.Zones() {
		Zones = append(Zones, searoute.ECAZone{Name: z.Name, BBoxes: z.BBoxes})
	}
	searoute.RegisterECAZones(Zones)
}
