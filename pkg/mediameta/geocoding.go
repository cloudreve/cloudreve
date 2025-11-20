package mediameta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/driver"
	"github.com/cloudreve/Cloudreve/v4/pkg/filemanager/manager/entitysource"
	"github.com/cloudreve/Cloudreve/v4/pkg/logging"
	"github.com/cloudreve/Cloudreve/v4/pkg/request"
	"github.com/cloudreve/Cloudreve/v4/pkg/setting"
)

const mapBoxURL = "https://api.mapbox.com/search/geocode/v6/reverse"

const (
	Street   = "street"
	Locality = "locality"
	Place    = "place"
	District = "district"
	Region   = "region"
	Country  = "country"
)

type geocodingExtractor struct {
	settings setting.Provider
	l        logging.Logger
	client   request.Client
}

func newGeocodingExtractor(settings setting.Provider, l logging.Logger, client request.Client) *geocodingExtractor {
	return &geocodingExtractor{
		settings: settings,
		l:        l,
		client:   client,
	}
}

func (e *geocodingExtractor) Exts() []string {
	return exifExts
}

func (e *geocodingExtractor) Extract(ctx context.Context, ext string, source entitysource.EntitySource, opts ...optionFunc) ([]driver.MediaMeta, error) {
	option := &option{}
	for _, opt := range opts {
		opt.apply(option)
	}

	// Find GPS info from extracted
	var latStr, lngStr string
	for _, meta := range option.extracted {
		if meta.Key == GpsLat {
			latStr = meta.Value
		}
		if meta.Key == GpsLng {
			lngStr = meta.Value
		}
	}

	if latStr == "" || lngStr == "" {
		return nil, nil
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return nil, fmt.Errorf("geocoding: failed to parse latitude: %w", err)
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		return nil, fmt.Errorf("geocoding: failed to parse longitude: %w", err)
	}

	metas, err := e.getGeocoding(ctx, lat, lng, option.language)
	if err != nil {
		return nil, fmt.Errorf("geocoding: failed to get geocoding: %w", err)
	}

	for i, _ := range metas {
		metas[i].Type = driver.MetaTypeGeocoding
	}

	return metas, nil
}

func (e *geocodingExtractor) getGeocoding(ctx context.Context, lat, lng float64, language string) ([]driver.MediaMeta, error) {
	values := url.Values{}
	values.Add("longitude", fmt.Sprintf("%f", lng))
	values.Add("latitude", fmt.Sprintf("%f", lat))
	values.Add("limit", "1")
	values.Add("access_token", e.settings.MediaMetaGeocodingMapboxAK(ctx))
	if language != "" {
		values.Add("language", language)
	}

	resp, err := e.client.Request(
		"GET",
		mapBoxURL+"?"+values.Encode(),
		nil,
		request.WithContext(ctx),
		request.WithLogger(e.l),
	).CheckHTTPResponse(http.StatusOK).GetResponse()
	if err != nil {
		return nil, fmt.Errorf("failed to get geocoding from mapbox: %w", err)
	}

	var geocoding MapboxGeocodingResponse
	if err := json.Unmarshal([]byte(resp), &geocoding); err != nil {
		return nil, fmt.Errorf("failed to unmarshal geocoding from mapbox: %w", err)
	}

	if len(geocoding.Features) == 0 {
		return nil, nil
	}

	metas := make([]driver.MediaMeta, 0)
	contexts := geocoding.Features[0].Properties.Context
	if contexts.Street != nil {
		metas = append(metas, driver.MediaMeta{
			Key:   Street,
			Value: contexts.Street.Name,
		})
	}
	if contexts.Locality != nil {
		metas = append(metas, driver.MediaMeta{
			Key:   Locality,
			Value: contexts.Locality.Name,
		})
	}
	if contexts.Place != nil {
		metas = append(metas, driver.MediaMeta{
			Key:   Place,
			Value: contexts.Place.Name,
		})
	}
	if contexts.District != nil {
		metas = append(metas, driver.MediaMeta{
			Key:   District,
			Value: contexts.District.Name,
		})
	}
	if contexts.Region != nil {
		metas = append(metas, driver.MediaMeta{
			Key:   Region,
			Value: contexts.Region.Name,
		})
	}
	if contexts.Country != nil {
		metas = append(metas, driver.MediaMeta{
			Key:   Country,
			Value: contexts.Country.Name,
		})
	}

	return metas, nil
}

