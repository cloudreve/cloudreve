import { Box, Collapse } from "@mui/material";
import { useAppSelector } from "../../../redux/hooks.ts";
import { RadiusFrame } from "../../Frame/RadiusFrame.tsx";
import ReadMeContent from "./ReadMeContent.tsx";

const ReadMeSideBar = () => {
  const readMeOpen = useAppSelector((state) => state.globalState.shareReadmeOpen);
  return (
    <Box>
      <Collapse in={readMeOpen} sx={{ height: "100%" }} orientation={"horizontal"} unmountOnExit timeout={"auto"}>
        <RadiusFrame
          sx={{
            width: "400px",
            height: "100%",
            ml: 1,
            overflow: "hidden",
            borderRadius: (theme) => theme.shape.borderRadius / 8,
          }}
          withBorder={true}
        >
          <ReadMeContent />
        </RadiusFrame>
      </Collapse>
    </Box>
  );
};

export default ReadMeSideBar;
