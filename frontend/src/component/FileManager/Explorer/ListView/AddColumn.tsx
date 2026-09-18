import { useTranslation } from "react-i18next";

import { Icon } from "@iconify/react/dist/iconify.js";
import { ListItemIcon, Menu } from "@mui/material";
import { bindMenu, bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import { useAppSelector } from "../../../../redux/hooks.ts";
import { SecondaryButton } from "../../../Common/StyledComponents.tsx";
import Add from "../../../Icons/Add.tsx";
import { CascadingSubmenu } from "../../ContextMenu/CascadingMenu.tsx";
import { DenseDivider, SquareMenuItem } from "../../ContextMenu/ContextMenu.tsx";
import { ColumType, ColumTypeProps, getColumnTypeDefaults, ListViewColumnSetting } from "./Column.tsx";

export interface AddColumnProps {
  onColumnAdded: (column: ListViewColumnSetting) => void;
}

const options: ColumType[] = [
  ColumType.name,
  ColumType.size,
  ColumType.date_modified,
  ColumType.date_created,
  ColumType.parent,
];

const recycleOptions: ColumType[] = [ColumType.recycle_restore_parent, ColumType.recycle_expire];

// null => divider
const mediaInfoOptions: (ColumType | null)[] = [
  ColumType.aperture,
  ColumType.exposure,
  ColumType.iso,
  ColumType.focal_length,
  ColumType.exposure_bias,
  ColumType.flash,
  null,
  ColumType.camera_make,
  ColumType.camera_model,
  ColumType.lens_make,
  ColumType.lens_model,
  null,
  ColumType.software,
  ColumType.taken_at,
  ColumType.image_size,
  null,
  ColumType.title,
  ColumType.artist,
  ColumType.album,
  ColumType.duration,
  null,
  ColumType.street,
  ColumType.locality,
  ColumType.place,
  ColumType.district,
  ColumType.region,
  ColumType.country,
];

const AddColumn = (props: AddColumnProps) => {
  const { t } = useTranslation();
  const customPropsOptions = useAppSelector((state) => state.siteConfig.explorer?.config?.custom_props);
  const conditionPopupState = usePopupState({
    variant: "popover",
    popupId: "columns",
  });
  const { onClose, ...menuProps } = bindMenu(conditionPopupState);
  const onConditionAdd = (type: ColumType, p?: ColumTypeProps) => {
    props.onColumnAdded({ type, props: p });
    onClose();
  };
  return (
    <>
      <SecondaryButton {...bindTrigger(conditionPopupState)} startIcon={<Add />} sx={{ px: "15px" }}>
        {t("fileManager.addColumn")}
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
          <SquareMenuItem dense key={index} onClick={() => onConditionAdd(option)}>
            {t(getColumnTypeDefaults({ type: option }).title)}
          </SquareMenuItem>
        ))}
        <CascadingSubmenu popupId={"mediaInfo"} title={t("application:navbar.trash")}>
          {recycleOptions.map((option, index) => (
            <SquareMenuItem dense key={index} onClick={() => onConditionAdd(option)}>
              {t(getColumnTypeDefaults({ type: option }).title)}
            </SquareMenuItem>
          ))}
        </CascadingSubmenu>
        <CascadingSubmenu popupId={"mediaInfo"} title={t("application:fileManager.mediaInfo")}>
          {mediaInfoOptions.map((option, index) =>
            option ? (
              <SquareMenuItem dense key={index} onClick={() => onConditionAdd(option)}>
                {t(getColumnTypeDefaults({ type: option }).title)}
              </SquareMenuItem>
            ) : (
              <DenseDivider />
            ),
          )}
        </CascadingSubmenu>
        {customPropsOptions && customPropsOptions.length > 0 && (
          <CascadingSubmenu popupId={"customProps"} title={t("application:fileManager.customProps")}>
            {customPropsOptions.map((option, index) => (
              <SquareMenuItem
                dense
                key={index}
                onClick={() => onConditionAdd(ColumType.custom_props, { custom_props_id: option.id })}
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

export default AddColumn;
