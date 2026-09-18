import { useAppDispatch } from "../../../redux/hooks.ts";
import { ActionButton, ActionButtonGroup } from "./TopActions.tsx";
import ArrowSync from "../../Icons/ArrowSync.tsx";
import { styled, Tooltip } from "@mui/material";
import { useTranslation } from "react-i18next";
import { useContext, useState } from "react";
import MoreHorizontal from "../../Icons/MoreHorizontal.tsx";
import { refreshFileList } from "../../../redux/thunks/filemanager.ts";
import { bindMenu, bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import MoreActionMenu from "./MoreActionMenu.tsx";
import { FileManagerIndex } from "../FileManager.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";

const SpinArrowSync = styled(ArrowSync)(() => ({
  "@keyframes spin": {
    from: {
      transform: "rotate(0deg)",
    },
    to: {
      transform: "rotate(360deg)",
    },
  },
}));

const TopActionsSecondary = () => {
  const { t } = useTranslation();
  const fmIndex = useContext(FmIndexContext);
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(false);
  const morePopupState = usePopupState({
    variant: "popover",
    popupId: "moreActions",
  });
  const refresh = async () => {
    setLoading(true);
    await dispatch(refreshFileList(fmIndex));
    setLoading(false);
  };
  return (
    <>
      <ActionButtonGroup variant="outlined">
        <Tooltip enterDelay={200} title={t("application:fileManager.refresh")}>
          <ActionButton onClick={() => refresh()}>
            <SpinArrowSync
              sx={[
                loading && {
                  animation: "spin 1s linear 0.2s infinite",
                },
              ]}
              fontSize={"small"}
            />
          </ActionButton>
        </Tooltip>
        {fmIndex == FileManagerIndex.main && (
          <ActionButton {...bindTrigger(morePopupState)}>
            <MoreHorizontal fontSize={"small"} />
          </ActionButton>
        )}
      </ActionButtonGroup>
      <MoreActionMenu {...bindMenu(morePopupState)} />
    </>
  );
};

export default TopActionsSecondary;
