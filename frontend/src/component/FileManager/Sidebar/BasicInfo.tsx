import { Box, Link, Skeleton, Tooltip, Typography } from "@mui/material";
import dayjs from "dayjs";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getFileInfo, sendPatchViewSync } from "../../../api/api.ts";
import {
  EncryptionCipher,
  ExplorerView,
  FileResponse,
  FileType,
  FolderSummary,
  Metadata,
} from "../../../api/explorer.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import SessionManager from "../../../session/index.ts";
import { sizeToString } from "../../../util";
import CrUri from "../../../util/uri.ts";
import TimeBadge from "../../Common/TimeBadge.tsx";
import ShieldDismiss from "../../Icons/ShieldDismiss.tsx";
import ShieldLockFilled from "../../Icons/ShieldLockFilled.tsx";
import FileBadge from "../FileBadge.tsx";
import InfoRow from "./InfoRow.tsx";

export interface BasicInfoProps {
  target: FileResponse;
}

export interface EncryptionStatus {
  status: "full" | "partial" | "none";
  cipher: EncryptionCipher[];
}

export const cipherDisplayName = (cipher: EncryptionCipher): string => {
  switch (cipher) {
    case EncryptionCipher.aes256ctr:
      return "AES-256-CTR";
    default:
      return cipher;
  }
};

export const EncryptionStatusText = ({
  status,
  simplified = false,
  flexWrap = true,
}: {
  status: EncryptionStatus;
  simplified?: boolean;
  flexWrap?: boolean;
}) => {
  const { t } = useTranslation();
  const title = useMemo(() => {
    switch (status.status) {
      case "full":
        return t("application:fileManager.fullEncryption", {
          cipher: status.cipher.map(cipherDisplayName).join(", "),
        });
      case "partial":
        return t("application:fileManager.partialEncryption");
    }
    return t("application:fileManager.noEncryption");
  }, [status.status, t]);

  const tooltipTitle = useMemo(() => {
    if (simplified) {
      return title;
    }
    return status.status === "partial" ? t("application:fileManager.partialEncryptionDes") : "";
  }, [status.status, t, simplified]);

  return (
    <Tooltip title={tooltipTitle}>
      <Box sx={{ display: "flex", alignItems: "center", gap: 0.5, flexWrap: flexWrap ? "wrap" : "nowrap" }}>
        {status.status === "full" ? (
          <ShieldLockFilled
            sx={{
              width: "20px",
              height: "20px",
              color: (theme) => theme.palette.success.main,
            }}
          />
        ) : status.status === "partial" ? (
          <ShieldLockFilled
            sx={{
              width: "20px",
              height: "20px",
              color: (theme) => theme.palette.action.disabled,
            }}
          />
        ) : (
          <ShieldDismiss
            sx={{
              width: "20px",
              height: "20px",
              color: (theme) => theme.palette.action.disabled,
            }}
          />
        )}
        {!simplified && (
          <>
            <Typography variant={"body2"} color={"text.secondary"}>
              {title}
            </Typography>
          </>
        )}
      </Box>
    </Tooltip>
  );
};

