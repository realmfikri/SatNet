package orbits

import (
	"math"
	"testing"
	"time"
)

func TestCircularAnomalies(t *testing.T) {
	mean := 1.2
	eccentricity := 0.0

	eccentric := EccentricAnomalyFromMean(mean, eccentricity)
	if eccentric != mean {
		t.Fatalf("expected eccentric anomaly to equal mean anomaly for circular orbit, got %v", eccentric)
	}

	trueAnomaly := TrueAnomalyFromEccentric(eccentric, eccentricity)
	if trueAnomaly != mean {
		t.Fatalf("expected true anomaly to equal mean anomaly for circular orbit, got %v", trueAnomaly)
	}
}

func TestEllipticalConversions(t *testing.T) {
	mean := 2.4
	eccentricity := 0.3

	eccentric := EccentricAnomalyFromMean(mean, eccentricity)
	computedMean := MeanAnomalyFromEccentric(eccentric, eccentricity)
	if math.Abs(computedMean-mean) > 1e-12 {
		t.Fatalf("mean anomaly mismatch: expected %v, got %v", mean, computedMean)
	}

	trueAnomaly := TrueAnomalyFromEccentric(eccentric, eccentricity)
	if trueAnomaly <= 0 || trueAnomaly >= twoPi {
		t.Fatalf("true anomaly should wrap into [0, 2pi), got %v", trueAnomaly)
	}
}

func TestPropagationUpdatesMeanAnomaly(t *testing.T) {
	elements := KeplerianElements{
		SemiMajorAxis: 7000,
		Eccentricity:  0,
		MeanAnomaly:   0,
		Epoch:         time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	dt := 10 * time.Minute
	propagated := elements.Propagate(dt)

	expected := elements.MeanMotion() * dt.Seconds()
	if math.Abs(propagated.MeanAnomaly-expected) > 1e-9 {
		t.Fatalf("propagated mean anomaly mismatch: expected %v, got %v", expected, propagated.MeanAnomaly)
	}

	if !propagated.Epoch.Equal(elements.Epoch.Add(dt)) {
		t.Fatalf("epoch was not advanced; expected %v got %v", elements.Epoch.Add(dt), propagated.Epoch)
	}
}

func TestGeostationaryOrbitPeriod(t *testing.T) {
	elements := KeplerianElements{
		SemiMajorAxis: 42164, // kilometers
		Eccentricity:  0.001,
		MeanAnomaly:   1.0,
		Epoch:         time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	dt := 86164 * time.Second // one sidereal day
	propagated := elements.Propagate(dt)

	// GEO should complete roughly one revolution per sidereal day.
	revolution := normalizeAngle(propagated.MeanAnomaly - elements.MeanAnomaly)
	// Normalized revolution close to 0 or 2pi both indicate a full cycle.
	deviation := math.Min(revolution, math.Abs(revolution-twoPi))
	if deviation > 1e-4 {
		t.Fatalf("geostationary orbit should complete one revolution: delta %v", deviation)
	}
}

func TestTrueAnomalyFromMeanMatchesEccentricPath(t *testing.T) {
	mean := 0.7
	eccentricity := 0.2

	trueDirect := TrueAnomalyFromMean(mean, eccentricity)
	eccentric := EccentricAnomalyFromMean(mean, eccentricity)
	trueFromEcc := TrueAnomalyFromEccentric(eccentric, eccentricity)

	if math.Abs(trueDirect-trueFromEcc) > 1e-12 {
		t.Fatalf("true anomaly conversion mismatch: %v vs %v", trueDirect, trueFromEcc)
	}
}
