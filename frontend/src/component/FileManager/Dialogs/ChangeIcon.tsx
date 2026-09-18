import { useTranslation } from "react-i18next";
import { Box, Button, DialogContent, Skeleton, styled, Tab, Tabs, useTheme } from "@mui/material";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { useCallback, useEffect, useMemo, useState } from "react";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import { closeChangeIconDialog } from "../../../redux/globalStateSlice.ts";
import { LoadingButton } from "@mui/lab";
import { loadSiteConfig } from "../../../redux/thunks/site.ts";
import AutoHeight from "../../Common/AutoHeight.tsx";
import { ConfigLoadState } from "../../../redux/siteConfigSlice.ts";
import { applyIcon } from "../../../redux/thunks/file.ts";
import { FileManagerIndex } from "../FileManager.tsx";

interface EmojiSetting {
  [key: string]: string[];
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
  loading?: boolean;
}

function CustomTabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`simple-tabpanel-${index}`}
      aria-labelledby={`simple-tab-${index}`}
      {...other}
    >
      {value === index && children}
    </div>
  );
}

const StyledTab = styled(Tab)(({ theme }) => ({
  minWidth: 0,
  minHeight: 0,
  fontSize: theme.typography.h6.fontSize,
  padding: "8px 10px",
}));

const EmojiButton = styled(Button)(({ theme }) => ({
  minWidth: 0,
  padding: "0px 4px",
  fontSize: theme.typography.h6.fontSize,
}));

const SelectorBox = styled(Box)({
  display: "flex",
  flexWrap: "wrap",
});

const ChangeIcon = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();

  const [tabValue, setTabValue] = useState(0);
  const [loading, setLoading] = useState(false);

  const open = useAppSelector((state) => state.globalState.changeIconDialogOpen);
  const targets = useAppSelector((state) => state.globalState.changeIconDialogFile);
  const emojiStr = useAppSelector((state) => state.siteConfig.emojis.config.emoji_preset);
  const emojiStrLoaded = useAppSelector((state) => state.siteConfig.emojis.loaded);

  const emojiSetting = useMemo((): EmojiSetting => {
    if (!emojiStr) return {};
    try {
      return JSON.parse(emojiStr) as EmojiSetting;
    } catch (e) {
      console.warn("failed to parse emoji setting", e);
    }
    return {};
  }, [emojiStr]);

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  useEffect(() => {
    if (open && emojiStrLoaded != ConfigLoadState.Loaded) {
      dispatch(loadSiteConfig("emojis"));
    }
  }, [open]);

  const onClose = useCallback(() => {
    if (!loading) {
      dispatch(closeChangeIconDialog());
    }
  }, [dispatch, loading]);

  const onAccept = useCallback(
    (icon?: string) => async (e?: React.MouseEvent<HTMLElement>) => {
      if (e) {
        e.preventDefault();
      }

      if (!targets) return;

      setLoading(true);
      try {
        await dispatch(applyIcon(FileManagerIndex.main, targets, icon));
      } catch (e) {
      } finally {
        setLoading(false);
        dispatch(closeChangeIconDialog());
      }
    },
    [dispatch, targets, setLoading],
  );

  return (
    <DraggableDialog
      title={t("application:fileManager.customizeIcon")}
      showActions
      loading={loading}
      showCancel
      hideOk
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "sm",
      }}
      secondaryAction={
        <LoadingButton onClick={onAccept()} loading={loading} color="primary">
          <span>{t("application:modals.resetToDefault")}</span>
        </LoadingButton>
      }
    >
      <DialogContent>
        <AutoHeight>
          <Box sx={{ width: "100%" }}>
            <Box sx={{ borderBottom: 1, borderColor: "divider" }}>
              <Tabs
                sx={{ minHeight: 0 }}
                value={tabValue}
                variant="scrollable"
                scrollButtons="auto"
                onChange={handleTabChange}
              >
                {emojiStrLoaded ? (
                  Object.keys(emojiSetting).map((key) => <StyledTab label={key} key={key} />)
                ) : (
                  <StyledTab label={<Skeleton sx={{ minWidth: "20px" }} />} />
                )}
              </Tabs>
            </Box>
            <Box sx={{ maxHeight: "200px", overflowY: "auto" }}>
              {emojiStrLoaded ? (
                Object.keys(emojiSetting).map((key, index) => (
                  <CustomTabPanel value={tabValue} index={index}>
                    <SelectorBox>
                      {emojiSetting[key].map((emoji) => (
                        <EmojiButton onClick={onAccept(emoji)}>{emoji}</EmojiButton>
                      ))}
                    </SelectorBox>
                  </CustomTabPanel>
                ))
              ) : (
                <CustomTabPanel value={tabValue} index={0}>
                  <SelectorBox>
                    {[...Array(50).keys()].map(() => (
                      <EmojiButton disabled>
                        <Skeleton sx={{ minWidth: "20px" }} />
                      </EmojiButton>
                    ))}
                  </SelectorBox>
                </CustomTabPanel>
              )}
            </Box>
          </Box>
        </AutoHeight>
      </DialogContent>
    </DraggableDialog>
  );
};
export default ChangeIcon;
