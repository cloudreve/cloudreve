import { Box, IconButton, Tooltip, Typography } from "@mui/material";
import ArrowClockwiseFilled from "../Icons/ArrowClockwiseFilled.tsx";
import { useTranslation } from "react-i18next";
import PageTitle from "../../router/PageTitle.tsx";

export const PageTabQuery = "tab";

export interface PageHeaderProps {
  title: string;
  loading?: boolean;
  onRefresh?: () => void;
  skipChangingDocumentTitle?: boolean;
  secondaryAction?: React.ReactNode;
}

const PageHeader = ({ title, secondaryAction, onRefresh, loading, skipChangingDocumentTitle }: PageHeaderProps) => {
  const { t } = useTranslation();
  return (
    <Box sx={{ mb: 4 }}>
      <Box sx={{ display: "flex", alignItems: "center" }}>
        <Typography variant={"h4"} fontWeight={600}>
          {title}
        </Typography>
        {!skipChangingDocumentTitle && <PageTitle title={title} />}
        {onRefresh && (
          <Tooltip title={t("application:fileManager.refresh")}>
            <IconButton onClick={onRefresh} disabled={loading} sx={{ ml: 1 }}>
              <ArrowClockwiseFilled />
            </IconButton>
          </Tooltip>
        )}
        <Box sx={{ flexGrow: 1 }} />
        {secondaryAction && secondaryAction}
      </Box>
    </Box>
  );
};

export default PageHeader;
