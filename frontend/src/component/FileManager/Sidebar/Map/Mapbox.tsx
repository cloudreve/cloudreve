import { Box, useTheme } from "@mui/material";
import mapboxgl from "mapbox-gl";
import "mapbox-gl/dist/mapbox-gl.css";
import { useEffect, useRef } from "react";
import { MapLoaderProps } from "./MapLoader.tsx";

const MapboxComponent = ({ center, height, mapProvider, token, sx, ...rest }: MapLoaderProps) => {
  const theme = useTheme();
  const mapRef = useRef<mapboxgl.Map | null>(null);
  const mapContainerRef = useRef<HTMLDivElement | null>(null);
  const markerRef = useRef<mapboxgl.Marker | null>(null);

  // Dynamic map style based on theme mode
  const mapStyle = "mapbox://styles/mapbox/standard";

  useEffect(() => {
    if (!mapContainerRef.current) return;

    // Set Mapbox access token
    mapboxgl.accessToken = token;

    // Initialize map
    mapRef.current = new mapboxgl.Map({
      container: mapContainerRef.current,
      style: mapStyle,
      config: {
        basemap: {
          lightPreset: theme.palette.mode === "dark" ? "night" : "day",
        },
      },
      center: [center[1], center[0]], // Mapbox uses [lng, lat] format
      zoom: 13,
    });

    // Add marker
    markerRef.current = new mapboxgl.Marker().setLngLat([center[1], center[0]]).addTo(mapRef.current);

    return () => {
      if (markerRef.current) {
        markerRef.current.remove();
      }
      if (mapRef.current) {
        mapRef.current.remove();
      }
    };
  }, [center]);

  useEffect(() => {
    if (mapRef.current) {
      mapRef.current.setConfigProperty("basemap", "lightPreset", theme.palette.mode === "dark" ? "night" : "day");
    }
  }, [theme.palette.mode]);

  // Update marker position when center changes
  useEffect(() => {
    if (markerRef.current) {
      markerRef.current.setLngLat([center[1], center[0]]);
    }
    if (mapRef.current) {
      mapRef.current.setCenter([center[1], center[0]]);
    }
  }, [center]);

  return (
    <Box
      sx={{
        width: "100%",
        height: height,
        borderRadius: theme.shape.borderRadius / 8,
        ...sx,
      }}
      {...rest}
    >
      <div
        ref={mapContainerRef}
        style={{
          height: "100%",
          borderRadius: theme.shape.borderRadius,
        }}
      />
    </Box>
  );
};

export default MapboxComponent;
