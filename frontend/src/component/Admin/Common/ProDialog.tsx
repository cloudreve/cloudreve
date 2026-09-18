import {
  Alert,
  AlertTitle,
  Button,
  DialogContent,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Typography,
  styled,
} from "@mui/material";
import dayjs from "dayjs";
import { useCallback, useMemo } from "react";
import { Trans, useTranslation } from "react-i18next";
import DraggableDialog, { StyledDialogActions } from "../../Dialogs/DraggableDialog";
import CheckmarkCircleFilled from "../../Icons/CheckmarkCircleFilled";
import Gift from "../../Icons/Gift";
import { Code } from "../../Common/Code.tsx";

export interface ProDialogProps {
  open: boolean;
  onClose: () => void;
}

const features = [
  "shareLinkCollabration",
  "filePermission",
  "multipleStoragePolicy",
  "auditAndActivity",
  "vasService",
  "sso",
  "more",
];

const StyledButton = styled(Button)(({ theme }) => ({
  background: `linear-gradient(45deg, ${theme.palette.primary.main} 30%, ${theme.palette.primary.light} 90%)`,
  color: theme.palette.primary.contrastText,
  "&:hover": {
    background: `linear-gradient(45deg, ${theme.palette.primary.dark} 30%, ${theme.palette.primary.main} 90%)`,
  },
  transition: "all 300ms cubic-bezier(0.4, 0, 0.2, 1) !important",
}));

// TODO: fetch from cloudreve.org
const currentPromotion = {
  code: "LX8YL9QP0",
  discount: 15,
  start: "2026-03-14T02:00:00Z",
  end: "2026-06-14T02:00:00Z",
};

const ProDialog = ({ open, onClose }: ProDialogProps) => {
  const { t } = useTranslation("dashboard");
  const openMore = useCallback((e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    window.open("https://cloudreve.org/pro", "_blank");
  }, []);
  const showPromotion = useMemo(() => {
    const now = dayjs();
    return now >= dayjs(currentPromotion.start) && now <= dayjs(currentPromotion.end);
  }, []);
  return (
    <DraggableDialog
      title={t("pro.title")}
      dialogProps={{
        open,
        onClose,
        maxWidth: "sm",
        fullWidth: true,
      }}
    >
      <DialogContent>
        <Typography variant="body1" color="text.secondary">
          {t("pro.description")}
        </Typography>

        <Typography variant="body1" fontWeight={600} sx={{ mt: 2 }}>
          {t("pro.proInclude")}
        </Typography>

        <List dense>
          {features.map((feature) => (
            <ListItem key={feature}>
              <ListItemIcon
                sx={{
                  minWidth: "36px",
                }}
              >
                <CheckmarkCircleFilled color="primary" />
              </ListItemIcon>
              <ListItemText
                slotProps={{
                  primary: {
                    sx: {},
                    variant: "body1",
                  },
                }}
              >
                {t(`pro.${feature}`)}
              </ListItemText>
            </ListItem>
          ))}
        </List>
        {showPromotion && (
          <Alert
            iconMapping={{
              info: <Gift fontSize="inherit" />,
            }}
            severity="info"
            sx={{ mt: 2 }}
          >
            <AlertTitle>{t("pro.promotionTitle")}</AlertTitle>
            <Trans
              i18nKey="dashboard:pro.promotion"
              values={{
                code: currentPromotion.code,
                discount: currentPromotion.discount,
              }}
              components={[<Code />, <Typography component={"span"} fontWeight={600} />]}
            />
          </Alert>
        )}
      </DialogContent>
      <StyledDialogActions
        sx={{
          justifyContent: "flex-end",
        }}
      >
        <Button variant="outlined" color="primary" onClick={onClose}>
          {t("pro.later")}
        </Button>
        <StyledButton onClick={openMore} variant="contained" color="primary">
          {t("pro.learnMore")}
        </StyledButton>
      </StyledDialogActions>
    </DraggableDialog>
  );
};

export default ProDialog;
