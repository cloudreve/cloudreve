import { Box, FormControl, FormHelperText } from "@mui/material";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { QueueType } from "../../../../api/dashboard.ts";
import { DenseFilledTextField } from "../../../Common/StyledComponents.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import SettingForm from "../../../Pages/Setting/SettingForm.tsx";

export interface QueueSettingDialogProps {
  open: boolean;
  onClose: () => void;
  queue: QueueType;
  settings: {
    [key: string]: string;
  };
  setSettings: (settings: { [key: string]: string }) => void;
}

const NoMarginHelperText = (props: any) => (
  <FormHelperText
    {...props}
    sx={{
      marginLeft: 0,
      marginRight: 0,
      marginTop: 0.5,
    }}
  />
);

const QueueSettingDialog = ({ open, onClose, queue, settings, setSettings }: QueueSettingDialogProps) => {
  const { t } = useTranslation("dashboard");
  const formRef = useRef<HTMLFormElement>(null);
  const [localSettings, setLocalSettings] = useState<{ [key: string]: string }>({});

  // Initialize local settings when dialog opens or queue changes
  useEffect(() => {
    if (open) {
      const queueSettings: { [key: string]: string } = {};
      const settingKeys = [
        "worker_num",
        "max_execution",
        "backoff_factor",
        "backoff_max_duration",
        "max_retry",
        "retry_delay",
      ];

      settingKeys.forEach((key) => {
        const fullKey = `queue_${queue}_${key}`;
        queueSettings[key] = settings[fullKey] || "";
      });

      setLocalSettings(queueSettings);
    }
  }, [open, queue, settings]);

  const handleSave = () => {
    if (formRef.current?.reportValidity()) {
      // Apply all settings at once
      const updatedSettings: { [key: string]: string } = {};
      Object.entries(localSettings).forEach(([key, value]) => {
        updatedSettings[`queue_${queue}_${key}`] = value as string;
      });

      setSettings(updatedSettings);
      onClose();
    }
  };

  const updateLocalSetting = (key: string, value: string) => {
    setLocalSettings((prev: { [key: string]: string }) => ({
      ...prev,
      [key]: value,
    }));
  };

  return (
    <DraggableDialog
      title={t("queue.editQueueSettings", {
        name: t(`queue.queueName_${queue}`),
      })}
      showActions
      showCancel
      onAccept={handleSave}
      dialogProps={{
        open,
        onClose,
        fullWidth: true,
        maxWidth: "sm",
      }}
    >
      <Box
        component={"form"}
        ref={formRef}
        sx={{ display: "flex", flexDirection: "column", gap: 2, mt: 1, px: 3, pb: 2 }}
      >
        <SettingForm title={t("queue.workerNum")} lgWidth={12}>
          <FormControl fullWidth>
            <DenseFilledTextField
              value={localSettings.worker_num || ""}
              onChange={(e) => updateLocalSetting("worker_num", e.target.value)}
              type="number"
              slotProps={{
                htmlInput: {
                  min: 1,
                },
              }}
              required
            />
            <NoMarginHelperText>{t("queue.workerNumDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>

        <SettingForm title={t("queue.maxExecution")} lgWidth={12}>
          <FormControl fullWidth>
            <DenseFilledTextField
              value={localSettings.max_execution || ""}
              onChange={(e) => updateLocalSetting("max_execution", e.target.value)}
              type="number"
              inputProps={{
                min: 1,
              }}
              required
            />
            <NoMarginHelperText>{t("queue.maxExecutionDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>

        <SettingForm title={t("queue.backoffFactor")} lgWidth={12}>
          <FormControl fullWidth>
            <DenseFilledTextField
              value={localSettings.backoff_factor || ""}
              onChange={(e) => updateLocalSetting("backoff_factor", e.target.value)}
              type="number"
              slotProps={{
                htmlInput: {
                  min: 1,
                  step: 0.1,
                },
              }}
              required
            />
            <NoMarginHelperText>{t("queue.backoffFactorDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>

        <SettingForm title={t("queue.backoffMaxDuration")} lgWidth={12}>
          <FormControl fullWidth>
            <DenseFilledTextField
              value={localSettings.backoff_max_duration || ""}
              onChange={(e) => updateLocalSetting("backoff_max_duration", e.target.value)}
              type="number"
              slotProps={{
                htmlInput: {
                  min: 1,
                },
              }}
              required
            />
            <NoMarginHelperText>{t("queue.backoffMaxDurationDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>

        <SettingForm title={t("queue.maxRetry")} lgWidth={12}>
          <FormControl fullWidth>
            <DenseFilledTextField
              value={localSettings.max_retry || ""}
              onChange={(e) => updateLocalSetting("max_retry", e.target.value)}
              type="number"
              slotProps={{
                htmlInput: {
                  min: 0,
                },
              }}
              required
            />
            <NoMarginHelperText>{t("queue.maxRetryDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>

        <SettingForm title={t("queue.retryDelay")} lgWidth={12}>
          <FormControl fullWidth>
            <DenseFilledTextField
              value={localSettings.retry_delay || ""}
              onChange={(e) => updateLocalSetting("retry_delay", e.target.value)}
              type="number"
              slotProps={{
                htmlInput: {
                  min: 0,
                },
              }}
              required
            />
            <NoMarginHelperText>{t("queue.retryDelayDes")}</NoMarginHelperText>
          </FormControl>
        </SettingForm>
      </Box>
    </DraggableDialog>
  );
};

export default QueueSettingDialog;
