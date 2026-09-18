import { Avatar, Badge, BadgeProps, Box, Skeleton, styled, SvgIconProps, Tooltip } from "@mui/material";
import { forwardRef, memo, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FileResponse, FileType, Metadata } from "../../../api/explorer.ts";
import UserAvatar from "../../Common/User/UserAvatar.tsx";
import ShareAndroid from "../../Icons/ShareAndroid.tsx";
import EmojiIcon from "./EmojiIcon.tsx";
import FileTypeIcon from "./FileTypeIcon.tsx";

export interface FileIconProps {
  file?: FileResponse;
  variant?: "default" | "small" | "large";
  loading?: boolean;
  notLoaded?: boolean;
  [key: string]: any;
  iconProps?: SvgIconProps;
}

interface StyledBadgeProps extends BadgeProps {
  iconVariant?: "default" | "small" | "large" | "largeMobile" | "shareSingle";
}

const StyledBadge = styled(Badge)<StyledBadgeProps>(({ iconVariant }) => ({
  "& .MuiBadge-badge": {
    right: 3,
    top: variantTop[iconVariant ?? "default"],
    padding: "0",
    height: "initial",
    minWidth: "initial",
  },
  verticalAlign: "initial",
}));

const variantTop = {
  default: 18,
  small: 15,
  medium: 24,
  large: 70,
  largeMobile: 52,
  shareSingle: 26,
};

const variantAvatarSize = {
  default: 16,
  small: 13,
  medium: 20,
  large: 32,
  largeMobile: 24,
  shareSingle: 20,
};

const FileIcon = memo(
  forwardRef(({ file, loading, variant = "default", iconProps, notLoaded, sx, ...rest }: FileIconProps, ref) => {
    const { t } = useTranslation();
    const iconColor = useMemo(() => {
      if (file && file.metadata && file.metadata[Metadata.icon_color]) {
        return file.metadata[Metadata.icon_color];
      }
    }, [file]);
    const typedIcon = useMemo(() => {
      if (file?.metadata?.[Metadata.emoji]) {
        const { sx, ...restIcon } = iconProps ?? {};
        return <EmojiIcon sx={sx} emoji={file.metadata[Metadata.emoji]} />;
      }
      return (
        <FileTypeIcon
          name={file?.name ?? ""}
          fileType={file?.type ?? FileType.folder}
          color={"action"}
          notLoaded={notLoaded}
          customizedColor={iconColor ?? ""}
          {...iconProps}
        />
      );
    }, [file, iconColor, iconProps, notLoaded]);
    const badgeContent = useMemo(() => {
      const avatarSize = variantAvatarSize[variant];
      if (file?.metadata?.[Metadata.share_redirect]) {
        return (
          <UserAvatar
            overwriteTextSize
            sx={{ width: avatarSize, height: avatarSize }}
            uid={file.metadata[Metadata.share_owner]}
          />
        );
      } else if (file?.shared) {
        return (
          <Tooltip title={t("application:fileManager.sharedWithOthers")}>
            <Avatar
              sx={{
                width: avatarSize,
                height: avatarSize,
                bgcolor: (theme) => theme.palette.background.default,
              }}
            >
              <ShareAndroid sx={{ fontSize: `${avatarSize - 0}px!important` }} color={"action"} />
            </Avatar>
          </Tooltip>
        );
      }
    }, [file, variant]);
    return (
      <Box ref={ref} sx={{ px: 2, py: 1.5, ...sx }} {...rest}>
        {!loading &&
          (badgeContent ? (
            <StyledBadge iconVariant={variant} badgeContent={badgeContent}>
              {typedIcon}
            </StyledBadge>
          ) : (
            typedIcon
          ))}
        {loading && <Skeleton variant="circular" width={24} height={24} />}
      </Box>
    );
  }),
);

export default FileIcon;
