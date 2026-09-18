import { createTheme, CssBaseline, GlobalStyles, styled, ThemeProvider, useMediaQuery, useTheme } from "@mui/material";
import { grey } from "@mui/material/colors";
import { ThemeOptions } from "@mui/material/styles/createTheme";
import i18next from "i18next";
import { enqueueSnackbar, MaterialDesignContent, SnackbarProvider } from "notistack";
import { Suspense, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Outlet } from "react-router-dom";
import { useRegisterSW } from "virtual:pwa-register/react";
import FileIconSnackbar from "./component/Common/Snackbar/FileIconSnackbar.tsx";
import LoadingSnackbar from "./component/Common/Snackbar/LoadingSnackbar.tsx";
import { ServiceWorkerUpdateAction } from "./component/Common/Snackbar/snackbar.tsx";
import GlobalDialogs from "./component/Dialogs/GlobalDialogs.tsx";
import { GrowDialogTransition } from "./component/FileManager/Search/SearchPopup.tsx";
import Warning from "./component/Icons/Warning.tsx";
import { useAppSelector } from "./redux/hooks.ts";
import { changeThemeColor } from "./util";

export const applyThemeWithOverrides = (themeConfig: ThemeOptions): ThemeOptions => {
  return {
    ...themeConfig,
    shape: {
      ...themeConfig.shape,
      borderRadius: 12,
    },
    components: {
      MuiCssBaseline: {
        styleOverrides: {
          body: {
            overscrollBehavior: "none",
          },
        },
      },
      MuiTooltip: {
        defaultProps: {
          enterDelay: 500,
        },
      },
      MuiToggleButton: {
        styleOverrides: {
          root: {
            textTransform: "none",
          },
        },
      },
      MuiButton: {
        styleOverrides: {
          root: {
            textTransform: "none",
          },
        },
        defaultProps: {
          disableElevation: true,
        },
      },
      MuiAlert: {
        defaultProps: {
          iconMapping: {
            warning: <Warning color={"inherit"} />,
          },
        },
      },
      MuiListItemButton: {
        styleOverrides: {
          root: {
            borderRadius: 12,
          },
        },
      },
      MuiTab: {
        styleOverrides: {
          root: {
            textTransform: "none",
          },
        },
      },
      MuiSkeleton: {
        defaultProps: {
          animation: "wave",
        },
      },
      MuiMenu: {
        styleOverrides: {
          paper: {
            borderRadius: "8px",
          },
          list: {
            padding: "4px 0",
          },
        },
        defaultProps: {
          slotProps: {
            paper: {
              elevation: 3,
            },
          },
        },
      },
      MuiDialogContent: {
        styleOverrides: {
          root: {
            paddingTop: 0,
          },
        },
      },
      MuiMenuItem: {
        styleOverrides: {
          root: {
            borderRadius: "8px",
            margin: "0px 4px",
            paddingLeft: "8px",
            paddingRight: "8px",
          },
        },
      },
      MuiDialog: {
        defaultProps: {
          TransitionComponent: GrowDialogTransition,
        },
      },
      MuiFilledInput: {
        styleOverrides: {
          root: {
            "&::before, &::after": {
              borderBottom: "none",
            },
            "&:hover:not(.Mui-disabled, .Mui-error):before": {
              borderBottom: "none",
            },
            borderRadius: 12,
            // '&:hover:not(.Mui-disabled, .Mui-error):before': {
            //   borderBottom: '2px solid var(--TextField-brandBorderHoverColor)',
            // },
            // '&.Mui-focused:after': {
            //   borderBottom: '2px solid var(--TextField-brandBorderFocusedColor)',
            // },
          },
        },
      },
    },
  };
};

export const useGeneratedTheme = (preferedDark?: boolean, subTheme?: boolean) => {
  const themes = useAppSelector((state) => state.siteConfig.basic.config.themes);
  const defaultTheme = useAppSelector((state) => state.siteConfig.basic.config.default_theme);
  const preferredTheme = useAppSelector((state) => state.globalState.preferredTheme);
  let darkMode = useAppSelector((state) => state.globalState.darkMode);
  darkMode = darkMode;
  const prefersDarkMode = useMediaQuery("(prefers-color-scheme: dark)");
  const mode =
    preferedDark !== undefined
      ? preferedDark
        ? "dark"
        : "light"
      : darkMode === undefined
        ? prefersDarkMode
          ? "dark"
          : "light"
        : darkMode
          ? "dark"
          : "light";
  const theme = useMemo(() => {
    // Determine preferred theme
    var themeConfig = {} as ThemeOptions;
    if (themes) {
      try {
        const themeOptions = JSON.parse(themes) as themeOptions;
        themeConfig = getPreferredTheme(themeOptions, mode, preferredTheme, defaultTheme);
      } catch (e) {
        console.log("failed to parse theme config, using default", e);
      }
    }

    themeConfig = {
      ...themeConfig,
      palette: {
        ...themeConfig.palette,
        mode: mode,
      },
    };

    const t = createTheme(applyThemeWithOverrides(themeConfig));
    if (!subTheme) {
      changeThemeColor(themeConfig?.palette?.mode === "light" ? t.palette.grey[100] : t.palette.grey[900]);
    }
    return t;
  }, [prefersDarkMode, preferredTheme, defaultTheme, themes, darkMode]);

  return theme;
};

