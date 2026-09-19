package mediameta

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
)

// DMSDecimal converts degree/minute/second GPS components to a signed
// decimal degree; negative flips the sign for south/west references.
func DMSDecimal(deg, minutes, seconds float64, negative bool) float64 {
	decimal := deg + minutes/60.0 + seconds/3600.0
	if negative {
		return -decimal
	}
	return decimal
}

// GpsRationalElem parses one rational GPS element of the form "n/d" into a
// decimal value. Returns 0 for malformed or zero-denominator elements.
func GpsRationalElem(elm string) float64 {
	elements := strings.Split(elm, "/")
	if len(elements) != 2 {
		return 0
	}

	numerator, err := strconv.ParseFloat(elements[0], 64)
	if err != nil {
		return 0
	}

	denominator, err := strconv.ParseFloat(elements[1], 64)
	if err != nil || denominator == 0 {
		return 0
	}

	return numerator / denominator
}

// GpsMeta builds the latitude/longitude MediaMeta pair from raw EXIF parts.
// parse converts a raw coordinate string and its hemisphere reference into
// decimal degrees; drivers pass their vendor-specific parser.
func GpsMeta(latitude, longitude, latRef, lonRef string, parse func(raw, ref string) float64) []driver.MediaMeta {
	if latitude == "" || longitude == "" || latRef == "" || lonRef == "" {
		return nil
	}

	lat := parse(latitude, latRef)
	lon := parse(longitude, lonRef)
	if math.IsNaN(lat) || math.IsNaN(lon) {
		return nil
	}

	latN, lngN := NormalizeGPS(lat, lon)
	return []driver.MediaMeta{{
		Key:   GpsLat,
		Value: fmt.Sprintf("%f", latN),
	}, {
		Key:   GpsLng,
		Value: fmt.Sprintf("%f", lngN),
	}}
}
