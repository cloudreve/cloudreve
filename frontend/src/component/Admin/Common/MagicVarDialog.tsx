import { DialogContent, Table, TableBody, TableCell, TableContainer, TableHead, TableRow } from "@mui/material";
import { useTranslation } from "react-i18next";
import { NoWrapTableCell } from "../../Common/StyledComponents.tsx";
import DraggableDialog from "../../Dialogs/DraggableDialog";

export interface MagicVar {
  name: string;
  value: string;
  example?: string;
}

export interface MagicVarDialogProps {
  open: boolean;
  onClose: () => void;
  vars: MagicVar[];
}

const MagicVarDialog = ({ open, onClose, vars }: MagicVarDialogProps) => {
  const { t } = useTranslation("dashboard");

  return (
    <DraggableDialog
      title={t("policy.magicVar.variable")}
      onAccept={onClose}
      dialogProps={{
        fullWidth: true,
        maxWidth: "md",
        open,
        onClose,
      }}
    >
      <DialogContent>
        <TableContainer sx={{ mt: 1, maxHeight: 440 }}>
          <Table stickyHeader sx={{ width: "100%", tableLayout: "fixed" }} size="small">
            <TableHead>
              <TableRow>
                <NoWrapTableCell width={350}>{t("policy.magicVar.variable")}</NoWrapTableCell>
                <NoWrapTableCell width={200}>{t("policy.magicVar.description")}</NoWrapTableCell>
                <NoWrapTableCell width={250}>{t("policy.magicVar.example")}</NoWrapTableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {vars.map((v, i) => (
                <TableRow key={i} sx={{ "&:last-child td, &:last-child th": { border: 0 } }} hover>
                  <TableCell>
                    <code>{v.name}</code>
                  </TableCell>
                  <TableCell>{t(v.value)}</TableCell>
                  <TableCell sx={{ wordBreak: "break-all" }}>{v.example}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </DialogContent>
    </DraggableDialog>
  );
};

export default MagicVarDialog;