// MapboxGeocodingResponse represents the response from Mapbox Geocoding API
type MapboxGeocodingResponse struct {
	Type        string    `json:"type"`        // "FeatureCollection"
	Features    []Feature `json:"features"`    // Array of feature objects
	Attribution string    `json:"attribution"` // Attribution to Mapbox
}

// Feature represents a feature object in the geocoding response
type Feature struct {
	ID         string     `json:"id"`         // Feature ID (same as mapbox_id)
	Type       string     `json:"type"`       // "Feature"
	Geometry   Geometry   `json:"geometry"`   // Spatial geometry of the feature
	Properties Properties `json:"properties"` // Feature details
}

// Geometry represents the spatial geometry of a feature
type Geometry struct {
	Type        string    `json:"type"`        // "Point"
	Coordinates []float64 `json:"coordinates"` // [longitude, latitude]
}

// Properties contains the feature's detailed information
type Properties struct {
	MapboxID       string      `json:"mapbox_id"`       // Unique feature identifier
	FeatureType    string      `json:"feature_type"`    // Type of feature (country, region, etc.)
	Name           string      `json:"name"`            // Formatted address string
	NamePreferred  string      `json:"name_preferred"`  // Canonical or common alias
	PlaceFormatted string      `json:"place_formatted"` // Formatted context string
	FullAddress    string      `json:"full_address"`    // Full formatted address
	Context        Context     `json:"context"`         // Hierarchy of parent features
	Coordinates    Coordinates `json:"coordinates"`     // Geographic position and accuracy
	BBox           []float64   `json:"bbox,omitempty"`  // Bounding box [minLon,minLat,maxLon,maxLat]
	MatchCode      MatchCode   `json:"match_code"`      // Metadata about result matching
}

// Context represents the hierarchy of encompassing parent features
type Context struct {
	Country      *ContextFeature `json:"country,omitempty"`
	Region       *ContextFeature `json:"region,omitempty"`
	Postcode     *ContextFeature `json:"postcode,omitempty"`
	District     *ContextFeature `json:"district,omitempty"`
	Place        *ContextFeature `json:"place,omitempty"`
	Locality     *ContextFeature `json:"locality,omitempty"`
	Neighborhood *ContextFeature `json:"neighborhood,omitempty"`
	Street       *ContextFeature `json:"street,omitempty"`
}

// ContextFeature represents a feature in the context hierarchy
type ContextFeature struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	NamePreferred string `json:"name_preferred,omitempty"`
	MapboxID      string `json:"mapbox_id"`
}

// Coordinates represents geographical position and accuracy information
type Coordinates struct {
	Longitude      float64         `json:"longitude"`                 // Longitude of result
	Latitude       float64         `json:"latitude"`                  // Latitude of result
	Accuracy       string          `json:"accuracy,omitempty"`        // Accuracy metric for address results
	RoutablePoints []RoutablePoint `json:"routable_points,omitempty"` // Array of routable points
}

// RoutablePoint represents a routable point for an address feature
type RoutablePoint struct {
	Name      string  `json:"name"`      // Name of the routable point
	Longitude float64 `json:"longitude"` // Longitude coordinate
	Latitude  float64 `json:"latitude"`  // Latitude coordinate
}

// MatchCode contains metadata about how result components match the input query
type MatchCode struct {
	// Add specific match code fields as needed based on Mapbox documentation
	// This structure may vary depending on the specific match codes returned
}
