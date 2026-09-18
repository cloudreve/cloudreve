import { Icon } from "@iconify/react/dist/iconify.js";
import {
  alpha,
  Box,
  Grow,
  IconButton,
  ListItemIcon,
  ListItemText,
  Menu,
  styled,
  Typography,
  useTheme,
} from "@mui/material";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { CustomProps, CustomPropsType } from "../../../../api/explorer.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import { searchMetadata } from "../../../../redux/thunks/filemanager.ts";
import { copyToClipboard } from "../../../../util";
import Clipboard from "../../../Icons/Clipboard.tsx";
import DeleteOutlined from "../../../Icons/DeleteOutlined.tsx";
import MoreVertical from "../../../Icons/MoreVertical.tsx";
import Search from "../../../Icons/Search.tsx";
import { SquareMenuItem } from "../../ContextMenu/ContextMenu.tsx";
import { FileManagerIndex } from "../../FileManager.tsx";
import { StyledButtonBase } from "../MediaMetaCard.tsx";
import BooleanPropsItem from "./BooleanPropsContent.tsx";
import { CustomPropsItem } from "./CustomProps.tsx";
import LinkPropsContent from "./LinkPropsContent.tsx";
import MultiSelectPropsContent from "./MultiSelectPropsContent.tsx";
import NumberPropsContent from "./NumberPropsContent.tsx";
import RatingPropsItem from "./RatingPropsItem.tsx";
import SelectPropsContent from "./SelectPropsContent.tsx";
import TextPropsContent from "./TextPropsContent.tsx";
import UserPropsContent from "./UserPropsContent.tsx";

export interface CustomPropsCardProps {
  prop: CustomPropsItem;
  loading?: boolean;
  onPropPatch: (prop: CustomPropsItem) => void;
  onPropDelete?: (prop: CustomPropsItem) => void;
  readOnly?: boolean;
}

export interface PropsContentProps {
  prop: CustomPropsItem;
  onChange: (value: string) => void;
  loading?: boolean;
  readOnly?: boolean;
  backgroundColor?: string;
  fullSize?: boolean;
}

const PropsCard = styled(StyledButtonBase)(({ theme }) => ({
  flexDirection: "column",
  alignItems: "flex-start",
  gap: 9,
}));

export const getPropsContent = (
  prop: CustomPropsItem,
  onChange: (value: string) => void,
  loading?: boolean,
  readOnly?: boolean,
  fullSize?: boolean,
) => {
  switch (prop.props.type) {
    case CustomPropsType.text:
      return (
        <TextPropsContent prop={prop} onChange={onChange} loading={loading} readOnly={readOnly} fullSize={fullSize} />
      );
    case CustomPropsType.rating:
      return (
        <RatingPropsItem prop={prop} onChange={onChange} loading={loading} readOnly={readOnly} fullSize={fullSize} />
      );
    case CustomPropsType.number:
      return (
        <NumberPropsContent prop={prop} onChange={onChange} loading={loading} readOnly={readOnly} fullSize={fullSize} />
      );
    case CustomPropsType.boolean:
      return (
        <BooleanPropsItem prop={prop} onChange={onChange} loading={loading} readOnly={readOnly} fullSize={fullSize} />
      );
    case CustomPropsType.select:
      return (
        <SelectPropsContent prop={prop} onChange={onChange} loading={loading} readOnly={readOnly} fullSize={fullSize} />
      );
    case CustomPropsType.multi_select:
      return (
        <MultiSelectPropsContent
          prop={prop}
          onChange={onChange}
          loading={loading}
          readOnly={readOnly}
          fullSize={fullSize}
        />
      );
    case CustomPropsType.link:
      return (
        <LinkPropsContent prop={prop} onChange={onChange} loading={loading} readOnly={readOnly} fullSize={fullSize} />
      );
    default:
      return null;
  }
};

export const isCustomPropStrongMatch = (prop: CustomProps) => {
  return prop.type === CustomPropsType.rating || prop.type === CustomPropsType.number;
};

const CustomPropsCard = ({ prop, loading, onPropPatch, onPropDelete, readOnly }: CustomPropsCardProps) => {
  const { t } = useTranslation();
  const theme = useTheme();
  const dispatch = useAppDispatch();
  const [mouseOver, setMouseOver] = useState(false);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

  const handleMenuClose = () => {
    setAnchorEl(null);
  };

  const handleCopy = () => {
    const value = prop.value || "";
    copyToClipboard(value);
    handleMenuClose();
  };

  const handleSearch = () => {
    if (prop.value) {
      dispatch(
        searchMetadata(
          FileManagerIndex.main,
          `props:${prop.props.id}`,
          prop.value,
          false,
          isCustomPropStrongMatch(prop.props),
        ),
      );
    }
    handleMenuClose();
  };

  const handleDelete = () => {
    if (onPropDelete) {
      onPropDelete(prop);
    }
    handleMenuClose();
  };

  const Content = useMemo(() => {
    return getPropsContent(prop, (value) => onPropPatch({ ...prop, value }), loading, readOnly, true);
  }, [prop, loading, onPropPatch, readOnly]);

  return (
    <PropsCard onMouseEnter={() => setMouseOver(true)} onMouseLeave={() => setMouseOver(false)}>
      <Box sx={{ position: "relative", display: "flex", alignItems: "center", width: "100%", gap: 1 }}>
        <Grow in={mouseOver} unmountOnExit>
          <Box
            sx={{
              position: "absolute",
              transition: "opacity 250ms cubic-bezier(0.4, 0, 0.2, 1) 0ms",
              backgroundColor:
                theme.palette.mode === "light"
                  ? alpha(theme.palette.grey[100], 0.73)
                  : alpha(theme.palette.grey[900], 0.73),
              top: -4,
              right: -5,
            }}
          >
            <IconButton size="small" onClick={(e) => setAnchorEl(e.currentTarget)}>
              <MoreVertical />
            </IconButton>
          </Box>
        </Grow>
        {prop.props.icon && <Icon width={24} height={24} color={theme.palette.action.active} icon={prop.props.icon} />}
        <Typography variant={"body2"} color="textPrimary" fontWeight={500} sx={{ flexGrow: 1 }}>
          {prop.props.type === CustomPropsType.boolean ? Content : t(prop.props.name)}
        </Typography>
      </Box>

      {prop.props.type !== CustomPropsType.boolean && (
        <Typography variant={"body2"} color={"text.secondary"} sx={{ width: "100%" }}>
          {Content}
        </Typography>
      )}

      <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={handleMenuClose}>
        {prop.value && (
          <>
            <SquareMenuItem onClick={handleCopy} dense>
              <ListItemIcon>
                <Clipboard fontSize="small" />
              </ListItemIcon>
              <ListItemText>{t("application:fileManager.copyToClipboard")}</ListItemText>
            </SquareMenuItem>
            <SquareMenuItem onClick={handleSearch} dense>
              <ListItemIcon>
                <Search fontSize="small" />
              </ListItemIcon>
              <ListItemText>{t("application:fileManager.searchProperty")}</ListItemText>
            </SquareMenuItem>
          </>
        )}

        <SquareMenuItem onClick={handleDelete} dense disabled={readOnly}>
          <ListItemIcon>
            <DeleteOutlined fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t("application:fileManager.delete")}</ListItemText>
        </SquareMenuItem>
      </Menu>
    </PropsCard>
  );
};

export default CustomPropsCard;
