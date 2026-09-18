import { Icon, ListItemIcon, Menu, MenuItem, Tooltip } from "@mui/material";
import React, { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { DenseDivider } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import Checkmark from "../../Icons/Checkmark.tsx";
import TextGrammarLighting from "../../Icons/TextGrammarLighting.tsx";
import DeleteOutlined from "../../Icons/DeleteOutlined.tsx";
import ArrowClockwise from "../../Icons/ArrowClockwise.tsx";
import ConcurrentOptionDialog from "./ConcurrentOptionDialog.tsx";
import SessionManager, { UserSettings } from "../../../session";
import UploadManager from "../core";

export interface MoreActionsProps {
  anchorEl: null | HTMLElement;
  onClose: () => void;
  uploadManager: UploadManager;
  useAvgSpeed: boolean;
  setUseAvgSpeed: (val: boolean) => void;
  filter: string;
  setFilter: (val: string) => void;
  sorter: string;
  setSorter: (val: string) => void;
  cleanFinished: () => void;
  retryFailed: () => void;
}

export default function MoreActions({
  anchorEl,
  onClose,
  uploadManager,
  useAvgSpeed,
  setUseAvgSpeed,
  filter,
  setFilter,
  sorter,
  setSorter,
  cleanFinished,
  retryFailed,
}: MoreActionsProps) {
  const { t } = useTranslation("application", { keyPrefix: "uploader" });
  const [concurrentDialog, setConcurrentDialog] = useState(false);
  const [overwrite, setOverwrite] = useState(SessionManager.getWithFallback(UserSettings.UploadOverwrite));
  const actionClicked = (next: () => void) => () => {
    onClose();
    next();
  };

  const toggleOverwrite = (val: boolean) => {
    SessionManager.set(UserSettings.UploadOverwrite, val);
    uploadManager.overwrite = val;
    setOverwrite(val);
  };

  const open = Boolean(anchorEl);
  const id = open ? "uploader-action-popover" : undefined;

  const listItems = useMemo(
    () => [
      {
        tooltip: t("overwriteTooltip"),
        onClick: () => toggleOverwrite(!overwrite),
        icon: overwrite ? <Checkmark /> : <Icon />,
        text: t("overwrite"),
        divider: true,
      },
      {
        tooltip: t("hideCompletedTooltip"),
        onClick: () => setFilter(filter === "default" ? "ongoing" : "default"),
        icon: filter !== "default" ? <Checkmark /> : <Icon />,
        text: t("hideCompleted"),
        divider: true,
      },
      {
        tooltip: t("addTimeAscTooltip"),
        onClick: () => setSorter("default"),
        icon: sorter === "default" ? <Checkmark /> : <Icon />,
        text: t("addTimeAsc"),
        divider: false,
      },
      {
        tooltip: t("addTimeDescTooltip"),
        onClick: () => setSorter("reverse"),
        icon: sorter === "reverse" ? <Checkmark /> : <Icon />,
        text: t("addTimeDesc"),
        divider: true,
      },
      {
        tooltip: t("showInstantSpeedTooltip"),
        onClick: () => setUseAvgSpeed(false),
        icon: useAvgSpeed ? <Icon /> : <Checkmark />,
        text: t("showInstantSpeed"),
        divider: false,
      },
      {
        tooltip: t("showAvgSpeedTooltip"),
        onClick: () => setUseAvgSpeed(true),
        icon: !useAvgSpeed ? <Icon /> : <Checkmark />,
        text: t("showAvgSpeed"),
        divider: true,
      },
      {
        tooltip: t("cleanCompletedTooltip"),
        onClick: () => cleanFinished(),
        icon: <DeleteOutlined />,
        text: t("cleanCompleted"),
        divider: false,
      },
      {
        tooltip: t("retryFailedTasksTooltip"),
        onClick: () => retryFailed(),
        icon: <ArrowClockwise />,
        text: t("retryFailedTasks"),
        divider: true,
      },
      {
        tooltip: t("setConcurrentTooltip"),
        onClick: () => setConcurrentDialog(true),
        icon: <TextGrammarLighting />,
        text: t("setConcurrent"),
        divider: false,
      },
    ],
    [useAvgSpeed, setUseAvgSpeed, sorter, setSorter, filter, setFilter, cleanFinished, overwrite],
  );

  const onConcurrentLimitSave = (val: string) => {
    const valNum = parseInt(val);
    if (valNum > 0) {
      SessionManager.set(UserSettings.ConcurrentLimit, valNum);
      uploadManager.changeConcurrentLimit(valNum);
    }
    setConcurrentDialog(false);
  };

  return (
    <>
      <Menu id={id} open={open} anchorEl={anchorEl} onClose={onClose}>
        {listItems.map((item) => [
          <Tooltip enterNextDelay={500} key={item.text} title={item.tooltip}>
            <MenuItem dense onClick={actionClicked(item.onClick)}>
              <ListItemIcon sx={{ minWidth: 38 }}>{item.icon}</ListItemIcon>
              {item.text}
            </MenuItem>
          </Tooltip>,
          item.divider && <DenseDivider />,
        ])}
      </Menu>
      <ConcurrentOptionDialog
        open={concurrentDialog}
        onClose={() => setConcurrentDialog(false)}
        onSave={onConcurrentLimitSave}
      />
    </>
  );
}
