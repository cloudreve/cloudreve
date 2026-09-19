import { LoadingButton } from "@mui/lab";
import { Box, Checkbox, Dialog, DialogActions, DialogContent, DialogTitle, FormControlLabel, Skeleton, useTheme } from "@mui/material";
import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getAnnouncement, sendUpdateUserSetting } from "../../../api/api.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";

const MarkdownEditor = lazy(() => import("../../Viewers/MarkdownEditor/Editor.tsx"));

const Loading = () => (
  <Box>
    <Skeleton variant="text" width="100%" height={24} />
    <Skeleton variant="text" width="60%" height={24} />
  </Box>
);

// AnnouncementDialog fetches the current site announcement once on mount and
// shows it as a post-login modal. Dismissing with "don't show again" records
// the content server-side so the modal stays silent until the admin edits it.
const AnnouncementDialog = () => {
  const { t } = useTranslation();
  const theme = useTheme();
  const dispatch = useAppDispatch();
  const [content, setContent] = useState<string | undefined>();
  const [dontShow, setDontShow] = useState(false);
  const [closing, setClosing] = useState(false);

  useEffect(() => {
    let mounted = true;
    dispatch(getAnnouncement())
      .then((res) => {
        if (mounted && res.content) {
          setContent(res.content);
        }
      })
      .catch(() => {});
    return () => {
      mounted = false;
    };
  }, [dispatch]);

  const close = useCallback(() => {
    if (dontShow) {
      setClosing(true);
      dispatch(sendUpdateUserSetting({ dismiss_announcement: true }))
        .catch(() => {})
        .finally(() => setContent(undefined));
    } else {
      setContent(undefined);
    }
  }, [dispatch, dontShow]);

  if (!content) {
    return null;
  }

  return (
    <Dialog open onClose={close} maxWidth="sm" fullWidth>
      <DialogTitle>{t("application:announcement.title")}</DialogTitle>
      <DialogContent dividers>
        <Suspense fallback={<Loading />}>
          <MarkdownEditor
            displayOnly
            value={content}
            darkMode={theme.palette.mode === "dark"}
            readOnly={true}
            onChange={() => {}}
            initialValue={content}
          />
        </Suspense>
      </DialogContent>
      <DialogActions sx={{ justifyContent: "space-between" }}>
        <FormControlLabel
          control={<Checkbox checked={dontShow} onChange={(e) => setDontShow(e.target.checked)} />}
          label={t("application:announcement.dontShowAgain")}
        />
        <LoadingButton loading={closing} variant="contained" onClick={close}>
          {t("common:ok")}
        </LoadingButton>
      </DialogActions>
    </Dialog>
  );
};

export default AnnouncementDialog;
