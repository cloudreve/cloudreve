import { User } from "../../../api/user.ts";
import UserAvatar from "../../Common/User/UserAvatar.tsx";
import SettingForm from "./SettingForm.tsx";
import { useTranslation } from "react-i18next";
import Edit from "../../Icons/Edit.tsx";
import { SecondaryButton } from "../../Common/StyledComponents.tsx";
import { ListItemIcon, ListItemText, Menu } from "@mui/material";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import React, { useState } from "react";
import { bindMenu, bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import Upload from "../../Icons/Upload.tsx";
import Fingerprint from "../../Icons/Fingerprint.tsx";
import AvatarCropperDialog from "./AvatarCropperDialog.tsx";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { sendUploadAvatar } from "../../../api/api.ts";
import { useSnackbar } from "notistack";
import { DefaultCloseAction } from "../../Common/Snackbar/snackbar.tsx";

export interface AvatarSettingProps {
  user: User;
}

const AvatarSetting = ({ user }: AvatarSettingProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();
  const popupState = usePopupState({
    popupId: "avatarEdit",
    variant: "popover",
  });
  const fileInput = React.createRef<HTMLInputElement>();
  const { onClose } = bindMenu(popupState);
  const [imageCropperOpen, setImageCropperOpen] = useState(false);
  const [avtarFile, setAvtarFile] = useState<File | undefined>(undefined);
  const [avatarGeneration, setAvatarGeneration] = useState(1);

  const onUploadClicked = () => {
    onClose();
    fileInput.current?.click();
  };

  const onImageChange = () => {
    const file = fileInput.current?.files?.[0];
    if (!file) {
      return;
    }

    fileInput.current.value = "";
    setAvtarFile(file);
    setImageCropperOpen(true);
  };

  const onAvatarUpdated = () => {
    setAvatarGeneration((a) => a + 1);
  };

  const setGravatar = () => {
    dispatch(sendUploadAvatar()).then(() => {
      enqueueSnackbar({
        message: t("application:setting.avatarUpdated", {}),
        variant: "success",
        action: DefaultCloseAction,
      });
      onAvatarUpdated();
    });
    onClose();
  };

  return (
    <SettingForm title={t("setting.avatar")} lgWidth={6}>
      <UserAvatar
        square
        cacheKey={avatarGeneration.toString()}
        user={user}
        sx={{ width: 200, height: 200 }}
        overwriteTextSize
      />
      <SecondaryButton sx={{ mt: 1 }} variant={"contained"} startIcon={<Edit />} {...bindTrigger(popupState)}>
        {t("fileManager.edit")}
      </SecondaryButton>
      <AvatarCropperDialog
        onAvatarUpdated={onAvatarUpdated}
        file={avtarFile}
        onClose={() => setImageCropperOpen(false)}
        open={imageCropperOpen}
      />
      <input onChange={onImageChange} type="file" accept="image/*" style={{ display: "none" }} ref={fileInput} />
      <Menu
        slotProps={{
          paper: {
            sx: {
              minWidth: 150,
            },
          },
        }}
        {...bindMenu(popupState)}
      >
        <SquareMenuItem onClick={onUploadClicked} dense>
          <ListItemIcon>
            <Upload fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t(`setting.uploadImage`)}</ListItemText>
        </SquareMenuItem>
        <SquareMenuItem dense onClick={setGravatar}>
          <ListItemIcon>
            <Fingerprint fontSize="small" />
          </ListItemIcon>
          <ListItemText>{t(`setting.useGravatar`)}</ListItemText>
        </SquareMenuItem>
      </Menu>
    </SettingForm>
  );
};

export default AvatarSetting;
