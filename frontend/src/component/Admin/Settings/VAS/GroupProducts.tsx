import { Box, Table, TableBody, TableContainer, TableHead, TableRow, Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import { NoWrapCell, SecondaryButton, StyledTableContainerPaper } from "../../../Common/StyledComponents.tsx";
import Add from "../../../Icons/Add.tsx";

const GroupProducts = () => {
  const { t } = useTranslation("dashboard");

  return (
    <Box>
      <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 1 }}>
        <SecondaryButton variant="contained" startIcon={<Add />}>
          {t("settings.addGroupProduct")}
        </SecondaryButton>
      </Box>

      <TableContainer component={StyledTableContainerPaper}>
        <Table sx={{ width: "100%" }} size="small">
          <TableHead>
            <TableRow>
              <NoWrapCell>{t("settings.displayName")}</NoWrapCell>
              <NoWrapCell>{t("settings.price")}</NoWrapCell>
              <NoWrapCell>{t("settings.duration")}</NoWrapCell>
              <NoWrapCell>{t("settings.description")}</NoWrapCell>
              <NoWrapCell>{t("settings.actions")}</NoWrapCell>
            </TableRow>
          </TableHead>
          <TableBody>
            <TableRow>
              <NoWrapCell colSpan={5} align="center">
                <Typography variant="caption" color="text.secondary">
                  {t("application:setting.listEmpty")}
                </Typography>
              </NoWrapCell>
            </TableRow>
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  );
};

export default GroupProducts;