const BasicInfo = ({ target }: BasicInfoProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const showEncryptionStatus = useAppSelector((state) => state.siteConfig?.explorer?.config?.show_encryption_status);
  const ftsEnabled = useAppSelector((state) => state.siteConfig?.explorer?.config?.full_text_search);

  // null: not valid, undefined: not loaded, FolderSummary: loaded
  const [folderSummary, setFolderSummary] = useState<FolderSummary | undefined | null>(null);
  useEffect(() => {
    setFolderSummary(null);
  }, [target]);

  const [viewSetting, setViewSetting] = useState<ExplorerView | undefined>(undefined);

  useEffect(() => {
    setViewSetting(target?.extended_info?.view);
  }, [target]);

  const isSymbolicLink = useMemo(() => {
    return !!(target.metadata && target.metadata[Metadata.share_redirect]);
  }, [target.metadata]);
  const fileType = useMemo(() => {
    let srcType = "";
    switch (target.type) {
      case FileType.file:
        srcType = t("fileManager.file");
        break;
      case FileType.folder:
        srcType = t("fileManager.folder");
        break;
      default:
        srcType = t("fileManager.file");
    }

    if (isSymbolicLink) {
      return t("fileManager.symbolicLink", { srcType });
    }

    return srcType;
  }, [target, isSymbolicLink, t]);

  const displaySize = useCallback(
    (size: number): string => sizeToString(size) + t("fileManager.bytes", { bytes: size.toLocaleString() }),
    [t],
  );

  const storagePolicy = useMemo(() => {
    if (target.extended_info) {
      if (!target.extended_info.storage_policy) {
        return t("fileManager.unset");
      }

      return target.extended_info.storage_policy.name;
    }
    return <Skeleton variant={"text"} width={75} />;
  }, [target.extended_info, t]);

  const targetCrUri = useMemo(() => {
    return new CrUri(target.path);
  }, [target]);

  const viewSyncEnabled = useMemo(() => {
    return !SessionManager.currentLoginOrNull()?.user?.disable_view_sync;
  }, [target]);

  const restoreParent = useMemo(() => {
    if (!target.metadata || !target.metadata[Metadata.restore_uri]) {
      return null;
    }
    return new CrUri(target.metadata[Metadata.restore_uri]);
  }, [target]);

  const getFolderSummary = useCallback(() => {
    setFolderSummary(undefined);
    dispatch(getFileInfo({ uri: target.path, folder_summary: true }))
      .then((res) => {
        setFolderSummary(res.folder_summary ?? null);
      })
      .catch(() => {
        setFolderSummary(null);
      });
  }, [target, setFolderSummary, dispatch]);

  const folderSize = useMemo(() => {
    if (!folderSummary) {
      return "";
    }

    const sizeText = displaySize(folderSummary.size);
    if (!folderSummary.completed) {
      return t("fileManager.moreThan", { text: sizeText });
    }
    return sizeText;
  }, [folderSummary, t]);

  const folderChildren = useMemo(() => {
    if (!folderSummary) {
      return "";
    }

    let files = folderSummary.files.toLocaleString();
    let folders = folderSummary.folders.toLocaleString();

    if (!folderSummary.completed) {
      files += "+";
      folders += "+";
    }

    return t("application:fileManager.folderChildren", {
      files,
      folders,
    });
  }, [folderSummary, t]);

  const encryptionStatus = useMemo(() => {
    if (target.extended_info) {
      const status: EncryptionStatus = { status: "none", cipher: [] };
      let encrypted = 0;
      target.extended_info.entities?.forEach((entity) => {
        if (entity.encrypted_with) {
          encrypted++;
          if (!status.cipher.includes(entity.encrypted_with)) {
            status.cipher.push(entity.encrypted_with);
          }
        }
      });

      if (encrypted > 0) {
        status.status = encrypted === target.extended_info.entities?.length ? "full" : "partial";
      }
      return <EncryptionStatusText status={status} />;
    }
    return <Skeleton variant={"text"} width={75} />;
  }, [target.extended_info, t]);

  const handleDeleteViewSetting = useCallback(() => {
    dispatch(sendPatchViewSync({ uri: target.path }))
      .then(() => {
        setViewSetting(undefined);
      })
      .catch((error) => {
        console.error("Failed to delete view setting:", error);
      });
  }, [target.path, dispatch]);

  return (
    <>
      <Typography sx={{ pt: 1 }} color="textPrimary" fontWeight={500} variant={"subtitle1"}>
        {t("application:fileManager.basicInfo")}
      </Typography>
      <InfoRow title={t("fileManager.type")} content={fileType} />
      <InfoRow
        title={t("fileManager.parentFolder")}
        content={
          <FileBadge
            clickable
            variant={"outlined"}
            sx={{ px: 1, mt: "2px" }}
            simplifiedFile={{
              path: targetCrUri.parent().toString(),
              type: FileType.folder,
            }}
          />
        }
      />
      {restoreParent && (
        <InfoRow
          title={t("fileManager.originalLocation")}
          content={
            <FileBadge
              clickable
              variant={"outlined"}
              sx={{ px: 1, mt: "2px" }}
              simplifiedFile={{
                path: restoreParent.parent().toString(),
                type: FileType.folder,
              }}
            />
          }
        />
      )}
      {target.metadata && target.metadata[Metadata.expected_collect_time] && (
        <InfoRow
          title={t("application:fileManager.expires")}
          content={
            <TimeBadge
              variant={"body2"}
              datetime={dayjs.unix(parseInt(target.metadata[Metadata.expected_collect_time]))}
            />
          }
        />
      )}
      <InfoRow
        title={t("application:fileManager.createdAt")}
        content={<TimeBadge variant={"body2"} datetime={target.created_at} />}
      />
      <InfoRow
        title={t("application:fileManager.modifiedAt")}
        content={<TimeBadge variant={"body2"} datetime={target.updated_at} />}
      />
      {target.type == FileType.folder && !isSymbolicLink && (
        <>
          {!folderSummary && (
            <InfoRow
              title={t("fileManager.size")}
              content={
                folderSummary === undefined ? (
                  <Skeleton variant={"text"} width={75} />
                ) : (
                  <Link href={"#"} onClick={getFolderSummary} underline={"hover"}>
                    {t("fileManager.calculate")}
                  </Link>
                )
              }
            />
          )}
          {folderSummary && (
            <>
              <InfoRow title={t("fileManager.size")} content={folderSize} />
              <InfoRow title={t("fileManager.descendant")} content={folderChildren} />
              <InfoRow
                title={t("application:fileManager.statisticAt")}
                content={<TimeBadge variant={"body2"} datetime={folderSummary.calculated_at} />}
              />
            </>
          )}
        </>
      )}
      {target.type == FileType.file && (
        <>
          <InfoRow title={t("fileManager.size")} content={displaySize(target.size)} />
          <InfoRow
            title={t("application:fileManager.storageUsed")}
            content={
              target.extended_info ? (
                displaySize(target.extended_info.storage_used)
              ) : (
                <Skeleton variant={"text"} width={75} />
              )
            }
          />
          {showEncryptionStatus && encryptionStatus && (
            <InfoRow title={t("application:fileManager.encryption")} content={encryptionStatus} />
          )}
          {ftsEnabled && (
            <InfoRow
              title={t("application:fileManager.ftsStatus")}
              content={
                target.metadata && target.metadata[Metadata.full_text_index]
                  ? t("application:fileManager.ftsIndexed")
                  : t("application:fileManager.ftsNotIndexed")
              }
            />
          )}
        </>
      )}
      <InfoRow
        title={t("application:fileManager.createdAt")}
        content={<TimeBadge variant={"body2"} datetime={target.created_at} />}
      />
      <InfoRow
        title={t("application:fileManager.modifiedAt")}
        content={<TimeBadge variant={"body2"} datetime={target.updated_at} />}
      />
      {target.type == FileType.folder && viewSyncEnabled && target.owned && !restoreParent && !isSymbolicLink && (
        <InfoRow
          title={t("application:fileManager.viewSetting")}
          content={
            !!viewSetting ? (
              <>
                {t("application:fileManager.saved")}{" "}
                <Link
                  href={"#"}
                  onClick={(e) => {
                    e.preventDefault();
                    handleDeleteViewSetting();
                  }}
                  underline={"hover"}
                >
                  {t("application:fileManager.deleteViewSetting")}
                </Link>
              </>
            ) : (
              t("application:fileManager.notSet")
            )
          }
        />
      )}
    </>
  );
};

export default BasicInfo;
