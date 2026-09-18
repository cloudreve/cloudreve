import { Stack, useMediaQuery, useTheme } from "@mui/material";
import Breadcrumb from "./Breadcrumb.tsx";
import TopActions from "./TopActions.tsx";
import { RadiusFrame } from "../../Frame/RadiusFrame.tsx";
import TopActionsSecondary from "./TopActionsSecondary.tsx";
import { SearchIndicator } from "../Search/SearchIndicator.tsx";

const NavHeader = () => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  return (
    <Stack
      direction={"row"}
      spacing={1}
      sx={{
        px: isMobile ? 2 : "initial",
      }}
    >
      <RadiusFrame
        sx={{
          flexGrow: 1,
          p: 0.5,
          overflow: "hidden",
          display: "flex",
        }}
        withBorder
      >
        <Breadcrumb />
        <SearchIndicator />
      </RadiusFrame>
      {!isMobile && (
        <RadiusFrame>
          <TopActionsSecondary />
        </RadiusFrame>
      )}
      <RadiusFrame>
        <TopActions />
      </RadiusFrame>
    </Stack>
  );
};

export default NavHeader;
