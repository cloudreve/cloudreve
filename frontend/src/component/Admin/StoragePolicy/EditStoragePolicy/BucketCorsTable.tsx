import { Table, TableBody, TableCell, TableContainer, TableHead, TableRow } from "@mui/material";
import { useTranslation } from "react-i18next";
import { NoWrapTableCell, StyledTableContainerPaper } from "../../../Common/StyledComponents";

export interface BucketCorsTableProps {
  exposedHeaders?: string[];
}

const BucketCorsTable = ({ exposedHeaders }: BucketCorsTableProps) => {
  const { t } = useTranslation("dashboard");
  return (
    <TableContainer sx={{ mt: 1, maxHeight: 440 }} component={StyledTableContainerPaper}>
      <Table stickyHeader sx={{ width: "100%", tableLayout: "fixed" }} size="small">
        <TableHead>
          <TableRow>
            <NoWrapTableCell width={64}>{t("policy.origin")}</NoWrapTableCell>
            <NoWrapTableCell width={200}>{t("policy.allowMethods")}</NoWrapTableCell>
            <NoWrapTableCell width={150}>{t("policy.allowHeaders")}</NoWrapTableCell>
            <NoWrapTableCell width={150}>{t("policy.exposeHeaders")}</NoWrapTableCell>
            <NoWrapTableCell width={250}>{t("policy.maxAge")}</NoWrapTableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          <TableRow>
            <TableCell>*</TableCell>
            <TableCell>
              {["GET", "POST", "PUT", "DELETE", "HEAD"].map((t) => (
                <div>{t}</div>
              ))}
            </TableCell>
            <TableCell>*</TableCell>
            <TableCell>{exposedHeaders?.map((h) => <div>{h}</div>)}</TableCell>
            <TableCell>3600</TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </TableContainer>
  );
};

export default BucketCorsTable;
