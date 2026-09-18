import { BoxProps, Skeleton } from "@mui/material";
import { lazy, Suspense } from "react";
import { useAppSelector } from "../../../../redux/hooks.ts";

const LeafletMapBox = lazy(() => import("./LeafletMapBox.tsx"));
const Mapbox = lazy(() => import("./Mapbox.tsx"));

export interface MapLoaderProps extends BoxProps {
  height: number;
  center: [number, number];
  mapProvider?: string;
  googleTileType?: string;
  token?: string;
}

const MapLoader = (props: MapLoaderProps) => {
  const mapProvider = useAppSelector((state) => state.siteConfig.explorer.config.map_provider);
  const mapboxAk = useAppSelector((state) => state.siteConfig.explorer.config.mapbox_ak);
  const googleTileType = useAppSelector((state) => state.siteConfig.explorer.config.google_map_tile_type);
  return (
    <Suspense fallback={<Skeleton variant="rounded" width={"100%"} height={props.height} />}>
      {mapProvider !== "mapbox" && (
        <LeafletMapBox mapProvider={mapProvider} googleTileType={googleTileType} {...props} />
      )}
      {mapProvider === "mapbox" && (
        <Mapbox mapProvider={mapProvider} googleTileType={googleTileType} token={mapboxAk} {...props} />
      )}
    </Suspense>
  );
};

export default MapLoader;
