import {
  Box,
  IconButton,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
} from "@mui/material";
import CloseIcon from "@mui/icons-material/Close";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentWindow } from "@tauri-apps/api/window";
import CloudreveLogo from "../../common/CloudreveLogo";
import DrivesSection from "./DrivesSection";
import GeneralSection from "./GeneralSection";
import AboutSection from "./AboutSection";
import HardDrive from "../../common/icons/HardDrive";
import { default as SettingsIcon } from "../../common/icons/Settings";
import Info from "../../common/icons/Info";

type SettingsSection = "drives" | "general" | "about";

export default function Settings() {
  const { t } = useTranslation();
  const [activeSection, setActiveSection] = useState<SettingsSection>("drives");

  const sections = [
    { id: "drives" as const, icon: <HardDrive />, label: t("settings.drives") },
    { id: "general" as const, icon: <SettingsIcon />, label: t("settings.general") },
    { id: "about" as const, icon: <Info />, label: t("settings.about") },
  ];

  const renderContent = () => {
    switch (activeSection) {
      case "drives":
        return <DrivesSection />;
      case "general":
        return <GeneralSection />;
      case "about":
        return <AboutSection />;
      default:
        return <DrivesSection />;
    }
  };

  return (
    <Box
      sx={{
        height: "100vh",
        display: "flex",
        bgcolor: "background.paper",
        overflow: "hidden",
      }}
    >
      {/* Left Navigation */}
      <Box
        sx={{
          width: 200,
          borderRight: 1,
          borderColor: "divider",
          display: "flex",
          flexDirection: "column",
          bgcolor: (theme) =>
            theme.palette.mode === "light"
              ? theme.palette.grey[50]
              : theme.palette.grey[900],
        }}
      >
        {/* Title with drag region */}
        <Box
          data-tauri-drag-region
          sx={{
            px: 2,
            pt: 2,
            pb: 1,
            display: "flex",
            alignItems: "center",
            gap: 1.5,
          }}
        >
          <CloudreveLogo height={24} />
        </Box>

        {/* Navigation Items */}
        <List sx={{ px: 1 }} dense>
          {sections.map((section) => (
            <ListItemButton
              key={section.id}
              selected={activeSection === section.id}
              onClick={() => setActiveSection(section.id)}
              sx={{
                borderRadius: 1,
                mb: 0.5,
                "&.Mui-selected": {
                  bgcolor: "action.selected",
                },
              }}
            >
              <ListItemIcon sx={{ minWidth: 36 }}>{section.icon}</ListItemIcon>
              <ListItemText primary={section.label} />
            </ListItemButton>
          ))}
        </List>
      </Box>

      {/* Main Content */}
      <Box
        sx={{
          flex: 1,
          overflow: "auto",
          display: "flex",
          flexDirection: "column",
        }}
      >
        {/* Drag region for window */}
        <Box
          data-tauri-drag-region
          sx={{
            height: 32,
            flexShrink: 0,
            display: "flex",
            alignItems: "center",
            justifyContent: "flex-end",
            px: 0.5,
          }}
        >
          <IconButton
            size="small"
            onClick={() => getCurrentWindow().close()}
            sx={{
              // Keep the button itself draggable-region-free so clicks work.
              WebkitAppRegion: "no-drag",
              appRegion: "no-drag",
            }}
          >
            <CloseIcon fontSize="small" />
          </IconButton>
        </Box>
        {/* Content */}
        <Box sx={{ flex: 1, overflow: "auto", px: 3, pb: 3,pt:1 }}>
          {renderContent()}
        </Box>
      </Box>
    </Box>
  );
}
