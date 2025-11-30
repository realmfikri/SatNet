package visibility

import (
	"math"
	"testing"
)

func TestSlantRange(t *testing.T) {
	a := Vector3{X: EarthRadius, Y: 0, Z: 0}
	b := Vector3{X: EarthRadius + 1000, Y: 0, Z: 0}

	got := SlantRange(a, b)
	want := 1000.0

	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("slant range mismatch: got %f, want %f", got, want)
	}
}

func TestElevationAndMask(t *testing.T) {
	ground := Vector3{X: EarthRadius, Y: 0, Z: 0}
	horizonSat := Vector3{X: EarthRadius, Y: 0, Z: 500}

	elev := Elevation(ground, horizonSat)
	if math.Abs(elev) > 1e-9 {
		t.Fatalf("expected horizon elevation ~0, got %f", elev)
	}

	if !GroundToSatelliteVisible(ground, horizonSat, 0) {
		t.Fatalf("satellite on horizon should be visible when mask is zero")
	}

	if GroundToSatelliteVisible(ground, horizonSat, 1e-3) {
		t.Fatalf("satellite on horizon should not pass a positive elevation mask")
	}
}

func TestGroundToSatelliteLineOfSight(t *testing.T) {
	ground := Vector3{X: EarthRadius, Y: 0, Z: 0}
	overhead := Vector3{X: EarthRadius + 500, Y: 0, Z: 0}
	blocked := Vector3{X: -(EarthRadius + 500), Y: 0, Z: 0}

	if !GroundToSatelliteVisible(ground, overhead, 0) {
		t.Fatalf("overhead satellite should be visible")
	}

	if GroundToSatelliteVisible(ground, blocked, 0) {
		t.Fatalf("satellite through Earth should be blocked")
	}
}

func TestSatelliteToSatelliteLineOfSight(t *testing.T) {
	highAltitude := EarthRadius + 3000
	satA := Vector3{X: highAltitude, Y: 0, Z: 0}
	satB := Vector3{X: 0, Y: highAltitude, Z: 0}

	if !SatelliteToSatelliteVisible(satA, satB) {
		t.Fatalf("high-altitude cross link should clear Earth")
	}

	satC := Vector3{X: EarthRadius + 500, Y: 0, Z: 0}
	satD := Vector3{X: -(EarthRadius + 500), Y: 0, Z: 0}

	if SatelliteToSatelliteVisible(satC, satD) {
		t.Fatalf("cross-Earth satellite link should be blocked")
	}
}

func TestPolarVisibility(t *testing.T) {
	polarGround := Vector3{X: 0, Y: 0, Z: EarthRadius}
	polarSat := Vector3{X: 0, Y: 0, Z: EarthRadius + 800}

	if elev := Elevation(polarGround, polarSat); math.Abs(elev-math.Pi/2) > 1e-6 {
		t.Fatalf("expected near-zenith elevation at pole, got %f", elev)
	}

	if !GroundToSatelliteVisible(polarGround, polarSat, 0.2) {
		t.Fatalf("polar overhead satellite should exceed elevation mask")
	}
}

func TestHorizonIsNotBlocked(t *testing.T) {
	ground := Vector3{X: EarthRadius, Y: 0, Z: 0}
	tangentSat := Vector3{X: EarthRadius, Y: 0, Z: 1000}

	if segmentIntersectsEarth(ground, tangentSat, EarthRadius) {
		t.Fatalf("tangent path should not be considered intersecting Earth")
	}
}
