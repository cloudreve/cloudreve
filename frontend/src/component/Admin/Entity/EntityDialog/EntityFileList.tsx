import { Box, Link, Table, TableBody, TableCell, TableContainer, TableHead, TableRow } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { File } from "../../../../api/dashboard";
import { FileType } from "../../../../api/explorer";
import { sizeToString } from "../../../../util";
import {
  NoWrapCell,
  NoWrapTableCell,
  NoWrapTypography,
  StyledTableContainerPaper,
} from "../../../Common/StyledComponents";
import TimeBadge from "../../../Common/TimeBadge";
import UserAvatar from "../../../Common/User/UserAvatar";
import FileTypeIcon from "../../../FileManager/Explorer/FileTypeIcon";
import FileDialog from "../../File/FileDialog/FileDialog";
import UserDialog from "../../User/UserDialog/UserDialog";

const EntityFileList = ({ files, userHashIDMap }: { files: File[]; userHashIDMap: Record<number, string> }) => {
  const { t } = useTranslation("dashboard");
  const [userDialogOpen, setUserDialogOpen] = useState(false);
  const [userDialogID, setUserDialogID] = useState<number>(0);
  const [fileDialogOpen, setFileDialogOpen] = useState(false);
  const [fileDialogID, setFileDialogID] = useState<number>(0);

  const userClicked = (uid: number) => (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    setUserDialogOpen(true);
    setUserDialogID(uid);
  };

  const fileClicked = (fid: number) => (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    setFileDialogOpen(true);
    setFileDialogID(fid);
  };
  return (
    <>
      <UserDialog open={userDialogOpen} onClose={() => setUserDialogOpen(false)} userID={userDialogID} />
      <FileDialog open={fileDialogOpen} onClose={() => setFileDialogOpen(false)} fileID={fileDialogID} />
      <TableContainer component={StyledTableContainerPaper} sx={{ maxHeight: "300px" }}>
        <Table size="small" stickyHeader sx={{ width: "100%", tableLayout: "fixed" }}>
          <TableHead>
            <TableRow>
              <NoWrapTableCell width={90}>{t("group.#")}</NoWrapTableCell>
              <NoWrapTableCell width={200}>{t("file.name")}</NoWrapTableCell>
              <NoWrapTableCell width={100}>{t("file.size")}</NoWrapTableCell>
              <NoWrapTableCell width={100}>{t("file.uploader")}</NoWrapTableCell>
              <NoWrapTableCell width={200}>{t("file.createdAt")}</NoWrapTableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {files?.map((option, index) => {
              return (
                <TableRow key={option.id} hover sx={{ cursor: "pointer" }} onClick={fileClicked(option.id ?? 0)}>
                  <TableCell>
                    <NoWrapTypography variant="inherit">{option.id}</NoWrapTypography>
                  </TableCell>
                  <NoWrapTableCell>
                    <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                      <FileTypeIcon name={option.name ?? ""} fileType={FileType.file} />
                      <NoWrapTypography variant="inherit">{option.name}</NoWrapTypography>
                    </Box>
                  </NoWrapTableCell>
                  <TableCell>
                    <NoWrapTypography variant="inherit">{sizeToString(option.size ?? 0)}</NoWrapTypography>
                  </TableCell>
                  <NoWrapTableCell>
                    <Box sx={{ display: "flex", alignItems: "center", gap: 1 }}>
                      <UserAvatar
                        sx={{ width: 24, height: 24 }}
                        overwriteTextSize
                        user={{
                          id: userHashIDMap[option.owner_id ?? 0] ?? "",
                          nickname: option.edges?.owner?.nick ?? "",
                          created_at: option.edges?.owner?.created_at ?? "",
                        }}
                      />
                      <NoWrapTypography variant="inherit">
                        <Link
                          sx={{
                            whiteSpace: "nowrap",
                            overflow: "hidden",
                            textOverflow: "ellipsis",
                          }}
                          onClick={userClicked(option.owner_id ?? 0)}
                          underline="hover"
                          href="#/"
                        >
                          {option.edges?.owner?.nick}
                        </Link>
                      </NoWrapTypography>
                    </Box>
                  </NoWrapTableCell>
                  <TableCell>
                    <NoWrapTypography variant="inherit">
                      <TimeBadge datetime={option.created_at ?? ""} variant="inherit" timeAgoThreshold={0} />
                    </NoWrapTypography>
                  </TableCell>
                </TableRow>
              );
            })}
            {!files?.length && (
              <TableRow>
                <NoWrapCell colSpan={5} align="center">
                  {t("file.noRecords")}
                </NoWrapCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </>
  );
};

export default EntityFileList;
