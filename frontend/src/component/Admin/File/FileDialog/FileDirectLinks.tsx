import { IconButton, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Tooltip } from "@mui/material";
import { useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { sizeToString } from "../../../../util";
import {
  NoWrapCell,
  NoWrapTableCell,
  NoWrapTypography,
  StyledTableContainerPaper,
} from "../../../Common/StyledComponents";
import TimeBadge from "../../../Common/TimeBadge";
import Delete from "../../../Icons/Delete";
import Open from "../../../Icons/Open";
import { FileDialogContext } from "./FileDialog";

const FileDirectLinks = () => {
  const { t } = useTranslation("dashboard");
  const { setFile, values } = useContext(FileDialogContext);

  const handleDelete = (id: number) => {
    setFile((prev) => ({
      ...prev,
      edges: {
        ...prev.edges,
        direct_links: prev.edges?.direct_links?.filter((link) => link.id !== id),
      },
    }));
  };

  const handleOpen = (id: number) => {
    window.open(values?.direct_link_map?.[id] ?? "", "_blank");
  };

  const linkId = useCallback(
    (id: number) => {
      const url = new URL(values?.direct_link_map?.[id] ?? "");
      return url.pathname;
    },
    [values?.direct_link_map],
  );

  return (
    <TableContainer component={StyledTableContainerPaper} sx={{ maxHeight: "300px" }}>
      <Table size="small" stickyHeader sx={{ width: "100%", tableLayout: "fixed" }}>
        <TableHead>
          <TableRow>
            <NoWrapTableCell width={90}>{t("group.#")}</NoWrapTableCell>
            <NoWrapTableCell width={200}>{t("file.name")}</NoWrapTableCell>
            <NoWrapTableCell width={100}>{t("file.downloads")}</NoWrapTableCell>
            <NoWrapTableCell width={100}>{t("file.speed")}</NoWrapTableCell>
            <NoWrapTableCell width={150}>{t("file.directLinkId")}</NoWrapTableCell>
            <NoWrapTableCell width={200}>{t("file.createdAt")}</NoWrapTableCell>
            <NoWrapTableCell width={100} align="right"></NoWrapTableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {values?.edges?.direct_links?.map((option, index) => {
            const lid = linkId(option.id);
            return (
              <TableRow key={option.id} hover>
                <TableCell>
                  <NoWrapTypography variant="inherit">{option.id}</NoWrapTypography>
                </TableCell>
                <TableCell>
                  <NoWrapTypography variant="inherit">{option.name ?? ""}</NoWrapTypography>
                </TableCell>
                <TableCell>
                  <NoWrapTypography variant="inherit">{option.downloads ?? 0}</NoWrapTypography>
                </TableCell>
                <TableCell>
                  <NoWrapTypography variant="inherit">
                    {option.speed ? `${sizeToString(option.speed)}/s` : "-"}
                  </NoWrapTypography>
                </TableCell>
                <TableCell>
                  <Tooltip title={lid}>
                    <NoWrapTypography variant="inherit">{lid}</NoWrapTypography>
                  </Tooltip>
                </TableCell>
                <TableCell>
                  <NoWrapTypography variant="inherit">
                    <TimeBadge datetime={option.created_at ?? ""} variant="inherit" timeAgoThreshold={0} />
                  </NoWrapTypography>
                </TableCell>
                <TableCell align="right">
                  <IconButton size="small" onClick={() => handleOpen(option.id)}>
                    <Open fontSize="small" />
                  </IconButton>
                  <IconButton size="small" onClick={() => handleDelete(option.id)}>
                    <Delete fontSize="small" />
                  </IconButton>
                </TableCell>
              </TableRow>
            );
          })}
          {!values?.edges?.direct_links?.length && (
            <TableRow>
              <NoWrapCell colSpan={6} align="center">
                {t("file.noRecords")}
              </NoWrapCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

export default FileDirectLinks;
