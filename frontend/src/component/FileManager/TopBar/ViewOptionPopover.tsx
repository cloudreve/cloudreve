import {
  Box,
  Collapse,
  Popover,
  PopoverProps,
  Slider,
  SvgIconProps,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from "@mui/material";
import React, { useContext, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { Layouts } from "../../../redux/fileManagerSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import {
  applyGalleryWidth,
  changePageSize,
  setLayoutSetting,
  setThumbToggle,
} from "../../../redux/thunks/filemanager.ts";
import NavIconTransition from "../../Frame/NavBar/NavIconTransition.tsx";
import AppsList from "../../Icons/AppsList.tsx";
import AppsListOutlined from "../../Icons/AppsListOutlined.tsx";
import Grid from "../../Icons/Grid.tsx";
import GridOutlined from "../../Icons/GridOutlined.tsx";
import ImageCopy from "../../Icons/ImageCopy.tsx";
import ImageCopyOutlined from "../../Icons/ImageCopyOutlined.tsx";
import ImageOffOutlined from "../../Icons/ImageOffOutlined.tsx";
import ImageOutlined from "../../Icons/ImageOutlined.tsx";

import { setListViewColumnSettingDialog } from "../../../redux/globalStateSlice.ts";
import Setting from "../../Icons/Setting.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";

const layoutOptions: {
  label: string;
  value: string;
  icon: ((props: SvgIconProps) => JSX.Element)[];
}[] = [
  {
    label: "application:fileManager.gridView",
    value: "grid",
    icon: [Grid, GridOutlined],
  },
  {
    label: "application:fileManager.listView",
    value: "list",
    icon: [AppsList, AppsListOutlined],
  },
  {
    label: "application:fileManager.galleryView",
    value: "gallery",
    icon: [ImageCopy, ImageCopyOutlined],
  },
];

const thumbOptions: {
  label: string;
  value: boolean;
  icon: (props: SvgIconProps) => JSX.Element;
}[] = [
  {
    label: "application:fileManager.on",
    value: true,
    icon: ImageOutlined,
  },
  {
    label: "application:fileManager.off",
    value: false,
    icon: ImageOffOutlined,
  },
];

export const MinPageSize = 50;

const ViewOptionPopover = ({ ...rest }: PopoverProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const fmIndex = useContext(FmIndexContext);
  const layout = useAppSelector((state) => state.fileManager[fmIndex].layout);
  const showThumb = useAppSelector((state) => state.fileManager[fmIndex].showThumb);
  const pageSize = useAppSelector((state) => state.fileManager[fmIndex].pageSize);
  const pageSizeMax = useAppSelector((state) => state.fileManager[fmIndex].list?.props.max_page_size);
  const galleryWidth = useAppSelector((state) => state.fileManager[fmIndex].galleryWidth);
  const [desiredPageSize, setDesiredPageSize] = React.useState(pageSize);
  const pageSizeMaxSafe = pageSizeMax ?? desiredPageSize;
  const step = pageSizeMaxSafe - MinPageSize <= 100 ? 1 : 10;
  const [desiredImageWidth, setDesiredImageWidth] = React.useState(galleryWidth);

  useEffect(() => {
    setDesiredPageSize(pageSize);
  }, [pageSize]);

  const handleLayoutChange = (_event: React.MouseEvent<HTMLElement>, newMode: string) => {
    if (newMode) {
      dispatch(setLayoutSetting(fmIndex, newMode));
    }
  };

  const handleThumbChange = (_event: React.MouseEvent<HTMLElement>, newMode: boolean) => {
    dispatch(setThumbToggle(fmIndex, newMode));
  };

  const handlePageSlideChange = (_event: Event, newValue: number | number[]) => {
    setDesiredPageSize(newValue as number);
  };

  const commitPageSize = (_event: React.SyntheticEvent | Event, newValue: number | number[]) => {
    const pageSize = Math.max(MinPageSize, newValue as number);
    dispatch(changePageSize(fmIndex, pageSize));
  };

  const handleImageSizeChange = (_event: Event, newValue: number | number[]) => {
    setDesiredImageWidth(newValue as number);
  };

  const commitImageSize = (_event: React.SyntheticEvent | Event, newValue: number | number[]) => {
    dispatch(applyGalleryWidth(fmIndex, newValue as number));
  };

  return (
    <Popover
      anchorOrigin={{
        vertical: "bottom",
        horizontal: "center",
      }}
      transformOrigin={{
        vertical: "top",
        horizontal: "center",
      }}
      {...rest}
    >
      <Box sx={{ p: 2, minWidth: "300px" }}>
        <Box>
          <Typography variant="subtitle2" sx={{ mb: 0.5, ml: 0.5 }} color={"text.secondary"}>
            {t("application:fileManager.layout")}
          </Typography>
          <ToggleButtonGroup
            value={layout}
            onChange={handleLayoutChange}
            fullWidth
            size="small"
            color="primary"
            exclusive
          >
            {layoutOptions.map((option) => (
              <ToggleButton key={option.value} value={option.value}>
                <NavIconTransition
                  iconProps={{ fontSize: "small" }}
                  sx={{ mr: 1, height: "20px" }}
                  fileIcon={option.icon}
                  active={layout === option.value}
                />
                {t(option.label)}
              </ToggleButton>
            ))}
          </ToggleButtonGroup>
        </Box>
        <Collapse in={layout == Layouts.grid || layout == Layouts.list}>
          <Typography variant="subtitle2" sx={{ mt: 1.5, mb: 0.5, ml: 0.5 }} color={"text.secondary"}>
            {t("application:fileManager.thumbnails")}
          </Typography>
          <ToggleButtonGroup
            onChange={handleThumbChange}
            value={showThumb}
            fullWidth
            size="small"
            color="primary"
            exclusive
          >
            {thumbOptions.map((option) => (
              <ToggleButton key={option.label} value={option.value}>
                <option.icon sx={{ mr: 1, height: "20px" }} fontSize={"small"} />
                {t(option.label)}
              </ToggleButton>
            ))}
          </ToggleButtonGroup>
        </Collapse>
        <Collapse in={layout == Layouts.list}>
          <Typography variant="subtitle2" sx={{ mt: 1.5, mb: 0.5, ml: 0.5 }} color={"text.secondary"}>
            {t("application:fileManager.listColumnSetting")}
          </Typography>
          <ToggleButtonGroup value={0} fullWidth size="small" color="primary" exclusive>
            <ToggleButton value={1} onClick={() => dispatch(setListViewColumnSettingDialog(true))}>
              <Setting sx={{ mr: 1, height: "20px" }} fontSize={"small"} />
              {t("application:fileManager.listColumnSetting")}
            </ToggleButton>
          </ToggleButtonGroup>
        </Collapse>
        <Collapse in={layout == Layouts.gallery}>
          <Typography variant="subtitle2" sx={{ mt: 1.5, mb: 0.5, ml: 0.5 }} color={"text.secondary"}>
            {t("application:fileManager.imageSize")}
          </Typography>
          <Box sx={{ px: 1 }}>
            <Slider
              size={"small"}
              value={desiredImageWidth}
              valueLabelDisplay="auto"
              step={10}
              onChange={handleImageSizeChange}
              onChangeCommitted={commitImageSize}
              min={50}
              max={500}
            />
            <Box sx={{ display: "flex", justifyContent: "space-between" }}>
              <Typography variant="body2" color={"text.secondary"}>
                50
              </Typography>
              <Typography variant="body2" color={"text.secondary"}>
                500
              </Typography>
            </Box>
          </Box>
        </Collapse>
        <Typography variant="subtitle2" sx={{ mt: 1.5, mb: 0.5, ml: 0.5 }} color={"text.secondary"}>
          {t("application:fileManager.paginationSize")}
        </Typography>
        <Box sx={{ px: 1 }}>
          <Slider
            size={"small"}
            value={desiredPageSize}
            valueLabelDisplay="auto"
            step={step}
            onChange={handlePageSlideChange}
            onChangeCommitted={commitPageSize}
            min={MinPageSize}
            max={pageSizeMaxSafe}
          />
          <Box sx={{ display: "flex", justifyContent: "space-between" }}>
            <Typography variant="body2" color={"text.secondary"}>
              {MinPageSize}
            </Typography>
            <Typography variant="body2" color={"text.secondary"}>
              {pageSizeMaxSafe}
            </Typography>
          </Box>
        </Box>
      </Box>
    </Popover>
  );
};

export default ViewOptionPopover;
