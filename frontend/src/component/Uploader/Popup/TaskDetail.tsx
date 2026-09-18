import React from "react";
import Grid from "@mui/material/Grid";
import TimeAgo from "timeago-react";
import Base, { Status } from "../core/uploader/base";
import { Trans, useTranslation } from "react-i18next";
import { sizeToString } from "../../../util";
import FileBadge from "../../FileManager/FileBadge.tsx";
import { FileType } from "../../../api/explorer.ts";

export interface TaskDetailProps {
  uploader: Base;
  error: string;
}

export default function TaskDetail({ uploader, error }: TaskDetailProps) {
  const { t } = useTranslation();
  const items = [
    {
      name: t("uploader.fileName"),
      value: uploader.task.name,
    },
    {
      name: t("uploader.fileSize"),
      value: `${sizeToString(uploader.task.size)} ${
        uploader.task.session && uploader.task.session.chunk_size > 0
          ? t("uploader.chunkDescription", {
              total: uploader.task.chunkProgress.length,
              size: sizeToString(uploader.task.session.chunk_size),
            })
          : t("uploader.noChunks")
      }`,
    },
    {
      name: t("uploader.storagePolicy"),
      value: uploader.task.policy.name,
    },
    {
      name: t("uploader.destination"),
      value: (
        <FileBadge
          sx={{ my: 0.5 }}
          variant={"outlined"}
          clickable
          simplifiedFile={{ path: uploader.task.dst, type: FileType.folder }}
        />
      ),
    },
    uploader.task.session
      ? {
          name: t("uploader.uploadSession"),
          value: (
            <>
              <Trans
                i18nKey="uploader.sessionExpiredIn"
                components={[
                  <TimeAgo
                    key={0}
                    datetime={uploader.task.session ? uploader.task.session.expires * 1000 : 0}
                    locale={t("timeAgoLocaleCode", {
                      ns: "common",
                    })}
                  />,
                ]}
              />
            </>
          ),
        }
      : null,
    uploader.status === Status.error
      ? {
          name: t("uploader.errorDetails"),
          value: error,
        }
      : null,
  ];
  return (
    <Grid container sx={{ fontSize: (theme) => theme.typography.body2.fontSize }}>
      {items.map((i) => (
        <>
          {i && (
            <Grid key={i.name} container xs={12}>
              <Grid item xs={3} sx={{ fontWeight: 700 }}>
                {i.name}
              </Grid>
              <Grid
                item
                xs={9}
                sx={{
                  color: (theme) => theme.palette.text.secondary,
                  wordBreak: "break-all",
                }}
              >
                {i.value}
              </Grid>
            </Grid>
          )}
        </>
      ))}
    </Grid>
  );
}
