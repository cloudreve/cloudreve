import { Icon } from "@iconify/react";
import { Box, ListItemIcon, ListItemText, Menu, styled, Typography } from "@mui/material";
import { bindMenu, bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { CustomProps } from "../../../../api/explorer.ts";
import Add from "../../../Icons/Add.tsx";
import { SquareMenuItem } from "../../ContextMenu/ContextMenu.tsx";
import { CustomPropsItem } from "./CustomProps.tsx";

const BorderedCard = styled(Box)(({ theme }) => ({
  border: `1px solid ${theme.palette.divider}`,
  borderRadius: theme.shape.borderRadius,
  padding: theme.spacing(2),
  backgroundColor: theme.palette.background.paper,
}));

const BorderedCardClickable = styled(BorderedCard)<{ disabled?: boolean }>(({ theme, disabled }) => ({
  cursor: "pointer",
  "&:hover": {
    backgroundColor: theme.palette.action.hover,
  },
  transition: "background-color 0.3s ease",
  height: "100%",
  borderStyle: "dashed",
  display: "flex",
  alignItems: "center",
  gap: 8,
  justifyContent: "center",
  color: theme.palette.text.secondary,
  opacity: disabled ? 0.5 : 1,
  pointerEvents: disabled ? "none" : "auto",
}));

export interface AddButtonProps {
  options: CustomProps[];
  existingPropIds: string[];
  onPropAdd: (prop: CustomPropsItem) => void;
  loading?: boolean;
  disabled?: boolean;
}

const AddButton = ({ options, existingPropIds, onPropAdd, disabled }: AddButtonProps) => {
  const { t } = useTranslation();
  const propPopupState = usePopupState({
    variant: "popover",
    popupId: "customProps",
  });
  const { onClose, ...menuProps } = bindMenu(propPopupState);

  const unSelectedOptions = useMemo(() => {
    return options?.filter((option) => !existingPropIds.includes(option.id)) ?? [];
  }, [options, existingPropIds]);

  const handlePropAdd = (prop: CustomProps) => {
    onPropAdd({
      props: prop,
      id: prop.id,
      value: prop.default ?? "",
    });
    onClose();
  };

  if (unSelectedOptions.length === 0) {
    return undefined;
  }

  return (
    <>
      <BorderedCardClickable disabled={disabled} {...bindTrigger(propPopupState)}>
        <Add sx={{ width: 20, height: 20 }} />
        <Typography variant="body1" fontWeight={500}>
          {t("fileManager.add")}
        </Typography>
      </BorderedCardClickable>
      <Menu
        onClose={onClose}
        anchorOrigin={{
          vertical: "bottom",
          horizontal: "left",
        }}
        transformOrigin={{
          vertical: "top",
          horizontal: "left",
        }}
        {...menuProps}
      >
        {unSelectedOptions.map((option) => (
          <SquareMenuItem dense key={option.id} onClick={() => handlePropAdd(option)}>
            {option.icon && (
              <ListItemIcon>
                <Icon icon={option.icon} />
              </ListItemIcon>
            )}
            <ListItemText
              slotProps={{
                primary: { variant: "body2" },
              }}
            >
              {t(option.name)}
            </ListItemText>
          </SquareMenuItem>
        ))}
      </Menu>
    </>
  );
};

export default AddButton;
