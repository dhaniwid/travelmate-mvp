package utils

import (
	"math"
)

// Coordinates represents a geographical point
type Coordinates struct {
	Lat float64
	Lng float64
}

// HaversineDistance calculates the distance between two points in kilometers
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Radius of Earth in kilometers

	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	lat1Rad := lat1 * (math.Pi / 180.0)
	lat2Rad := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// IsWithinRadius checks if a point is within a radius of a center point
func IsWithinRadius(centerLat, centerLon, pointLat, pointLon, radiusKm float64) bool {
	distance := HaversineDistance(centerLat, centerLon, pointLat, pointLon)
	return distance <= radiusKm
}
