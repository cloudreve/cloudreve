import { Link, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography } from "@mui/material";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { StyledTableContainerPaper } from "../../../Common/StyledComponents";
import EntityDialog from "../../Entity/EntityDialog/EntityDialog";

export interface BlobErrorsProps {
  privateState: any;
}
export const BlobErrors = ({ privateState }: BlobErrorsProps) => {
  const { t } = useTranslation("dashboard");
  const [openEntityDialogOpen, setOpenEntityDialogOpen] = useState(false);
  const [openTask, setOpenTask] = useState<number | undefined>(undefined);

  const openEntityDialog = (entityID: number) => (e: React.MouseEvent<HTMLElement>) => {
    e.stopPropagation();
    e.preventDefault();
    setOpenEntityDialogOpen(true);
    setOpenTask(entityID);
  };

  return (
    <TableContainer component={StyledTableContainerPaper} sx={{ maxHeight: 300 }}>
      <EntityDialog open={openEntityDialogOpen} onClose={() => setOpenEntityDialogOpen(false)} entityID={openTask} />
      <Table sx={{ width: "100%", tableLayout: "fixed" }} size="small">
        <TableHead>
          <TableRow>
            <TableCell width={70}>{t("task.blobID")}</TableCell>
            <TableCell width={60}>{t("task.retryIndex")}</TableCell>
            <TableCell width={400}>{t("common:error")}</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {privateState?.errors?.map((error: any, retryIndex: number) => [
            ...error.map((e: any, index: number) => [
              <TableRow hover key={index} sx={{ "&:last-child td, &:last-child th": { border: 0 } }}>
                <TableCell component="th" scope="row">
                  <Typography variant="body2" color="textSecondary">
                    <Link href="#/" onClick={openEntityDialog(e.id)}>
                      #{e.id}
                    </Link>
                  </Typography>
                </TableCell>
                <TableCell component="th" scope="row">
                  <Typography variant="body2" color="textSecondary">
                    {retryIndex}
                  </Typography>
                </TableCell>
                <TableCell component="th" scope="row">
                  <Typography variant="body2" color="textSecondary">
                    {e.error}
                  </Typography>
                </TableCell>
              </TableRow>,
            ]),
          ])}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

export default BlobErrors;
