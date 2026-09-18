import { Box, Stack, Typography, useTheme } from "@mui/material";
import { grey } from "@mui/material/colors";
import { useSnackbar } from "notistack";
import { lazy, Suspense, useEffect, useMemo, useRef, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { testNode, upsertNode } from "../../../../api/api";
import { DownloaderProvider, Node, NodeStatus, NodeType } from "../../../../api/dashboard";
import { useAppDispatch } from "../../../../redux/hooks";
import { randomString } from "../../../../util";
import FacebookCircularProgress from "../../../Common/CircularProgress";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar";
import { DenseFilledTextField, SecondaryButton } from "../../../Common/StyledComponents";
import DraggableDialog from "../../../Dialogs/DraggableDialog";
import SettingForm from "../../../Pages/Setting/SettingForm";
import { Code } from "../../../Common/Code.tsx";
import { EndpointInput } from "../../Common/EndpointInput";
import { NoMarginHelperText } from "../../Settings/Settings";
const MonacoEditor = lazy(() => import("../../../Viewers/CodeViewer/MonacoEditor"));

export interface NewNodeDialogProps {
  open: boolean;
  onClose: () => void;
}

const defaultNode: Node = {
  id: 0,
  name: "",
  type: NodeType.slave,
  status: NodeStatus.active,
  server: "",
  slave_key: "",
  capabilities: "",
  weight: 1,
  settings: {
    provider: DownloaderProvider.aria2,
    qbittorrent: {},
    aria2: {},
    interval: 5,
  },
  edges: {
    storage_policy: [],
  },
};

export const Step = ({ step, children }: { step: number; children: React.ReactNode }) => {
  return (
    <Box
      sx={{
        display: "flex",
        padding: "10px",
        transition: (theme) =>
          theme.transitions.create("background-color", {
            easing: theme.transitions.easing.sharp,
            duration: theme.transitions.duration.leavingScreen,
          }),
        "&:focus-within": {
          backgroundColor: (theme) => (theme.palette.mode == "dark" ? grey[900] : grey[100]),
        },
      }}
    >
      <Box sx={{ ml: "20px" }}>
        <Box
          sx={{
            width: "20px",
            fontSize: (t) => t.typography.body2.fontSize,
            height: "20px",
            backgroundColor: (theme) => theme.palette.primary.light,
            color: (theme) => theme.palette.primary.contrastText,
            textAlign: "center",
            borderRadius: " 50%",
          }}
        >
          {step}
        </Box>
      </Box>
      <Box sx={{ ml: "10px", mr: "20px", flexGrow: 1 }}>{children}</Box>
    </Box>
  );
};

export const NewNodeDialog = ({ open, onClose }: NewNodeDialogProps) => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const { enqueueSnackbar } = useSnackbar();
  const [loading, setLoading] = useState(false);
  const [node, setNode] = useState<Node>({ ...defaultNode });
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    if (open) {
      setNode({ ...defaultNode, slave_key: randomString(64) });
    }
  }, [open]);

  const handleSubmit = () => {
    if (!formRef.current?.checkValidity()) {
      formRef.current?.reportValidity();
      return;
    }

    setLoading(true);
    dispatch(upsertNode({ node }))
      .then((r) => {
        navigate(`/admin/node/${r.id}`);
        onClose();
      })
      .finally(() => {
        setLoading(false);
      });
  };

  const config = useMemo(() => {
    return `[System]
Mode = slave
Listen = :5212

[Slave]
Secret = ${node.slave_key}

; ${t("node.keepIfUpload")}
[CORS]
AllowOrigins = *
AllowMethods = OPTIONS,GET,POST
AllowHeaders = *
`;
  }, [t, node.slave_key]);

  const handleTest = () => {
    if (!formRef.current?.checkValidity()) {
      formRef.current?.reportValidity();
      return;
    }

    setLoading(true);
    dispatch(testNode({ node }))
      .then(() => {
        setLoading(false);
        enqueueSnackbar(t("node.testNodeSuccess"), { variant: "success", action: DefaultCloseAction });
      })
      .finally(() => setLoading(false));
  };

  return (
    <DraggableDialog
      onAccept={handleSubmit}
      loading={loading}
      title={t("node.addNewNode")}
      showActions
      showCancel
      dialogProps={{
        open,
        onClose,
        fullWidth: true,
        maxWidth: "sm",
      }}
    >
      <form ref={formRef} onSubmit={handleSubmit}>
        <Stack spacing={1}>
          <Step step={1}>
            <SettingForm lgWidth={12}>
              <Typography variant="body2" sx={{ mb: 1 }}>
                {t("node.nameTheNode")}
              </Typography>
              <DenseFilledTextField
                fullWidth
                required
                value={node.name}
                onChange={(e) => setNode({ ...node, name: e.target.value })}
              />
              <NoMarginHelperText>{t("node.nameNode")}</NoMarginHelperText>
            </SettingForm>
          </Step>
          <Step step={2}>
            <SettingForm lgWidth={12}>
              <Typography variant="body2" sx={{ mb: 1 }}>
                {t("node.runCrSlave")}
              </Typography>
              <Suspense fallback={<FacebookCircularProgress />}>
                <MonacoEditor
                  theme={theme.palette.mode === "dark" ? "vs-dark" : "vs"}
                  language="ini"
                  value={config}
                  height="200px"
                  minHeight="200px"
                  options={{
                    wordWrap: "on",
                    minimap: { enabled: false },
                    scrollBeyondLastLine: false,
                    readOnly: true,
                  }}
                />
              </Suspense>
              <NoMarginHelperText sx={{ mt: 1 }}>
                <Trans i18nKey="node.runCrWithConfig" ns="dashboard" components={[<Code />]} />
              </NoMarginHelperText>
            </SettingForm>
          </Step>
          <Step step={3}>
            <SettingForm lgWidth={12}>
              <Typography variant="body2" sx={{ mb: 1 }}>
                {t("node.inputServer")}
              </Typography>
              <EndpointInput
                variant={"outlined"}
                fullWidth
                required
                enforceProtocol
                value={node.server}
                onChange={(e) => setNode({ ...node, server: e.target.value })}
              />
              <NoMarginHelperText>{t("node.serverDes")}</NoMarginHelperText>
            </SettingForm>
          </Step>
          <Step step={4}>
            <SettingForm lgWidth={12}>
              <Typography variant="body2" sx={{ mb: 1 }}>
                {t("node.testButton")}
              </Typography>
              <SecondaryButton loading={loading} variant="contained" onClick={handleTest}>
                {t("node.testNode")}
              </SecondaryButton>
              <NoMarginHelperText sx={{ mt: 1 }}>
                <Trans i18nKey="node.hostHeaderHint" ns="dashboard" components={[<Code />]} />
              </NoMarginHelperText>
            </SettingForm>
          </Step>
        </Stack>
      </form>
    </DraggableDialog>
  );
};
