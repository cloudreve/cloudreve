import { Add } from "@mui/icons-material";
import { Box, Stack } from "@mui/material";
import { useTranslation } from "react-i18next";
import { SecondaryButton } from "../../../Common/StyledComponents";

export interface PaymentProviderProps {}

const PaymentProviders: React.FC<PaymentProviderProps> = ({}) => {
  const { t } = useTranslation("dashboard");

  return (
    <Stack spacing={2}>
      <Box>
        <SecondaryButton variant="contained" startIcon={<Add />}>
          {t("settings.addPaymentProvider")}
        </SecondaryButton>
      </Box>
    </Stack>
  );
};

export default PaymentProviders;