const removeI18nCache = () => {
  Object.keys(localStorage).forEach(function (key) {
    if (key && key.startsWith("i18next_res_")) {
      localStorage.removeItem(key);
    }
  });
};

export const App = () => {
  const theme = useGeneratedTheme();
  const { t } = useTranslation();

  const {
    offlineReady: [offlineReady, setOfflineReady],
    needRefresh: [needRefresh, setNeedRefresh],
    updateServiceWorker,
  } = useRegisterSW({
    onRegisterError(error) {
      console.log("SW registration error", error);
    },
  });

  useEffect(() => {
    if (needRefresh) {
      enqueueSnackbar({
        message: i18next.t("common:newVersionRefresh"),
        variant: "default",
        persist: true,
        action: ServiceWorkerUpdateAction(() => {
          updateServiceWorker(true);
          removeI18nCache();
        }),
      });
    }
  }, [needRefresh, updateServiceWorker]);

  return (
    <Suspense fallback={<div>Loading...</div>}>
      <ThemeProvider theme={theme}>
        <AppContent />
      </ThemeProvider>
    </Suspense>
  );
};

interface themeOptions {
  [key: string]: singleThemeOption;
}

interface singleThemeOption {
  light: ThemeOptions;
  dark?: ThemeOptions;
}

const getPreferredTheme = (
  opts: themeOptions,
  mode: "dark" | "light",
  preferredTheme?: string,
  defaultTheme?: string,
): ThemeOptions => {
  let themeConfig = {} as singleThemeOption;
  if (defaultTheme && opts[defaultTheme]) {
    themeConfig = opts[defaultTheme];
  }
  if (preferredTheme && opts[preferredTheme]) {
    themeConfig = opts[preferredTheme];
  }

  if (!themeConfig?.light) {
    themeConfig = Object.values(opts)[0];
  }

  if (mode === "dark" && themeConfig.dark) {
    return themeConfig.dark;
  }

  return themeConfig.light;
};

const StyledMaterialDesignContent = styled(MaterialDesignContent)(({ theme }) => ({
  "&.notistack-MuiContent": {
    borderRadius: 12,
  },
  "&.notistack-MuiContent-success": {
    backgroundColor: theme.palette.success.main,
  },
  "&.notistack-MuiContent-error": {
    backgroundColor: theme.palette.error.main,
  },
  "&.notistack-MuiContent-warning": {
    backgroundColor: theme.palette.warning.main,
  },
}));

const AppContent = () => {
  const title = useAppSelector((state) => state.siteConfig.basic.config.title);
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const scrollBar = {
    "&::-webkit-scrollbar-button": {
      width: 0,
      height: 0,
    },
    "&::-webkit-scrollbar-corner": {
      background: "0 0",
    },
    "&::-webkit-scrollbar-thumb": {
      borderRadius: 4,
      backgroundColor: "transparent",
    },
    "&::-webkit-scrollbar-track": {
      borderRadius: 4,
    },
    "&::-webkit-scrollbar-track:hover": {
      backgroundColor: theme.palette.mode == "light" ? grey[200] : grey[800],
    },
    "&::-webkit-scrollbar-thumb:hover": {
      backgroundColor: theme.palette.primary.main + "!important",
    },
    "& :hover::-webkit-scrollbar-thumb,:hover>:first-child::-webkit-scrollbar-thumb": {
      backgroundColor: theme.palette.mode == "light" ? grey[400] : grey[600],
    },
    "&::-webkit-scrollbar ": {
      width: 8,
      height: 8,
    },
  };

  useEffect(() => {
    const loader = document.getElementById("app-loader");
    if (loader) loader.style.display = "none";
  }, []);

  return (
    <>
      <CssBaseline />
      <GlobalStyles
        styles={() => ({
          html: {
            scrollbarWidth: isMobile ? "initial" : "thin",
            //scrollbarColor: theme.palette.action.selected + " transparent",
          },
          ...(isMobile ? undefined : scrollBar),
          body: {
            overflowY: isMobile ? "initial" : "hidden",
          },
          ".highlight-marker": {
            backgroundColor: "#ffc1079e",
            borderRadius: "4px",
            boxShadow: "0 0 0 2px #ffc1079e",
          },
        })}
      />
      <SnackbarProvider
        anchorOrigin={{
          horizontal: "left",
          vertical: "bottom",
        }}
        Components={{
          success: StyledMaterialDesignContent,
          error: StyledMaterialDesignContent,
          warning: StyledMaterialDesignContent,
          loading: LoadingSnackbar,
          default: StyledMaterialDesignContent,
          file: FileIconSnackbar,
        }}
      >
        <GlobalDialogs />
        <Outlet />
      </SnackbarProvider>
    </>
  );
};
