package orbits

import (
	"math"
	"time"
)

const (
	// EarthMu is the standard gravitational parameter for Earth in km^3/s^2.
	EarthMu = 398600.4418
	twoPi   = 2 * math.Pi
)

// KeplerianElements represents classical orbital elements referenced to an epoch.
type KeplerianElements struct {
	SemiMajorAxis       float64   // kilometers
	Eccentricity        float64   // unitless, 0 <= e < 1
	Inclination         float64   // radians
	RAAN                float64   // radians
	ArgumentOfPeriapsis float64   // radians
	MeanAnomaly         float64   // radians at Epoch
	Epoch               time.Time // reference epoch
	Mu                  float64   // gravitational parameter, km^3/s^2
}

// MeanMotion returns the mean motion (rad/s) for the orbit.
func (k KeplerianElements) MeanMotion() float64 {
	mu := k.Mu
	if mu == 0 {
		mu = EarthMu
	}

	return math.Sqrt(mu / math.Pow(k.SemiMajorAxis, 3))
}

// Propagate advances the mean anomaly using a Keplerian two-body model by the provided duration.
func (k KeplerianElements) Propagate(dt time.Duration) KeplerianElements {
	propagated := k
	propagated.Epoch = k.Epoch.Add(dt)
	propagated.MeanAnomaly = normalizeAngle(k.MeanAnomaly + k.MeanMotion()*dt.Seconds())

	return propagated
}

// MeanAnomalyFromEccentric computes mean anomaly M from eccentric anomaly E.
func MeanAnomalyFromEccentric(eccentricAnomaly, eccentricity float64) float64 {
	if eccentricity == 0 {
		return normalizeAngle(eccentricAnomaly)
	}

	return normalizeAngle(eccentricAnomaly - eccentricity*math.Sin(eccentricAnomaly))
}

// TrueAnomalyFromEccentric converts an eccentric anomaly to the true anomaly.
func TrueAnomalyFromEccentric(eccentricAnomaly, eccentricity float64) float64 {
	if eccentricity == 0 {
		return normalizeAngle(eccentricAnomaly)
	}

	sinE := math.Sin(eccentricAnomaly)
	cosE := math.Cos(eccentricAnomaly)
	sqrtTerm := math.Sqrt(1 - eccentricity*eccentricity)

	numerator := sqrtTerm * sinE
	denominator := cosE - eccentricity

	return normalizeAngle(math.Atan2(numerator, denominator))
}

// EccentricAnomalyFromMean solves Kepler's equation for the eccentric anomaly using Newton-Raphson iteration.
func EccentricAnomalyFromMean(meanAnomaly, eccentricity float64) float64 {
	if eccentricity == 0 {
		return normalizeAngle(meanAnomaly)
	}

	M := normalizeAngle(meanAnomaly)
	E := initialGuess(M, eccentricity)
	for i := 0; i < 50; i++ {
		f := E - eccentricity*math.Sin(E) - M
		fp := 1 - eccentricity*math.Cos(E)
		delta := f / fp
		E -= delta

		if math.Abs(delta) < 1e-12 {
			break
		}
	}

	return normalizeAngle(E)
}

// TrueAnomalyFromMean converts mean anomaly directly to true anomaly.
func TrueAnomalyFromMean(meanAnomaly, eccentricity float64) float64 {
	return TrueAnomalyFromEccentric(EccentricAnomalyFromMean(meanAnomaly, eccentricity), eccentricity)
}

func normalizeAngle(angle float64) float64 {
	wrapped := math.Mod(angle, twoPi)
	if wrapped < 0 {
		wrapped += twoPi
	}
	return wrapped
}

func initialGuess(meanAnomaly, eccentricity float64) float64 {
	if eccentricity < 0.8 {
		return meanAnomaly
	}
	if meanAnomaly < math.Pi {
		return meanAnomaly + eccentricity/2
	}
	return meanAnomaly - eccentricity/2
}
