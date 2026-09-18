import { Delete } from "@mui/icons-material";
import { Checkbox, IconButton, Table, TableBody, TableCell, TableContainer, TableHead, TableRow } from "@mui/material";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import {
  DenseFilledTextField,
  NoWrapCell,
  NoWrapTableCell,
  NoWrapTypography,
  StyledTableContainerPaper,
} from "../../../Common/StyledComponents";
import { FileDialogContext } from "./FileDialog";

const FileMetadata = () => {
  const { t } = useTranslation("dashboard");
  const { setFile, values } = useContext(FileDialogContext);
  const onKeyChange = (index: number, value: string) => {
    setFile((prev) => ({
      ...prev,
      edges: {
        ...prev.edges,
        metadata: prev.edges?.metadata?.map((item, i) => (i === index ? { ...item, name: value } : item)),
      },
    }));
  };
  const onValueChange = (index: number, value: string) => {
    setFile((prev) => ({
      ...prev,
      edges: {
        ...prev.edges,
        metadata: prev.edges?.metadata?.map((item, i) => (i === index ? { ...item, value: value } : item)),
      },
    }));
  };
  const onIsPublicChange = (index: number, value: boolean) => {
    setFile((prev) => ({
      ...prev,
      edges: {
        ...prev.edges,
        metadata: prev.edges?.metadata?.map((item, i) =>
          i === index ? { ...item, is_public: value ? true : undefined } : item,
        ),
      },
    }));
  };
  const handleDelete = (id: number) => {
    setFile((prev) => ({
      ...prev,
      edges: { ...prev.edges, metadata: prev.edges?.metadata?.filter((item) => item.id !== id) },
    }));
  };
  return (
    <TableContainer component={StyledTableContainerPaper} sx={{ maxHeight: "300px" }}>
      <Table size="small" stickyHeader sx={{ width: "100%", tableLayout: "fixed" }}>
        <TableHead>
          <TableRow>
            <NoWrapTableCell width={150}>{t("file.key")}</NoWrapTableCell>
            <NoWrapTableCell width={150}>{t("file.value")}</NoWrapTableCell>
            <NoWrapTableCell width={50}>{t("file.isPublic")}</NoWrapTableCell>
            <NoWrapTableCell width={90}>{t("group.#")}</NoWrapTableCell>
            <NoWrapTableCell width={60} align="right"></NoWrapTableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {values?.edges?.metadata?.map((option, index) => (
            <TableRow key={option.id} hover>
              <TableCell>
                <DenseFilledTextField
                  fullWidth
                  value={option.name}
                  required
                  onChange={(e) => onKeyChange(index, e.target.value)}
                />
              </TableCell>
              <TableCell>
                <DenseFilledTextField
                  fullWidth
                  value={option.value}
                  onChange={(e) => onValueChange(index, e.target.value)}
                />
              </TableCell>
              <TableCell>
                <Checkbox
                  size="small"
                  disableRipple
                  color="primary"
                  checked={!!option.is_public}
                  onChange={(e) => onIsPublicChange(index, e.target.checked)}
                />
              </TableCell>
              <TableCell>
                <NoWrapTypography variant="inherit">{option.id}</NoWrapTypography>
              </TableCell>
              <TableCell align="right">
                <IconButton size="small" onClick={() => handleDelete(option.id)}>
                  <Delete fontSize="small" />
                </IconButton>
              </TableCell>
            </TableRow>
          ))}
          {!values?.edges?.metadata?.length && (
            <TableRow>
              <NoWrapCell colSpan={6} align="center">
                {t("file.noMetadata")}
              </NoWrapCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

export default FileMetadata;
