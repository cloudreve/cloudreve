import { useAppDispatch, useAppSelector } from "../../redux/hooks.ts";
import { useTranslation } from "react-i18next";
import DraggableDialog, { StyledDialogContentText } from "./DraggableDialog.tsx";
import { useCallback, useMemo } from "react";
import { closeAggregatedErrorDialog } from "../../redux/globalStateSlice.ts";
import {
  DialogContent,
  DialogContentText,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from "@mui/material";
import { AppError } from "../../api/request.ts";
import { FileResponse, FileType } from "../../api/explorer.ts";
import FileBadge from "../FileManager/FileBadge.tsx";
import { StyledTableContainerPaper } from "../Common/StyledComponents.tsx";
import { CrUriPrefix } from "../../util/uri.ts";

interface ErrorTableProps {
  errors: {
    [key: string]: AppError;
  };
  files: {
    [key: string]: FileResponse;
  };
}

const ErrorTable = (props: ErrorTableProps) => {
  const { t } = useTranslation();
  return (
    <TableContainer component={StyledTableContainerPaper}>
      <Table sx={{ width: "100%" }} size="small">
        <TableHead>
          <TableRow>
            <TableCell>{t("common:object")}</TableCell>
            <TableCell>{t("common:error")}</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {Object.keys(props.errors).map((id) => (
            <TableRow hover key={id} sx={{ "&:last-child td, &:last-child th": { border: 0 } }}>
              <TableCell component="th" scope="row">
                {props.files[id] && <FileBadge sx={{ maxWidth: "250px" }} file={props.files[id]} />}
                {!props.files[id] && !id.startsWith(CrUriPrefix) && id}
                {!props.files[id] && id.startsWith(CrUriPrefix) && (
                  <FileBadge
                    sx={{ maxWidth: "250px" }}
                    simplifiedFile={{
                      type: FileType.file,
                      path: id,
                    }}
                  />
                )}
              </TableCell>
              <TableCell>{props.errors[id].message}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

const AggregatedErrorDetail = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const open = useAppSelector((state) => state.globalState.aggregatedErrorDialogOpen);
  const files = useAppSelector((state) => state.globalState.aggregatedErrorFile);
  const error = useAppSelector((state) => state.globalState.aggregatedError);
  const onClose = useCallback(() => {
    dispatch(closeAggregatedErrorDialog());
  }, [dispatch]);

  const [rootError, errors] = useMemo(() => {
    if (!error) {
      return [undefined, undefined];
    }

    let rootError = new AppError(error);
    const errors: { [key: string]: AppError } = {};
    Object.keys(error.aggregated_error ?? {}).forEach((key) => {
      const inner = error.aggregated_error?.[key];
      if (inner) {
        errors[key] = new AppError(inner);
      }
    });
    return [rootError, errors] as const;
  }, [error]);

  return (
    <DraggableDialog
      title={t("application:modals.errorDetailsTitle")}
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "sm",
      }}
    >
      <DialogContent>
        <Stack spacing={2}>
          <StyledDialogContentText>{rootError && rootError.message}</StyledDialogContentText>
          {files && errors && <ErrorTable errors={errors} files={files} />}
          {rootError && rootError.cid && (
            <DialogContentText variant={"caption"}>
              <code>{t("common:requestID", { id: rootError.cid })}</code>
            </DialogContentText>
          )}
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};

export default AggregatedErrorDetail;
