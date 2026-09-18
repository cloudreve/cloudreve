import { Box, Fade, IconButton, styled } from "@mui/material";
import { useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { CustomProps } from "../../../../api/explorer.ts";
import { NoWrapTypography } from "../../../Common/StyledComponents.tsx";
import ArrowSortDownFilled from "../../../Icons/ArrowSortDownFilled.tsx";
import Divider from "../../../Icons/Divider.tsx";
import { ResizeProps } from "./ListHeader.tsx";

export interface ListViewColumn {
  type: ColumType;
  width?: number;
  defaults: ColumTypeDefaults;
  props?: ColumTypeProps;
}

export interface ListViewColumnSetting {
  type: ColumType;
  width?: number;
  props?: ColumTypeProps;
}

export interface ColumTypeProps {
  metadata_key?: string;
  custom_props_id?: string;
}

export enum ColumType {
  name = 0,
  date_modified = 1,
  size = 2,
  metadata = 3,
  date_created = 4,
  permission = 5,
  parent = 6,
  recycle_restore_parent = 7,
  recycle_expire = 8,

  // Media info
  aperture = 9,
  exposure = 10,
  iso = 11,
  camera_make = 12,
  camera_model = 13,
  lens_make = 14,
  lens_model = 15,
  focal_length = 16,
  exposure_bias = 17,
  flash = 18,
  software = 19,
  taken_at = 20,
  image_size = 21,
  title = 22,
  artist = 23,
  album = 24,
  duration = 25,
  street = 27,
  locality = 28,
  place = 29,
  district = 30,
  region = 31,
  country = 32,

  // Custom props
  custom_props = 26,
}

export interface ColumTypeDefaults {
  title: string;
  width: number;
  widthMobile?: number;
  minWidth?: number;
  order_by?: string;
}

export interface ColumnProps {
  index: number;
  column: ListViewColumn;
  showDivider?: boolean;
  startResizing: (props: ResizeProps) => void;
  sortable?: boolean;
  sortDirection?: string;
  setSortBy?: (order_by: string, order_direction: string) => void;
}

export const ColumnTypeDefaults: { [key: number]: ColumTypeDefaults } = {
  [ColumType.name]: {
    title: "application:fileManager.name",
    widthMobile: 300,
    width: 600,
    order_by: "name",
  },
  [ColumType.size]: {
    title: "application:fileManager.size",
    width: 100,
    order_by: "size",
  },
  [ColumType.date_modified]: {
    title: "application:fileManager.lastModified",
    width: 200,
    order_by: "updated_at",
  },
  [ColumType.date_created]: {
    title: "application:fileManager.createDate",
    width: 200,
    order_by: "created_at",
  },
  [ColumType.parent]: {
    title: "application:fileManager.parentFolder",
    width: 200,
  },
  [ColumType.recycle_restore_parent]: {
    title: "application:fileManager.originalLocation",
    width: 200,
  },
  [ColumType.recycle_expire]: {
    title: "application:fileManager.expires",
    width: 200,
  },
  [ColumType.aperture]: {
    title: "application:fileManager.aperture",
    width: 100,
  },
  [ColumType.exposure]: {
    title: "application:fileManager.exposure",
    width: 100,
  },
  [ColumType.iso]: {
    title: "application:fileManager.iso",
    width: 100,
  },
  [ColumType.camera_make]: {
    title: "application:fileManager.cameraMake",
    width: 100,
  },
  [ColumType.camera_model]: {
    title: "application:fileManager.cameraModel",
    width: 100,
  },
  [ColumType.lens_make]: {
    title: "application:fileManager.lensMake",
    width: 100,
  },
  [ColumType.lens_model]: {
    title: "application:fileManager.lensModel",
    width: 100,
  },
  [ColumType.focal_length]: {
    title: "application:fileManager.focalLength",
    width: 100,
  },
  [ColumType.exposure_bias]: {
    title: "application:fileManager.exposureBias",
    width: 100,
  },
  [ColumType.flash]: {
    title: "application:fileManager.flash",
    width: 100,
  },
  [ColumType.software]: {
    title: "application:fileManager.software",
    width: 100,
  },
  [ColumType.taken_at]: {
    title: "application:fileManager.takenAt",
    width: 200,
  },
  [ColumType.image_size]: {
    title: "application:fileManager.resolution",
    width: 100,
  },
  [ColumType.title]: {
    title: "application:fileManager.title",
    width: 200,
  },
  [ColumType.artist]: {
    title: "application:fileManager.artist",
    width: 100,
  },
  [ColumType.album]: {
    title: "application:fileManager.album",
    width: 200,
  },
  [ColumType.duration]: {
    title: "application:fileManager.duration",
    width: 100,
  },
  [ColumType.street]: {
    title: "application:fileManager.street",
    width: 100,
  },
  [ColumType.locality]: {
    title: "application:fileManager.locality",
    width: 100,
  },
  [ColumType.place]: {
    title: "application:fileManager.place",
    width: 100,
  },
  [ColumType.district]: {
    title: "application:fileManager.district",
    width: 100,
  },
  [ColumType.region]: {
    title: "application:fileManager.region",
    width: 100,
  },
  [ColumType.country]: {
    title: "application:fileManager.country",
    width: 100,
  },
};

export const getColumnTypeDefaults = (
  c: ListViewColumnSetting,
  isMobile?: boolean,
  customProps?: CustomProps[],
): ColumTypeDefaults => {
  if (ColumnTypeDefaults[c.type]) {
    return {
      ...ColumnTypeDefaults[c.type],
      width:
        isMobile && ColumnTypeDefaults[c.type].widthMobile
          ? ColumnTypeDefaults[c.type].widthMobile
          : ColumnTypeDefaults[c.type].width,
    };
  }

  if (c.type === ColumType.custom_props) {
    const customProp = customProps?.find((p) => p.id === c.props?.custom_props_id);
    return {
      title: customProp?.name ?? "application:fileManager.customProps",
      width: 100,
    };
  }

  return {
    title: "application:fileManager.metadataColumn",
    width: 100,
  };
};

const ColumnContainer = styled(Box)<{
  w: number;
}>(({ w }) => ({
  height: "39px",
  width: `${w}px`,
  display: "flex",
  alignItems: "center",
  padding: "0 10px",
}));

const DividerContainer = styled(Box)(({ theme }) => ({
  color: theme.palette.divider,
  maxWidth: "10px",
  display: "flex",
  alignItems: "center",
  cursor: "col-resize",
  "&:hover": {
    color: theme.palette.primary.main,
  },
  transition: theme.transitions.create(["color"], {
    duration: theme.transitions.duration.shortest,
  }),
  position: "relative",
  right: "-8px",
}));

const SortArrow = styled(ArrowSortDownFilled)<{
  direction?: string;
}>(({ theme, direction }) => ({
  width: "18px",
  height: "18px",
  color: !direction ? theme.palette.action.disabled : theme.palette.action.active,
  transform: `rotate(${direction === "asc" ? 180 : 0}deg)`,
  transition: theme.transitions.create(["color", "transform"], {
    duration: theme.transitions.duration.shortest,
  }),
}));

const Column = ({ column, showDivider, index, startResizing, sortDirection, setSortBy, sortable }: ColumnProps) => {
  const [showSortButton, setShowSortButton] = useState(false);
  const { t } = useTranslation();
  const onSortOptionChange = useCallback(() => {
    if (!sortable || !column.defaults.order_by) return;
    const newDirection = sortDirection === "asc" ? "desc" : "asc";
    setSortBy && setSortBy(column.defaults.order_by, newDirection);
  }, [setSortBy, sortDirection, sortable, column]);

  return (
    <ColumnContainer w={column.width ?? column.defaults.width}>
      <Box
        sx={{
          width: "100%",
          display: "flex",
          alignItems: "center",
          cursor: sortable ? "pointer" : "default",
        }}
        onMouseEnter={() => setShowSortButton(!!sortable)}
        onMouseLeave={() => setShowSortButton(false)}
        onClick={sortable ? onSortOptionChange : undefined}
      >
        <NoWrapTypography variant={"body2"} fontWeight={600}>
          {t(column.defaults.title, {
            metadata: column.props?.metadata_key,
          })}
        </NoWrapTypography>
        {sortable && (
          <Fade in={showSortButton || !!sortDirection}>
            <IconButton sx={{ ml: 1 }} size={"small"}>
              <SortArrow direction={sortDirection} />
            </IconButton>
          </Fade>
        )}
      </Box>
      <Fade in={showDivider}>
        <DividerContainer
          onMouseDown={(e) =>
            startResizing({
              index,
              startX: e.clientX,
            })
          }
        >
          <Divider />
        </DividerContainer>
      </Fade>
    </ColumnContainer>
  );
};

export default Column;
