import { FileResponse, FileType } from "../../api/explorer.ts";
import FileIcon from "./Explorer/FileIcon.tsx";
import React, { useMemo } from "react";
import { Box, ButtonProps, Skeleton, Tooltip } from "@mui/material";
import { BadgeText, DefaultButton } from "../Common/StyledComponents.tsx";
import CrUri from "../../util/uri.ts";
import { useTranslation } from "react-i18next";
import { usePopupState } from "material-ui-popup-state/hooks";
import { bindHover, bindPopover } from "material-ui-popup-state";
import HoverPopover from "material-ui-popup-state/HoverPopover";
import Breadcrumb from "./TopBar/Breadcrumb.tsx";
import { useBreadcrumbButtons } from "./TopBar/BreadcrumbButton.tsx";

export interface FileBadgeFile {
  path: string;
  type: number;
}

export interface FileBadgeProps extends ButtonProps {
  file?: FileResponse;
  simplifiedFile?: FileBadgeFile;
  unknown?: boolean;
  clickable?: boolean;
}

const FileBadge = ({ file, clickable, simplifiedFile, unknown, ...rest }: FileBadgeProps) => {
  const { t } = useTranslation();
  const popupState = usePopupState({
    variant: "popover",
    popupId: "fileBadge",
  });
  const hoverProps = bindHover(popupState);
  const popoverProps = bindPopover(popupState);

  const name = useMemo(() => {
    if (unknown) {
      return t("application:modals.unknownParent");
    }

    if (file?.name) {
      return file?.name;
    }

    try {
      const uri = new CrUri(simplifiedFile?.path ?? "");
      return uri.elements().pop() ?? "";
    } catch (e) {
      return "";
    }
  }, [file, unknown, simplifiedFile]);

  const f = useMemo(() => {
    if (file) {
      return file;
    }

    return {
      name,
      type: simplifiedFile?.type ?? FileType.folder,
      id: "",
      created_at: "",
      updated_at: "",
      size: 0,
      path: simplifiedFile?.path ?? "",
    } as FileResponse;
  }, [file, unknown, simplifiedFile]);

  const [loading, displayName, startIcon, onClick] = useBreadcrumbButtons({
    name,
    is_latest: false,
    path: f.path,
  });

  const StartIcon = useMemo(() => {
    if (loading) {
      return <Skeleton width={20} height={20} variant={"rounded"} />;
    }
    if (startIcon?.Icons?.[0]) {
      const Icon = startIcon?.Icons?.[0];
      return <Icon color={"action"} fontSize={"small"} />;
    }
    if (startIcon?.Element) {
      return startIcon.Element({ sx: { width: 20, height: 20 } });
    }
  }, [startIcon, loading]);

  const tooltip = useMemo(() => {
    if (unknown) {
      return t("application:modals.unknownParentDes");
    }

    return "";
  }, [file, unknown, simplifiedFile]);

  const parent = useMemo(() => {
    const uri = simplifiedFile?.path ?? file?.path;
    if (!uri) {
      return "";
    }

    const crUri = new CrUri(uri);
    return crUri.parent().toString();
  }, [file, unknown, simplifiedFile]);

  return (
    <>
      <Tooltip title={tooltip}>
        <span style={{ maxWidth: "100%" }}>
          <DefaultButton
            sx={{ maxWidth: "100%" }}
            onClick={clickable ? onClick : undefined}
            disabled={unknown}
            {...rest}
            {...(unknown ? {} : hoverProps)}
          >
            {StartIcon ? (
              StartIcon
            ) : (
              <FileIcon
                variant={"small"}
                file={f}
                sx={{ px: 0, py: 0, height: "20px" }}
                fontSize={"small"}
                iconProps={{ fontSize: "small", sx: { minWidth: "20px" } }}
              />
            )}

            <BadgeText variant={"body2"}>{name == "" ? displayName : name}</BadgeText>
          </DefaultButton>
        </span>
      </Tooltip>
      {!unknown && (
        <HoverPopover
          anchorOrigin={{
            vertical: "top",
            horizontal: "center",
          }}
          transformOrigin={{
            vertical: "bottom",
            horizontal: "center",
          }}
          {...popoverProps}
          disableScrollLock={false}
        >
          <Box sx={{ maxWidth: "600px" }}>
            <Breadcrumb targetPath={parent} displayOnly />
          </Box>
        </HoverPopover>
      )}
    </>
  );
};

export default FileBadge;
