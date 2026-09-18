import { useTranslation } from "react-i18next";

import { Icon } from "@iconify/react/dist/iconify.js";
import { ListItemIcon, ListItemText, Menu } from "@mui/material";
import dayjs from "dayjs";
import { bindMenu, bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import { FileType, Metadata } from "../../../../api/explorer.ts";
import { useAppSelector } from "../../../../redux/hooks.ts";
import { SecondaryButton } from "../../../Common/StyledComponents.tsx";
import Add from "../../../Icons/Add.tsx";
import CalendarClock from "../../../Icons/CalendarClock.tsx";
import FolderOutlined from "../../../Icons/FolderOutlined.tsx";
import HardDriveOutlined from "../../../Icons/HardDriveOutlined.tsx";
import Info from "../../../Icons/Info.tsx";
import Numbers from "../../../Icons/Numbers.tsx";
import Tag from "../../../Icons/Tag.tsx";
import TextBulletListSquareEdit from "../../../Icons/TextBulletListSquareEdit.tsx";
import TextCaseTitle from "../../../Icons/TextCaseTitle.tsx";
import { CascadingSubmenu } from "../../ContextMenu/CascadingMenu.tsx";
import { DenseDivider, SquareMenuItem } from "../../ContextMenu/ContextMenu.tsx";
import { customPropsMetadataPrefix } from "../../Sidebar/CustomProps/CustomProps.tsx";
import { Condition, ConditionType } from "./ConditionBox.tsx";

export interface AddConditionProps {
  onConditionAdd: (condition: Condition) => void;
}

interface ConditionOption {
  name: string;
  icon?: JSX.Element;
  condition: Condition;
}

const options: ConditionOption[] = [
  {
    name: "application:modals.fileName",
    icon: <TextCaseTitle fontSize={"small"} />,
    condition: { type: ConditionType.name, case_folding: true },
  },
  {
    name: "application:navbar.fileType",
    icon: <FolderOutlined fontSize={"small"} />,
    condition: { type: ConditionType.type, file_type: FileType.file },
  },
  {
    name: "application:fileManager.tags",
    icon: <Tag fontSize={"small"} />,
    condition: { type: ConditionType.tag },
  },
  {
    name: "application:fileManager.metadata",
    icon: <Numbers fontSize={"small"} />,
    condition: { type: ConditionType.metadata },
  },
  {
    name: "application:navbar.fileSize",
    icon: <HardDriveOutlined fontSize={"small"} />,
    condition: { type: ConditionType.size, size_lte: 0, size_gte: 0 },
  },
  {
    name: "application:fileManager.createDate",
    icon: <CalendarClock fontSize={"small"} />,
    condition: {
      type: ConditionType.created,
      created_gte: dayjs().subtract(7, "d").unix(),
      created_lte: dayjs().unix(),
    },
  },
  {
    name: "application:fileManager.updatedDate",
    icon: <CalendarClock fontSize={"small"} />,
    condition: {
      type: ConditionType.modified,
      updated_gte: dayjs().subtract(7, "d").unix(),
      updated_lte: dayjs().unix(),
    },
  },
];

const mediaMetaOptions: (ConditionOption | null)[] = [
  {
    name: "application:fileManager.title",
    condition: {
      type: ConditionType.metadata,
      metadata_key_readonly: true,
      metadata_key: Metadata.music_title,
      id: Metadata.music_title,
    },
  },
  {
    name: "application:fileManager.artist",
    condition: {
      type: ConditionType.metadata,
      metadata_key_readonly: true,
      metadata_key: Metadata.music_artist,
      id: Metadata.music_artist,
    },
  },
  {
    name: "application:fileManager.album",
    condition: {
      type: ConditionType.metadata,
      metadata_key_readonly: true,
      metadata_key: Metadata.music_album,
      id: Metadata.music_album,
    },
  },
  null, // divider
  {
    name: "application:fileManager.cameraMake",
    condition: {
      type: ConditionType.metadata,
      metadata_key_readonly: true,
      metadata_key: Metadata.camera_make,
      id: Metadata.camera_make,
    },
  },
  {
    name: "application:fileManager.cameraModel",
    condition: {
      type: ConditionType.metadata,
      metadata_key_readonly: true,
      metadata_key: Metadata.camera_model,
      id: Metadata.camera_model,
    },
  },
  {
    name: "application:fileManager.lensMake",
    condition: {
      type: ConditionType.metadata,
      metadata_key_readonly: true,
      metadata_key: Metadata.lens_make,
      id: Metadata.lens_make,
    },
  },
  {
    name: "application:fileManager.lensModel",
    condition: {
      type: ConditionType.metadata,
      metadata_key_readonly: true,
      metadata_key: Metadata.lens_model,
      id: Metadata.lens_model,
    },
  },
  null, // divider
  {
    name: "application:fileManager.street",
    condition: {
      type: ConditionType.metadata,
      metadata_key_readonly: true,
      metadata_key: Metadata.street,
      id: Metadata.street,
    },
  },
  {
    name: "application:fileManager.locality",
    condition: {
      type: ConditionType.metadata,
      metadata_key_readonly: true,
      metadata_key: Metadata.locality,
      id: Metadata.locality,
    },
  },
  {
    name: "application:fileManager.place",
    condition: {
      type: ConditionType.metadata,
      metadata_key_readonly: true,
      metadata_key: Metadata.place,
      id: Metadata.place,
    },
  },
  {
    name: "application:fileManager.district",
    condition: {
      type: ConditionType.metadata,
      metadata_key_readonly: true,
      metadata_key: Metadata.district,
      id: Metadata.district,
    },
  },
  {
    name: "application:fileManager.region",
    condition: {
      type: ConditionType.metadata,
      metadata_key_readonly: true,
      metadata_key: Metadata.region,
      id: Metadata.region,
    },
  },
  {
    name: "application:fileManager.country",
    condition: {
      type: ConditionType.metadata,
      metadata_key_readonly: true,
      metadata_key: Metadata.country,
      id: Metadata.country,
    },
  },
];

const AddCondition = (props: AddConditionProps) => {
  const { t } = useTranslation();
  const customPropsOptions = useAppSelector((state) => state.siteConfig.explorer?.config?.custom_props);
  const conditionPopupState = usePopupState({
    variant: "popover",
    popupId: "conditions",
  });
  const { onClose, ...menuProps } = bindMenu(conditionPopupState);
  const onConditionAdd = (condition: Condition) => {
    props.onConditionAdd({
      ...condition,
      id: condition.type == ConditionType.metadata && !condition.id ? Math.random().toString() : condition.id,
    });
    onClose();
  };
  return (
    <>
      <SecondaryButton {...bindTrigger(conditionPopupState)} startIcon={<Add />} sx={{ px: "15px" }}>
        {t("navbar.addCondition")}
      </SecondaryButton>
      <Menu
        onClose={onClose}
        anchorOrigin={{
          vertical: "top",
          horizontal: "left",
        }}
        transformOrigin={{
          vertical: "bottom",
          horizontal: "left",
        }}
        {...menuProps}
      >
        {options.map((option, index) => (
          <SquareMenuItem dense key={index} onClick={() => onConditionAdd(option.condition)}>
            <ListItemIcon>{option.icon}</ListItemIcon>
            {t(option.name)}
          </SquareMenuItem>
        ))}
        <CascadingSubmenu
          icon={<Info fontSize="small" />}
          popupId={"mediaInfo"}
          title={t("application:fileManager.mediaInfo")}
        >
          {mediaMetaOptions.map((option, index) =>
            option ? (
              <SquareMenuItem key={index} dense onClick={() => onConditionAdd(option.condition)}>
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {t(option.name)}
                </ListItemText>
              </SquareMenuItem>
            ) : (
              <DenseDivider />
            ),
          )}
        </CascadingSubmenu>
        {customPropsOptions && customPropsOptions.length > 0 && (
          <CascadingSubmenu
            icon={<TextBulletListSquareEdit fontSize="small" />}
            popupId={"customProps"}
            title={t("application:fileManager.customProps")}
          >
            {customPropsOptions.map((option, index) => (
              <SquareMenuItem
                dense
                key={index}
                onClick={() =>
                  onConditionAdd({
                    type: ConditionType.metadata,
                    id: customPropsMetadataPrefix + option.id,
                    metadata_key: customPropsMetadataPrefix + option.id,
                  })
                }
              >
                {option.icon && (
                  <ListItemIcon>
                    <Icon icon={option.icon} />
                  </ListItemIcon>
                )}
                {t(option.name)}
              </SquareMenuItem>
            ))}
          </CascadingSubmenu>
        )}
      </Menu>
    </>
  );
};

export default AddCondition;
