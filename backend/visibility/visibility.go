package visibility

import "math"

// Vector3 represents a 3D position in an Earth-centered frame (kilometers).
type Vector3 struct {
	X float64
	Y float64
	Z float64
}

// EarthRadius is the mean Earth radius in kilometers.
const EarthRadius = 6371.0

// SlantRange returns the straight-line distance between two positions.
func SlantRange(a, b Vector3) float64 {
	return norm(sub(b, a))
}

// Elevation computes the elevation angle (radians) of a satellite relative to a ground point.
// A positive elevation indicates the satellite is above the local horizon.
func Elevation(ground, satellite Vector3) float64 {
	toSat := sub(satellite, ground)
	groundHat := scale(ground, 1.0/norm(ground))

	return math.Asin(dot(toSat, groundHat) / norm(toSat))
}

// MeetsElevationMask returns true when the satellite is above the provided elevation mask (radians).
func MeetsElevationMask(ground, satellite Vector3, mask float64) bool {
	return Elevation(ground, satellite) >= mask
}

// GroundToSatelliteVisible returns true when a ground point has line of sight to a satellite.
// Visibility requires clearing the Earth limb and satisfying the provided elevation mask (radians).
func GroundToSatelliteVisible(ground, satellite Vector3, elevationMask float64) bool {
	if !MeetsElevationMask(ground, satellite, elevationMask) {
		return false
	}

	return !segmentIntersectsEarth(ground, satellite, EarthRadius)
}

// SatelliteToSatelliteVisible returns true when the segment between two satellites does not intersect Earth.
func SatelliteToSatelliteVisible(a, b Vector3) bool {
	return !segmentIntersectsEarth(a, b, EarthRadius)
}

func segmentIntersectsEarth(p0, p1 Vector3, radius float64) bool {
	direction := sub(p1, p0)
	a := dot(direction, direction)
	b := 2 * dot(p0, direction)
	c := dot(p0, p0) - radius*radius

	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return false
	}

	sqrtD := math.Sqrt(discriminant)
	denom := 2 * a
	t1 := (-b - sqrtD) / denom
	t2 := (-b + sqrtD) / denom

	const epsilon = 1e-9
	return (t1 > epsilon && t1 < 1-epsilon) || (t2 > epsilon && t2 < 1-epsilon)
}

func dot(a, b Vector3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func norm(v Vector3) float64 {
	return math.Sqrt(dot(v, v))
}

func sub(a, b Vector3) Vector3 {
	return Vector3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}

func scale(v Vector3, factor float64) Vector3 {
	return Vector3{X: v.X * factor, Y: v.Y * factor, Z: v.Z * factor}
}
