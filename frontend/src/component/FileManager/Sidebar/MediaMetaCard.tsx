import {
  Box,
  Link,
  LinkProps,
  ListItemIcon,
  ListItemText,
  Menu,
  styled,
  SvgIconProps,
  Typography,
} from "@mui/material";
import SvgIcon from "@mui/material/SvgIcon/SvgIcon";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { searchMetadata } from "../../../redux/thunks/filemanager.ts";
import { copyToClipboard } from "../../../util";
import Clipboard from "../../Icons/Clipboard.tsx";
import Search from "../../Icons/Search.tsx";
import { SquareMenuItem } from "../ContextMenu/ContextMenu.tsx";
import { FileManagerIndex } from "../FileManager.tsx";

export interface MediaMetaElements {
  display: string;
  searchKey: string;
  searchValue: string;
}

export interface MediaMetaContent {
  title: (MediaMetaElements | string)[];
  content: (MediaMetaElements | string)[];
}

export interface MediaMetaCardProps {
  contents: MediaMetaContent[];
  icon?: typeof SvgIcon | ((props: SvgIconProps) => JSX.Element);
}

export const StyledButtonBase = styled(Box)(({ theme }) => {
  let bgColor = theme.palette.mode === "light" ? theme.palette.grey[100] : theme.palette.grey[900];
  let bgColorHover = theme.palette.mode === "light" ? theme.palette.grey[300] : theme.palette.grey[700];
  return {
    borderRadius: theme.shape.borderRadius,
    backgroundColor: bgColor,
    display: "flex",
    width: "100%",
    wordBreak: "break-all",
    alignItems: "center",
    justifyContent: "flex-start",
    padding: "8px 16px",
    transition: "all 250ms cubic-bezier(0.4, 0, 0.2, 1) 0ms",
    transitionProperty: "background-color,opacity,box-shadow",
    gap: 15,

    textAlign: "left",
    height: "100%",
    userSelect: "text",
    overflow: "hidden",
  };
});

export interface MediaMetaElementsProps extends LinkProps {
  element: MediaMetaElements;
}

export const MediaMetaElements = ({ element, ...rest }: MediaMetaElementsProps) => {
  const dispatch = useAppDispatch();
  const { t } = useTranslation();

  const handleSearch = () => {
    dispatch(searchMetadata(FileManagerIndex.main, element.searchKey, element.searchValue));
    handleClose();
  };

  const handleCopy = () => {
    copyToClipboard(element.display);
    handleClose();
  };

  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const handleClose = () => {
    setAnchorEl(null);
  };
  return (
    <>
      <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={handleClose}>
        <SquareMenuItem onClick={handleSearch} dense>
          <ListItemIcon>
            <Search fontSize="small" />
          </ListItemIcon>
          <ListItemText>
            {t("application:fileManager.searchSomething", {
              text: element?.display ?? "",
            })}
          </ListItemText>
        </SquareMenuItem>
        <SquareMenuItem onClick={handleCopy} dense>
          <ListItemIcon>
            <Clipboard fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.copyToClipboard")}</ListItemText>
        </SquareMenuItem>
      </Menu>
      <Link color="inherit" href={"#"} onClick={(e) => setAnchorEl(e.currentTarget)} underline="hover" {...rest}>
        {element.display}
      </Link>
    </>
  );
};

const MediaMetaCard = ({ contents, icon }: MediaMetaCardProps) => {
  const Icon = icon;
  return (
    <>
      <StyledButtonBase>
        {Icon && <Icon sx={{ pt: "2px" }} color={"action"} />}
        <Box
          sx={{
            mr: Icon ? 1 : 0,
            display: "flex",
            justifyContent: "space-between",
            width: "100%",
          }}
        >
          {contents.map(({ title, content }) => (
            <Box>
              <Typography variant={"body2"} color="textPrimary" fontWeight={500}>
                {title.map((element) =>
                  typeof element === "string" ? element : <MediaMetaElements element={element} />,
                )}
              </Typography>
              <Typography variant={"body2"} color={"text.secondary"}>
                {content.map((element) =>
                  typeof element === "string" ? element : <MediaMetaElements element={element} />,
                )}
              </Typography>
            </Box>
          ))}
        </Box>
      </StyledButtonBase>
    </>
  );
};
export default MediaMetaCard;
