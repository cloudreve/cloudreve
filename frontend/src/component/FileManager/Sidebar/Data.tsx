import { Link, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography } from "@mui/material";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { EntityType, FileResponse } from "../../../api/explorer.ts";
import { setVersionControlDialog } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { downloadSingleFile } from "../../../redux/thunks/download.ts";
import { sizeToString } from "../../../util";
import { NoWrapTableCell, StyledTableContainerPaper } from "../../Common/StyledComponents.tsx";
import TimeBadge from "../../Common/TimeBadge.tsx";

export interface DataProps {
  target: FileResponse;
}

export const EntityTypeText: Record<EntityType, string> = {
  [EntityType.thumbnail]: "application:fileManager.thumbnails",
  [EntityType.live_photo]: "application:fileManager.livePhoto",
  [EntityType.version]: "application:fileManager.version",
};

const Data = ({ target }: DataProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const downloadEntity = useCallback(
    (entityId: string) => {
      dispatch(downloadSingleFile(target, entityId));
    },
    [target, dispatch],
  );

  const versionSizes = useMemo(() => {
    let size = 0;
    let notFound = true;
    target.extended_info?.entities?.forEach((entity) => {
      if (entity.type === EntityType.version) {
        size += entity.size;
        notFound = false;
      }
    });

    return notFound ? undefined : size;
  }, [target.extended_info?.entities]);

  if (!target.extended_info?.entities) {
    return null;
  }

  return (
    <>
      <Typography sx={{ pt: 1 }} color="textPrimary" fontWeight={500} variant={"subtitle1"}>
        {t("application:fileManager.data")}
      </Typography>
      <TableContainer component={StyledTableContainerPaper}>
        <Table sx={{ width: "100%" }} size="small">
          <TableHead>
            <TableRow>
              <NoWrapTableCell>{t("fileManager.type")}</NoWrapTableCell>
              <NoWrapTableCell>{t("fileManager.size")}</NoWrapTableCell>
              <NoWrapTableCell>{t("fileManager.createdAt")}</NoWrapTableCell>
              <NoWrapTableCell>{t("fileManager.storagePolicy")}</NoWrapTableCell>
              <NoWrapTableCell>{t("fileManager.actions")}</NoWrapTableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {versionSizes != undefined && (
              <TableRow hover sx={{ "&:last-child td, &:last-child th": { border: 0 } }}>
                <NoWrapTableCell component="th" scope="row">
                  {t("fileManager.versionEntity")}
                </NoWrapTableCell>
                <NoWrapTableCell>{sizeToString(versionSizes)}</NoWrapTableCell>
                <TableCell>-</TableCell>
                <TableCell>-</TableCell>
                <TableCell>
                  <Link
                    href={"#"}
                    onClick={() => dispatch(setVersionControlDialog({ open: true, file: target }))}
                    underline={"hover"}
                  >
                    {t("fileManager.manageVersions")}
                  </Link>
                </TableCell>
              </TableRow>
            )}
            {target.extended_info?.entities
              ?.filter((e) => e.type != EntityType.version)
              .map((e) => (
                <TableRow hover sx={{ "&:last-child td, &:last-child th": { border: 0 } }}>
                  <NoWrapTableCell component="th" scope="row">
                    {t(EntityTypeText[e.type as EntityType])}
                  </NoWrapTableCell>
                  <NoWrapTableCell>{sizeToString(e.size)}</NoWrapTableCell>
                  <TableCell>
                    <TimeBadge variant={"body2"} datetime={e.created_at} />
                  </TableCell>
                  <NoWrapTableCell>{e.storage_policy?.name}</NoWrapTableCell>
                  <NoWrapTableCell>
                    <Link href={"#"} underline={"hover"} onClick={() => downloadEntity(e.id)}>
                      {t("fileManager.download")}
                    </Link>
                  </NoWrapTableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </TableContainer>
    </>
  );
};

export default Data;
