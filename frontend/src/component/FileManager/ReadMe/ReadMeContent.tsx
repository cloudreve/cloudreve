import { Box, Skeleton, useTheme } from "@mui/material";
import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { getEntityContent } from "../../../redux/thunks/file.ts";
import { markdownImagePreviewHandler } from "../../../redux/thunks/viewer.ts";
import Header from "../Sidebar/Header.tsx";

const MarkdownEditor = lazy(() => import("../../Viewers/MarkdownEditor/Editor.tsx"));

const Loading = () => {
  return (
    <Box sx={{ p: 2 }}>
      <Skeleton variant="text" width="100%" height={24} />
      <Skeleton variant="text" width="40%" height={24} />
      <Skeleton variant="text" width="75%" height={24} />
      <Skeleton variant="text" width="85%" height={24} />
      <Skeleton variant="text" width="20%" height={24} />
    </Box>
  );
};

const ReadMeContent = () => {
  const theme = useTheme();
  const dispatch = useAppDispatch();
  const readMeTarget = useAppSelector((state) => state.globalState.shareReadmeTarget);
  const [loading, setLoading] = useState(true);
  const [value, setValue] = useState("");

  useEffect(() => {
    if (readMeTarget) {
      setLoading(true);
      dispatch(getEntityContent(readMeTarget))
        .then((res) => {
          const content = new TextDecoder().decode(res);
          setValue(content);
          setLoading(false);
        })
        .catch(() => {
          setLoading(false);
        });
    }
  }, [readMeTarget]);

  const imagePreviewHandler = useCallback(
    async (imageSource: string) => {
      return dispatch(markdownImagePreviewHandler(imageSource, readMeTarget?.path ?? ""));
    },
    [dispatch, readMeTarget],
  );

  return (
    <Box sx={{ display: "flex", flexDirection: "column", height: "100%" }}>
      <Header target={readMeTarget} variant={"readme"} />
      <Box
        sx={{
          width: "100%",
          bgcolor: "background.paper",
          borderTop: 1,
          borderColor: "divider",
          overflow: "auto",
        }}
      >
        {loading && <Loading />}
        {!loading && (
          <Suspense fallback={<Loading />}>
            <MarkdownEditor
              displayOnly
              value={value}
              darkMode={theme.palette.mode === "dark"}
              readOnly={true}
              onChange={() => {}}
              initialValue={value}
              imagePreviewHandler={imagePreviewHandler}
            />
          </Suspense>
        )}
      </Box>
    </Box>
  );
};

export default ReadMeContent;
