import { ChevronLeft } from "@mui/icons-material";
import { Box, Fade, IconButton, styled, useMediaQuery, useTheme } from "@mui/material";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { setDrawerOpen } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import Logo from "../../Common/Logo.tsx";

export const DrawerHeaderContainer = styled("div")(({ theme }) => ({
  display: "flex",
  alignItems: "center",
  padding: theme.spacing(0, 1),
  // necessary for content to be below app bar
  ...theme.mixins.toolbar,
  justifyContent: "flex-end",
}));

const DrawerHeader = ({ disabled }: { disabled?: boolean }) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const dispatch = useAppDispatch();
  const navigate = useNavigate();

  const [showCollapse, setShowCollapse] = useState(false);

  return (
    <DrawerHeaderContainer
      onMouseEnter={() => setShowCollapse(disabled ? false : true)}
      onMouseLeave={() => setShowCollapse(false)}
    >
      <Box sx={{ width: "100%", pl: 2, cursor: "pointer" }} onClick={() => navigate("/home")}>
        <Logo
          sx={{
            height: "auto",
            maxWidth: 160,
            maxHeight: 35,
            width: "100%",
            objectPosition: "left",
            objectFit: "contain",
          }}
        />
      </Box>
      {!isMobile && (
        <Box>
          <Fade in={showCollapse}>
            <IconButton onClick={() => dispatch(setDrawerOpen(false))}>
              <ChevronLeft />
            </IconButton>
          </Fade>
        </Box>
      )}
    </DrawerHeaderContainer>
  );
};

export default DrawerHeader;
