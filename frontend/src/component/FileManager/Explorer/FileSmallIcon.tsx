import { Box, Fade } from "@mui/material";
import { memo, useCallback, useContext } from "react";
import { TransitionGroup } from "react-transition-group";
import { FileResponse } from "../../../api/explorer.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { fileIconClicked } from "../../../redux/thunks/file.ts";
import CheckmarkCircle from "../../Icons/CheckmarkCircle.tsx";
import CheckUnchecked from "../../Icons/CheckUnchecked.tsx";
import FileIcon from "./FileIcon.tsx";

import { FmIndexContext } from "../FmIndexContext.tsx";

export interface FileSmallIconProps {
  selected: boolean;
  file: FileResponse;
  loading?: boolean;
  ignoreHovered?: boolean;
  variant?: "list" | "grid";
}

const FileSmallIcon = memo(({ selected, variant, loading, file, ignoreHovered }: FileSmallIconProps) => {
  const dispatch = useAppDispatch();
  const fmIndex = useContext(FmIndexContext);
  const hovered = useAppSelector((state) => state.fileManager[fmIndex].multiSelectHovered[file.path]);
  const onIconClick = useCallback(
    (e: React.MouseEvent<HTMLElement>) => {
      if (!loading) {
        return dispatch(fileIconClicked(fmIndex, file, e));
      }
    },
    [file, loading, dispatch],
  );
  const isInList = variant === "list";
  return (
    <TransitionGroup onClick={onIconClick}>
      {!selected && (!hovered || ignoreHovered) && (
        <Fade>
          <FileIcon
            file={file}
            loading={loading}
            sx={
              isInList
                ? {
                    position: "absolute",
                    p: 0,
                  }
                : { position: "absolute" }
            }
          />
        </Fade>
      )}
      {!selected && hovered && !ignoreHovered && (
        <Fade>
          <Box sx={{ position: "absolute" }}>
            <CheckUnchecked
              sx={{
                width: isInList ? "20px" : "24px",
                height: "24px",
                mx: isInList ? "2px" : 2,
                my: isInList ? 0 : 1.5,
                position: "absolute",
              }}
              color={"action"}
            />
          </Box>
        </Fade>
      )}
      {selected && (
        <Fade>
          <Box sx={{ position: "absolute" }}>
            <CheckmarkCircle
              sx={{
                width: isInList ? "20px" : "24px",
                height: "24px",
                mx: isInList ? "2px" : 2,
                my: isInList ? 0 : 1.5,
              }}
              color={"primary"}
            />
          </Box>
        </Fade>
      )}
      <Box
        sx={{
          width: "24px",
          height: "24px",
          mx: isInList ? "2px" : 2,
          my: isInList ? 0 : 1.5,
        }}
      />
    </TransitionGroup>
  );
});

export default FileSmallIcon;
