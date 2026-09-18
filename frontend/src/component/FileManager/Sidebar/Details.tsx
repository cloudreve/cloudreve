import { Box, Stack, Typography } from "@mui/material";
import { useEffect, useState } from "react";
import { FileResponse, FileType, Metadata } from "../../../api/explorer.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { loadFileThumb } from "../../../redux/thunks/file.ts";
import { FileManagerIndex } from "../FileManager.tsx";
import BasicInfo from "./BasicInfo.tsx";
import CustomProps from "./CustomProps/CustomProps.tsx";
import Data from "./Data.tsx";
import MediaInfo from "./MediaInfo.tsx";
import Tags from "./Tags.tsx";
import { DisplayOption } from "../ContextMenu/useActionDisplayOpt.ts";

export interface DetailsProps {
  inPhotoViewer?: boolean;
  target: FileResponse;
  setTarget: (target: FileResponse | undefined | null) => void;
  targetDisplayOptions?: DisplayOption;
}

const InfoBlock = ({ title, children }: { title: string; children: React.ReactNode }) => {
  return (
    <Box>
      <Typography variant={"body2"}>{title}</Typography>
      <Typography variant={"body2"} color={"text.secondary"}>
        {children}
      </Typography>
    </Box>
  );
};

const Details = ({ target, inPhotoViewer, setTarget, targetDisplayOptions }: DetailsProps) => {
  const dispatch = useAppDispatch();
  const [thumbSrc, setThumbSrc] = useState<string | null>(null);
  useEffect(() => {
    if (target.type == FileType.file && (!target.metadata || target.metadata[Metadata.thumbDisabled] === undefined)) {
      dispatch(loadFileThumb(FileManagerIndex.main, target)).then((src) => {
        setThumbSrc(src);
      });
    }

    setThumbSrc(null);
  }, [target]);

  return (
    <Stack spacing={1}>
      {thumbSrc && !inPhotoViewer && (
        <Box
          onError={() => {
            setThumbSrc(null);
          }}
          src={thumbSrc}
          sx={{
            borderRadius: "8px",
          }}
          component={"img"}
        />
      )}
      <MediaInfo target={target} />
      <CustomProps file={target} setTarget={setTarget} targetDisplayOptions={targetDisplayOptions} />
      <BasicInfo target={target} />
      <Tags target={target} />
      <Data target={target} />
    </Stack>
  );
};

export default Details;
