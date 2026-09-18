import { Icon as IconifyIcon } from "@iconify/react/dist/iconify.js";
import { Android } from "@mui/icons-material";
import { Box, SvgIconProps, useTheme } from "@mui/material";
import SvgIcon from "@mui/material/SvgIcon/SvgIcon";
import { useMemo } from "react";
import { useAppSelector } from "../../../redux/hooks.ts";
import { fileExtension } from "../../../util";
import Book from "../../Icons/Book.tsx";
import Document from "../../Icons/Document.tsx";
import DocumentFlowchart from "../../Icons/DocumentFlowchart.tsx";
import DocumentPDF from "../../Icons/DocumentPDF.tsx";
import FileExclBox from "../../Icons/FileExclBox.tsx";
import FilePowerPointBox from "../../Icons/FilePowerPointBox.tsx";
import FileWordBox from "../../Icons/FileWordBox.tsx";
import Folder from "../../Icons/Folder.tsx";
import FolderOutlined from "../../Icons/FolderOutlined.tsx";
import FolderZip from "../../Icons/FolderZip.tsx";
import Image from "../../Icons/Image.tsx";
import LanguageC from "../../Icons/LanguageC.tsx";
import LanguageCPP from "../../Icons/LanguageCPP.tsx";
import LanguageGo from "../../Icons/LanguageGo.tsx";
import LanguageJS from "../../Icons/LanguageJS.tsx";
import LanguagePython from "../../Icons/LanguagePython.tsx";
import LanguageRust from "../../Icons/LanguageRust.tsx";
import MagnetOn from "../../Icons/MagnetOn.tsx";
import Markdown from "../../Icons/Markdown.tsx";
import MusicNote1 from "../../Icons/MusicNote1.tsx";
import Notepad from "../../Icons/Notepad.tsx";
import Raw from "../../Icons/Raw.tsx";
import Video from "../../Icons/Video.tsx";
import Whiteboard from "../../Icons/Whiteboard.tsx";
import WindowApps from "../../Icons/WindowApps.tsx";

export interface FileTypeIconProps extends SvgIconProps {
  name: string;
  fileType: number;
  notLoaded?: boolean;
  customizedColor?: string;
  [key: string]: any;
}

export interface FileTypeIconSetting {
  exts: string[];
  icon?: string;
  iconify?: string;
  img?: string;
  color?: string;
  color_dark?: string;
}

export interface ExpandedIconSettings {
  [key: string]: FileTypeIconSetting;
}

export const builtInIcons: {
  [key: string]: typeof SvgIcon | ((props: SvgIconProps) => JSX.Element);
} = {
  audio: MusicNote1,
  video: Video,
  image: Image,
  pdf: DocumentPDF,
  word: FileWordBox,
  ppt: FilePowerPointBox,
  excel: FileExclBox,
  text: Notepad,
  torrent: MagnetOn,
  zip: FolderZip,
  exe: WindowApps,
  android: Android,
  go: LanguageGo,
  c: LanguageC,
  cpp: LanguageCPP,
  js: LanguageJS,
  python: LanguagePython,
  book: Book,
  rust: LanguageRust,
  raw: Raw,
  flowchart: DocumentFlowchart,
  whiteboard: Whiteboard,
  markdown: Markdown,
};

interface TypeIcon {
  icon?: typeof SvgIcon | ((props: SvgIconProps) => JSX.Element);
  color?: string;
  color_dark?: string;
  img?: string;
  hideUnknown?: boolean;
  reverseDarkMode?: boolean;
}

interface IconComponentProps {
  icon?: typeof SvgIcon | ((props: SvgIconProps) => JSX.Element);
  color?: string;
  color_dark?: string;
  isDefault?: boolean;
  img?: string;
  iconify?: string;
}

const FileTypeIcon = ({
  name,
  fileType,
  notLoaded,
  sx,
  hideUnknown,
  customizedColor,
  reverseDarkMode,
  ...rest
}: FileTypeIconProps) => {
  const theme = useTheme();
  const iconOptions = useAppSelector((state) => state.siteConfig.explorer.typed?.icons) as ExpandedIconSettings;
  const IconComponent: IconComponentProps = useMemo(() => {
    if (fileType === 1) {
      return notLoaded ? { icon: FolderOutlined } : { icon: Folder };
    }

    if (name) {
      const fileSuffix = fileExtension(name);
      if (fileSuffix && iconOptions) {
        const options = iconOptions[fileSuffix];
        if (options) {
          const { icon, color, color_dark, img, iconify } = options;
          if (icon) {
            return {
              icon: builtInIcons[icon],
              color,
              color_dark,
            };
          } else if (img) {
            return {
              img,
            };
          } else if (iconify) {
            return {
              iconify,
              color,
              color_dark,
            };
          }
        }
      }
    }

    return { icon: Document, isDefault: true };
  }, [fileType, name, notLoaded]);

  const iconColor = useMemo(() => {
    if (customizedColor) {
      return customizedColor;
    }
    if (theme.palette.mode == (reverseDarkMode ? "light" : "dark")) {
      return IconComponent.color_dark ?? IconComponent.color ?? theme.palette.action.active;
    } else {
      return IconComponent.color ?? theme.palette.action.active;
    }
  }, [IconComponent, theme, customizedColor]);

  if (IconComponent.icon) {
    if (IconComponent.isDefault && hideUnknown) {
      return <></>;
    }
    return (
      <IconComponent.icon
        sx={{
          color: iconColor,
          ...sx,
        }}
        {...rest}
      />
    );
  } else if (IconComponent.iconify) {
    return (
      //@ts-ignore
      <Box
        component={IconifyIcon}
        sx={{
          color: iconColor,
          ...sx,
        }}
        width={24}
        height={24}
        icon={IconComponent.iconify}
        {...rest}
      />
    );
  } else {
    return (
      //@ts-ignore
      <Box
        component={"img"}
        draggable={false}
        sx={{
          width: "24px",
          height: "24px",
          objectFit: "contain",
          ...sx,
        }}
        src={IconComponent.img}
        {...rest}
      />
    );
  }
};

export default FileTypeIcon;
