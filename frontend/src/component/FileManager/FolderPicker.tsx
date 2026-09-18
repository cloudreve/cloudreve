import { Box, styled, useMediaQuery, useTheme } from "@mui/material";
import Grid from "@mui/material/Grid2";
import { useAppSelector } from "../../redux/hooks.ts";
import { getFileLinkedUri } from "../../util"; // Grid version 2
import ContextMenu from "./ContextMenu/ContextMenu.tsx";
import { FileManager, FileManagerIndex } from "./FileManager.tsx";
import TreeNavigation from "./TreeView/TreeNavigation.tsx";

const StyledGridItem = styled(Grid)(() => ({
  display: "flex",
  height: "100%",
}));

export const useFolderSelector = () => {
  const currentPath = useAppSelector((state) => state.fileManager[FileManagerIndex.selector].pure_path);
  const selected = useAppSelector((state) => state.fileManager[FileManagerIndex.selector].selected);

  if (selected && Object.keys(selected).length > 0) {
    const selectedFile = selected[Object.keys(selected)[0]];
    return [selectedFile, getFileLinkedUri(selectedFile)] as const;
  }

  return [undefined, currentPath] as const;
};

export interface FolderPickerProps {
  disableSharedWithMe?: boolean;
  disableTrash?: boolean;
  initialPath?: string;
}

const FolderPicker = ({ disableSharedWithMe, disableTrash, initialPath }: FolderPickerProps) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const path = useAppSelector((state) => state.fileManager[FileManagerIndex.main].path);

  return (
    <Box sx={{ width: "100%", display: "flex" }}>
      <Grid container columnSpacing={2} sx={{ width: "100%", margin: "0 -4px" }}>
        <StyledGridItem
          size={{
            xs: 12,
            md: 2,
          }}
          sx={{
            overflowY: "auto",
            overflowX: "hidden",
          }}
        >
          <TreeNavigation
            index={FileManagerIndex.selector}
            disableSharedWithMe={disableSharedWithMe}
            disableTrash={disableTrash}
          />
        </StyledGridItem>
        <StyledGridItem
          size={{
            xs: 12,
            md: 10,
          }}
          sx={{ height: isMobile ? "initial" : "100%" }}
        >
          <FileManager index={FileManagerIndex.selector} initialPath={initialPath ?? path} skipRender={isMobile} />
        </StyledGridItem>
        <ContextMenu fmIndex={FileManagerIndex.selector} />
      </Grid>
    </Box>
  );
};
export default FolderPicker;
