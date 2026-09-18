import { Icon } from "@iconify/react/dist/iconify.js";
import {
  Box,
  IconButton,
  InputAdornment,
  ListItemText,
  Table,
  TableBody,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import { styled, useTheme } from "@mui/material/styles";
import { memo, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  DenseFilledTextField,
  DenseSelect,
  NoWrapCell,
  NoWrapTableCell,
  SecondaryButton,
  StyledTableContainerPaper,
} from "../../../Common/StyledComponents.tsx";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu.tsx";
import { builtInIcons, FileTypeIconSetting } from "../../../FileManager/Explorer/FileTypeIcon.tsx";
import Add from "../../../Icons/Add.tsx";
import Dismiss from "../../../Icons/Dismiss.tsx";
import HexColorInput from "../HexColorInput.tsx";

export interface FileIconListProps {
  config: string;
  onChange: (value: string) => void;
}

export enum IconType {
  Image = "imageUrl",
  Iconify = "iconifyName",
}

const StyledDenseSelect = styled(DenseSelect)(() => ({
  "& .MuiFilledInput-input": {
    "&:focus": {
      backgroundColor: "initial",
    },
  },
  backgroundColor: "initial",
}));

const IconPreview = ({ icon }: { icon: FileTypeIconSetting }) => {
  const theme = useTheme();
  const IconComponent = useMemo(() => {
    if (icon.icon) {
      return builtInIcons[icon.icon];
    }
  }, [icon.icon]);

  const iconColor = useMemo(() => {
    if (theme.palette.mode == "dark") {
      return icon.color_dark ?? icon.color ?? theme.palette.action.active;
    } else {
      return icon.color ?? theme.palette.action.active;
    }
  }, [icon.color, icon.color_dark, theme]);

  if (IconComponent) {
    return (
      <IconComponent
        sx={{
          color: iconColor,
          fontSize: 32,
        }}
      />
    );
  }

  // Handle iconify icons
  if (icon.iconify) {
    return <Icon icon={icon.iconify} color={iconColor} fontSize={32} />;
  }

  return (
    <Box
      component={icon.img ? "img" : "div"}
      sx={{
        width: "32px",
        height: "32px",
      }}
      src={icon.img}
    />
  );
};

const FileIconList = memo(({ config, onChange }: FileIconListProps) => {
  const { t } = useTranslation("dashboard");
  const configParsed = useMemo((): FileTypeIconSetting[] => JSON.parse(config), [config]);
  const [inputCache, setInputCache] = useState<{
    [key: number]: string | undefined;
  }>({});
  const [iconUrlCache, setIconUrlCache] = useState<{
    [key: number]: string | undefined;
  }>({});
  const [iconTypeCache, setIconTypeCache] = useState<{
    [key: number]: IconType | undefined;
  }>({});

  return (
    <Box>
      {configParsed?.length > 0 && (
        <TableContainer sx={{ mt: 1, maxHeight: 440 }} component={StyledTableContainerPaper}>
          <Table stickyHeader sx={{ width: "100%", tableLayout: "fixed" }} size="small">
            <TableHead>
              <TableRow>
                <NoWrapTableCell width={64}>{t("settings.icon")}</NoWrapTableCell>
                <NoWrapTableCell width={250}>{t("settings.iconUrl")}</NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("settings.iconColor")}</NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("settings.iconColorDark")}</NoWrapTableCell>
                <NoWrapTableCell width={200}>{t("settings.exts")}</NoWrapTableCell>
                <NoWrapTableCell width={64}></NoWrapTableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {configParsed.map((r, i) => {
                const currentIconType =
                  iconTypeCache[i] ?? (r.img ? IconType.Image : r.iconify ? IconType.Iconify : IconType.Image);
                const currentIconUrl =
                  iconUrlCache[i] ?? (currentIconType === IconType.Image ? r.img : r.iconify) ?? "";

                return (
                  <TableRow key={i} sx={{ "&:last-child td, &:last-child th": { border: 0 } }} hover>
                    <NoWrapCell>
                      <IconPreview icon={r} />
                    </NoWrapCell>
                    <NoWrapCell>
                      {!r.icon ? (
                        <DenseFilledTextField
                          fullWidth
                          required
                          sx={{
                            "& .MuiInputBase-root.MuiOutlinedInput-root": {
                              paddingLeft: "0",
                            },
                          }}
                          value={currentIconUrl}
                          onBlur={() => {
                            const newConfig = [...configParsed];
                            const updatedItem = { ...r };

                            if (currentIconType === IconType.Image) {
                              updatedItem.img = currentIconUrl;
                              updatedItem.iconify = undefined;
                            } else {
                              updatedItem.iconify = currentIconUrl;
                              updatedItem.img = undefined;
                            }

                            newConfig[i] = updatedItem;
                            onChange(JSON.stringify(newConfig));

                            setIconUrlCache({
                              ...iconUrlCache,
                              [i]: undefined,
                            });
                          }}
                          onChange={(e) =>
                            setIconUrlCache({
                              ...iconUrlCache,
                              [i]: e.target.value,
                            })
                          }
                          slotProps={{
                            input: {
                              startAdornment: (
                                <InputAdornment position="start">
                                  <StyledDenseSelect
                                    value={currentIconType}
                                    onChange={(e) => {
                                      const newType = e.target.value as IconType;
                                      setIconTypeCache({
                                        ...iconTypeCache,
                                        [i]: newType,
                                      });

                                      // Clear the URL cache when switching types
                                      setIconUrlCache({
                                        ...iconUrlCache,
                                        [i]: "",
                                      });

                                      // Update the config immediately
                                      const newConfig = [...configParsed];
                                      const updatedItem = { ...r };

                                      if (newType === IconType.Image) {
                                        updatedItem.img = "";
                                        updatedItem.iconify = undefined;
                                      } else {
                                        updatedItem.iconify = "";
                                        updatedItem.img = undefined;
                                      }

                                      newConfig[i] = updatedItem;
                                      onChange(JSON.stringify(newConfig));
                                    }}
                                    renderValue={(value) => (
                                      <Typography variant="body2">{t(`settings.${value}`)}</Typography>
                                    )}
                                    size={"small"}
                                    variant="filled"
                                  >
                                    <SquareMenuItem value={IconType.Image}>
                                      <ListItemText
                                        slotProps={{
                                          primary: { variant: "body2" },
                                        }}
                                      >
                                        {t(`settings.${IconType.Image}`)}
                                      </ListItemText>
                                    </SquareMenuItem>
                                    <SquareMenuItem value={IconType.Iconify}>
                                      <ListItemText
                                        slotProps={{
                                          primary: { variant: "body2" },
                                        }}
                                      >
                                        {t(`settings.${IconType.Iconify}`)}
                                      </ListItemText>
                                    </SquareMenuItem>
                                  </StyledDenseSelect>
                                </InputAdornment>
                              ),
                            },
                          }}
                        />
                      ) : (
                        t("settings.builtinIcon")
                      )}
                    </NoWrapCell>
                    <NoWrapCell>
                      {!r.icon && !r.iconify ? (
                        "-"
                      ) : (
                        <HexColorInput
                          currentColor={r.color ?? ""}
                          onColorChange={(color) =>
                            onChange(
                              JSON.stringify([
                                ...configParsed.slice(0, i),
                                {
                                  ...r,
                                  color: color,
                                },
                                ...configParsed.slice(i + 1),
                              ]),
                            )
                          }
                        />
                      )}
                    </NoWrapCell>
                    <NoWrapCell>
                      {!r.icon && !r.iconify ? (
                        "-"
                      ) : (
                        <HexColorInput
                          currentColor={r.color_dark ?? ""}
                          onColorChange={(color) => {
                            const updatedItem = { ...r };
                            if (color) {
                              updatedItem.color_dark = color;
                            } else {
                              delete updatedItem.color_dark;
                            }
                            onChange(
                              JSON.stringify([
                                ...configParsed.slice(0, i),
                                updatedItem,
                                ...configParsed.slice(i + 1),
                              ]),
                            )
                          }}
                        />
                      )}
                    </NoWrapCell>
                    <NoWrapCell>
                      <DenseFilledTextField
                        fullWidth
                        multiline
                        required
                        value={inputCache[i] ?? r.exts.join()}
                        onBlur={() => {
                          onChange(
                            JSON.stringify([
                              ...configParsed.slice(0, i),
                              {
                                ...r,
                                exts: inputCache[i]?.split(",") ?? r.exts,
                              },
                              ...configParsed.slice(i + 1),
                            ]),
                          );
                          setInputCache({
                            ...inputCache,
                            [i]: undefined,
                          });
                        }}
                        onChange={(e) =>
                          setInputCache({
                            ...inputCache,
                            [i]: e.target.value,
                          })
                        }
                      />
                    </NoWrapCell>
                    <NoWrapCell>
                      {!r.icon && (
                        <IconButton
                          onClick={() => onChange(JSON.stringify(configParsed.filter((_, index) => index !== i)))}
                          size={"small"}
                        >
                          <Dismiss fontSize={"small"} />
                        </IconButton>
                      )}
                    </NoWrapCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>
      )}
      <SecondaryButton
        variant={"contained"}
        startIcon={<Add />}
        sx={{ mt: 1 }}
        onClick={() =>
          onChange(
            JSON.stringify([
              ...configParsed,
              {
                img: "",
                exts: [],
              },
            ]),
          )
        }
      >
        {t("settings.addIcon")}
      </SecondaryButton>
    </Box>
  );
});

export default FileIconList;
