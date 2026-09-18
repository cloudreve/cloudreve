import { Link, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Tooltip } from "@mui/material";
import { useContext, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link as RouterLink } from "react-router-dom";
import { EntityType } from "../../../../api/explorer";
import { sizeToString } from "../../../../util";
import {
  NoWrapCell,
  NoWrapTableCell,
  NoWrapTypography,
  StyledTableContainerPaper,
} from "../../../Common/StyledComponents";
import TimeBadge from "../../../Common/TimeBadge";
import { EncryptionStatusText } from "../../../FileManager/Sidebar/BasicInfo";
import { EntityTypeText } from "../../../FileManager/Sidebar/Data";
import EntityDialog from "../../Entity/EntityDialog/EntityDialog";
import UserDialog from "../../User/UserDialog/UserDialog";
import { FileDialogContext } from "./FileDialog";

const FileEntity = () => {
  const { t } = useTranslation("dashboard");
  const { setFile, values } = useContext(FileDialogContext);
  const [userDialogOpen, setUserDialogOpen] = useState(false);
  const [userDialogId, setUserDialogId] = useState<number | null>(null);
  const [entityDialogOpen, setEntityDialogOpen] = useState(false);
  const [entityDialogId, setEntityDialogId] = useState<number | null>(null);

  const handleUserDialogOpen = (id: number) => (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    setUserDialogId(id);
    setUserDialogOpen(true);
  };

  const handleEntityDialogOpen = (id: number) => (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    setEntityDialogId(id);
    setEntityDialogOpen(true);
  };
  return (
    <TableContainer component={StyledTableContainerPaper} sx={{ maxHeight: "300px" }}>
      <UserDialog open={userDialogOpen} onClose={() => setUserDialogOpen(false)} userID={userDialogId ?? undefined} />
      <EntityDialog
        open={entityDialogOpen}
        onClose={() => setEntityDialogOpen(false)}
        entityID={entityDialogId ?? undefined}
      />
      <Table size="small" stickyHeader sx={{ width: "100%", tableLayout: "fixed" }}>
        <TableHead>
          <TableRow>
            <NoWrapTableCell width={90}>{t("group.#")}</NoWrapTableCell>
            <NoWrapTableCell width={100}>{t("file.blobType")}</NoWrapTableCell>
            <NoWrapTableCell width={100}>{t("file.size")}</NoWrapTableCell>
            <NoWrapTableCell width={100}>{t("file.storagePolicy")}</NoWrapTableCell>
            <NoWrapTableCell width={200}>{t("file.source")}</NoWrapTableCell>
            <NoWrapTableCell width={100}>{t("application:fileManager.encryption")}</NoWrapTableCell>
            <NoWrapTableCell width={200}>{t("file.createdAt")}</NoWrapTableCell>
            <NoWrapTableCell width={150}>{t("file.creator")}</NoWrapTableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {values?.edges?.entities?.map((option, index) => (
            <TableRow key={option.id} hover sx={{ cursor: "pointer" }} onClick={handleEntityDialogOpen(option.id ?? 0)}>
              <TableCell>
                <NoWrapTypography variant="inherit">{option.id}</NoWrapTypography>
              </TableCell>
              <TableCell>
                <NoWrapTypography variant="inherit">
                  {t(EntityTypeText[option.type ?? EntityType.version])}
                </NoWrapTypography>
              </TableCell>
              <TableCell>
                <NoWrapTypography variant="inherit">{sizeToString(option.size ?? 0)}</NoWrapTypography>
              </TableCell>
              <TableCell>
                <NoWrapTypography variant="inherit">
                  <Link
                    sx={{
                      whiteSpace: "nowrap",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                    }}
                    component={RouterLink}
                    to={`/admin/policy/${option.edges?.storage_policy?.id}`}
                  >
                    {option.edges?.storage_policy?.name ?? ""}
                  </Link>
                </NoWrapTypography>
              </TableCell>
              <TableCell>
                <Tooltip title={option.source ?? ""}>
                  <NoWrapTypography variant="inherit">{option.source ?? ""}</NoWrapTypography>
                </Tooltip>
              </TableCell>
              <TableCell>
                <EncryptionStatusText
                  flexWrap={false}
                  simplified
                  status={{
                    status: option.props?.encrypt_metadata?.algorithm ? "full" : "none",
                    cipher: option.props?.encrypt_metadata?.algorithm
                      ? [option.props?.encrypt_metadata?.algorithm]
                      : [],
                  }}
                />
              </TableCell>
              <TableCell>
                <NoWrapTypography variant="inherit">
                  <TimeBadge datetime={option.created_at ?? ""} variant="inherit" timeAgoThreshold={0} />
                </NoWrapTypography>
              </TableCell>
              <TableCell>
                <NoWrapTypography variant="inherit">
                  <Link
                    underline="hover"
                    sx={{
                      whiteSpace: "nowrap",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                    }}
                    href="#/"
                    onClick={handleUserDialogOpen(option.edges?.user?.id ?? 0)}
                  >
                    {option.edges?.user?.nick ?? ""}
                  </Link>
                </NoWrapTypography>
              </TableCell>
            </TableRow>
          ))}
          {!values?.edges?.entities?.length && (
            <TableRow>
              <NoWrapCell colSpan={6} align="center">
                {t("file.noEntities")}
              </NoWrapCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

export default FileEntity;
