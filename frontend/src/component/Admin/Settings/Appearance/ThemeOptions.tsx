import {
  Box,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  useTheme,
} from "@mui/material";
import { useSnackbar } from "notistack";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar";
import {
  NoWrapTableCell,
  SecondaryButton,
  StyledCheckbox,
  StyledTableContainerPaper,
} from "../../../Common/StyledComponents";
import Add from "../../../Icons/Add";
import Delete from "../../../Icons/Delete";
import Edit from "../../../Icons/Edit";
import HexColorInput from "../../FileSystem/HexColorInput.tsx";
import ThemeOptionEditDialog from "./ThemeOptionEditDialog";

export interface ThemeOptionsProps {
  value: string;
  onChange: (value: string) => void;
  defaultTheme: string;
  onDefaultThemeChange: (value: string) => void;
}

interface ThemeOption {
  id: string;
  config: {
    light: {
      palette: {
        primary: {
          main: string;
          light?: string;
          dark?: string;
        };
        secondary: {
          main: string;
          light?: string;
          dark?: string;
        };
      };
    };
    dark: {
      palette: {
        primary: {
          main: string;
          light?: string;
          dark?: string;
        };
        secondary: {
          main: string;
          light?: string;
          dark?: string;
        };
      };
    };
  };
}

const ThemeOptions = ({ value, onChange, defaultTheme, onDefaultThemeChange }: ThemeOptionsProps) => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const { enqueueSnackbar } = useSnackbar();
  const [options, setOptions] = useState<Record<string, ThemeOption["config"]>>({});
  const [editingOption, setEditingOption] = useState<{ id: string; config: ThemeOption["config"] } | null>(null);
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  useEffect(() => {
    try {
      const parsedOptions = JSON.parse(value);
      setOptions(parsedOptions);
    } catch (e) {
      setOptions({});
    }
  }, [value]);

  const handleSave = useCallback(
    (newOptions: Record<string, ThemeOption["config"]>) => {
      onChange(JSON.stringify(newOptions));
    },
    [onChange],
  );

  const handleDelete = useCallback(
    (id: string) => {
      // Prevent deleting the default theme
      if (id === defaultTheme) {
        enqueueSnackbar({
          message: t("settings.cannotDeleteDefaultTheme"),
          variant: "warning",
          action: DefaultCloseAction,
        });
        return;
      }

      const newOptions = { ...options };
      delete newOptions[id];
      handleSave(newOptions);
    },
    [options, handleSave, defaultTheme, enqueueSnackbar, t],
  );

  const handleEdit = useCallback(
    (id: string) => {
      setEditingOption({ id, config: options[id] });
      setIsDialogOpen(true);
    },
    [options],
  );

  const handleAdd = useCallback(() => {
    // Generate a new default theme option with a random color
    const randomColor = `#${Math.floor(Math.random() * 16777215).toString(16)}`;
    setEditingOption({
      id: randomColor,
      config: {
        light: {
          palette: {
            primary: { main: randomColor },
            secondary: { main: "#f50057" },
          },
        },
        dark: {
          palette: {
            primary: { main: randomColor },
            secondary: { main: "#f50057" },
          },
        },
      },
    });
    setIsDialogOpen(true);
  }, []);

  const handleDialogClose = useCallback(() => {
    setIsDialogOpen(false);
    setEditingOption(null);
  }, []);

  const handleDialogSave = useCallback(
    (id: string, newId: string, config: string) => {
      try {
        const parsedConfig = JSON.parse(config);
        const newOptions = { ...options };

        // If ID has changed (primary color changed), delete the old entry and create a new one
        if (id !== newId) {
          // Check if the new ID already exists
          if (newOptions[newId]) {
            enqueueSnackbar({
              message: t("settings.duplicateThemeColor"),
              variant: "warning",
              action: DefaultCloseAction,
            });
            return;
          }

          // If we're changing the ID of the default theme, update the default theme reference
          if (id === defaultTheme) {
            onDefaultThemeChange(newId);
          }

          delete newOptions[id];
        }

        newOptions[newId] = parsedConfig;
        handleSave(newOptions);
        setIsDialogOpen(false);
        setEditingOption(null);
      } catch (e) {
        // Handle error
        enqueueSnackbar({
          message: t("settings.invalidThemeConfig"),
          variant: "warning",
          action: DefaultCloseAction,
        });
      }
    },
    [options, handleSave, enqueueSnackbar, defaultTheme, onDefaultThemeChange, t],
  );

  const handleColorChange = useCallback(
    (id: string, type: "primary" | "secondary", mode: "light" | "dark", color: string) => {
      const newOptions = { ...options };

      if (type === "primary" && mode === "light") {
        // If changing the primary color (which is the ID), we need to create a new entry
        const newId = color;

        // Check if the new ID already exists
        if (newOptions[newId] && newId !== id) {
          enqueueSnackbar({
            message: t("settings.duplicateThemeColor"),
            variant: "warning",
            action: DefaultCloseAction,
          });
          return;
        }

        const config = { ...newOptions[id] };
        config[mode].palette[type].main = color;

        // Delete old entry and create new one with the updated ID
        delete newOptions[id];
        newOptions[newId] = config;

        // If we're changing the ID of the default theme, update the default theme reference
        if (id === defaultTheme) {
          onDefaultThemeChange(newId);
        }
      } else {
        // For other colors, just update the value
        newOptions[id][mode].palette[type].main = color;
      }

      handleSave(newOptions);
    },
    [options, handleSave, enqueueSnackbar, t, defaultTheme, onDefaultThemeChange],
  );

  const handleDefaultThemeChange = useCallback(
    (id: string) => {
      onDefaultThemeChange(id);
    },
    [onDefaultThemeChange],
  );

  const optionsArray = useMemo(() => {
    return Object.entries(options).map(([id, config]) => ({
      id,
      config,
    }));
  }, [options]);

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        {t("settings.themeOptions")}
      </Typography>
      <Typography variant="body2" color="text.secondary" paragraph>
        {t("settings.themeOptionsDes")}
      </Typography>

      {optionsArray.length > 0 && (
        <TableContainer component={StyledTableContainerPaper} sx={{ mt: 2 }}>
          <Table size="small" stickyHeader sx={{ width: "100%", tableLayout: "fixed" }}>
            <TableHead>
              <TableRow>
                <NoWrapTableCell width={50}>{t("settings.defaultTheme")}</NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("settings.primaryColor")}</NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("settings.secondaryColor")}</NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("settings.primaryColorDark")}</NoWrapTableCell>
                <NoWrapTableCell width={150}>{t("settings.secondaryColorDark")}</NoWrapTableCell>
                <NoWrapTableCell width={100} align="right"></NoWrapTableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {optionsArray.map((option) => (
                <TableRow key={option.id} hover>
                  <TableCell>
                    <StyledCheckbox
                      size="small"
                      checked={option.id === defaultTheme}
                      onChange={() => handleDefaultThemeChange(option.id)}
                    />
                  </TableCell>
                  <TableCell>
                    <HexColorInput
                      required
                      currentColor={option.config.light.palette.primary.main}
                      onColorChange={(color) => handleColorChange(option.id, "primary", "light", color)}
                    />
                  </TableCell>
                  <TableCell>
                    <HexColorInput
                      currentColor={option.config.light.palette.secondary.main}
                      onColorChange={(color) => handleColorChange(option.id, "secondary", "light", color)}
                    />
                  </TableCell>
                  <TableCell>
                    <HexColorInput
                      currentColor={option.config.dark.palette.primary.main}
                      onColorChange={(color) => handleColorChange(option.id, "primary", "dark", color)}
                    />
                  </TableCell>
                  <TableCell>
                    <HexColorInput
                      currentColor={option.config.dark.palette.secondary.main}
                      onColorChange={(color) => handleColorChange(option.id, "secondary", "dark", color)}
                    />
                  </TableCell>
                  <TableCell align="right">
                    <IconButton size="small" onClick={() => handleEdit(option.id)}>
                      <Edit fontSize="small" />
                    </IconButton>
                    <IconButton
                      size="small"
                      onClick={() => handleDelete(option.id)}
                      disabled={optionsArray.length === 1 || option.id === defaultTheme}
                    >
                      <Delete fontSize="small" />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <SecondaryButton variant="contained" startIcon={<Add />} onClick={handleAdd} sx={{ mt: 2 }}>
        {t("settings.addThemeOption")}
      </SecondaryButton>

      {editingOption && (
        <ThemeOptionEditDialog
          open={isDialogOpen}
          onClose={handleDialogClose}
          id={editingOption.id}
          config={JSON.stringify(editingOption.config, null, 2)}
          onSave={handleDialogSave}
        />
      )}
    </Box>
  );
};

export default ThemeOptions;
