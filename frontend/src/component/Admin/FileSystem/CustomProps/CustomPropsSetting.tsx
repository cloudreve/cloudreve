import {
  Box,
  ListItemIcon,
  Menu,
  Stack,
  Table,
  TableBody,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import { bindMenu, bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import { createRef, useCallback, useContext, useEffect, useState } from "react";
import { DndProvider } from "react-dnd";
import { HTML5Backend } from "react-dnd-html5-backend";
import { useTranslation } from "react-i18next";
import { CustomProps, CustomPropsType } from "../../../../api/explorer";
import { NoWrapCell, SecondaryButton, StyledTableContainerPaper } from "../../../Common/StyledComponents";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu";
import Add from "../../../Icons/Add";
import { ProChip } from "../../../Pages/Setting/SettingForm";
import ProDialog from "../../Common/ProDialog";
import { SettingContext } from "../../Settings/SettingWrapper";
import DraggableCustomPropsRow, { FieldTypes } from "./DraggableCustomPropsRow";
import EditPropsDialog from "./EditPropsDialog";

const CustomPropsSetting = () => {
  const { t } = useTranslation("dashboard");
  const { formRef, setSettings, values } = useContext(SettingContext);
  const [customProps, setCustomProps] = useState<CustomProps[]>([]);
  const [open, setOpen] = useState(false);
  const [isNew, setIsNew] = useState(false);
  const [editProps, setEditProps] = useState<CustomProps | undefined>(undefined);
  const [proOpen, setProOpen] = useState(false);
  const newPropsPopupState = usePopupState({
    variant: "popover",
    popupId: "newProp",
  });
  const { onClose, ...menuProps } = bindMenu(newPropsPopupState);

  useEffect(() => {
    try {
      setCustomProps(JSON.parse(values.custom_props || "[]"));
    } catch {
      setCustomProps([]);
    }
  }, [values.custom_props]);

  const onChange = useCallback(
    (customProps: CustomProps[]) => {
      setSettings({
        custom_props: JSON.stringify(customProps),
      });
    },
    [setSettings],
  );

  const handleDeleteProduct = useCallback(
    (id: string) => {
      const newCustomProps = customProps.filter((p) => p.id !== id);
      setCustomProps(newCustomProps);
      onChange(newCustomProps);
    },
    [customProps, onChange],
  );

  const handleSave = useCallback(
    (props: CustomProps) => {
      const existingIndex = customProps.findIndex((p) => p.id === props.id);
      let newCustomProps: CustomProps[];
      if (existingIndex >= 0) {
        newCustomProps = [...customProps];
        newCustomProps[existingIndex] = props;
      } else {
        newCustomProps = [...customProps, props];
      }
      setCustomProps(newCustomProps);
      onChange(newCustomProps);
    },
    [customProps, onChange],
  );

  const moveRow = useCallback(
    (from: number, to: number) => {
      if (from === to) return;
      const updated = [...customProps];
      const [moved] = updated.splice(from, 1);
      updated.splice(to, 0, moved);
      setCustomProps(updated);
      onChange(updated);
    },
    [customProps, onChange],
  );

  const handleMoveUp = (idx: number) => {
    if (idx <= 0) return;
    moveRow(idx, idx - 1);
  };
  const handleMoveDown = (idx: number) => {
    if (idx >= customProps.length - 1) return;
    moveRow(idx, idx + 1);
  };

  const onNewProp = (type: CustomPropsType) => {
    setEditProps({
      type,
      id: "",
      name: "",
      default: "",
    });
    setIsNew(true);
    setOpen(true);
    onClose();
  };

  return (
    <Box component={"form"} ref={formRef} onSubmit={(e) => e.preventDefault()}>
      <Stack spacing={1}>
        <ProDialog open={proOpen} onClose={() => setProOpen(false)} />
        <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 1 }}>
          <SecondaryButton variant="contained" startIcon={<Add />} {...bindTrigger(newPropsPopupState)}>
            {t("customProps.add")}
          </SecondaryButton>
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
            {(Object.keys(FieldTypes) as CustomPropsType[]).map((type, index) => {
              const fieldType = FieldTypes[type];
              const Icon = fieldType.icon;
              return (
                <SquareMenuItem dense key={index} onClick={() => (fieldType.pro ? setProOpen(true) : onNewProp(type))}>
                  <ListItemIcon>
                    <Icon />
                  </ListItemIcon>
                  {t(fieldType.title)}
                  {fieldType.pro && <ProChip label="Pro" color="primary" size="small" />}
                </SquareMenuItem>
              );
            })}
          </Menu>
        </Box>
        <TableContainer component={StyledTableContainerPaper}>
          <DndProvider backend={HTML5Backend}>
            <Table sx={{ width: "100%" }} size="small">
              <TableHead>
                <TableRow>
                  <NoWrapCell>{t("settings.displayName")}</NoWrapCell>
                  <NoWrapCell>{t("customProps.type")}</NoWrapCell>
                  <NoWrapCell>{t("customProps.default")}</NoWrapCell>
                  <NoWrapCell>{t("settings.actions")}</NoWrapCell>
                  <NoWrapCell></NoWrapCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {customProps.map((prop, idx) => {
                  const rowRef = createRef<HTMLTableRowElement>();
                  return (
                    <DraggableCustomPropsRow
                      key={prop.id}
                      ref={rowRef}
                      customProps={prop}
                      index={idx}
                      moveRow={moveRow}
                      onEdit={(props) => {
                        setEditProps(props);
                        setIsNew(false);
                        setOpen(true);
                      }}
                      onDelete={handleDeleteProduct}
                      onMoveUp={() => handleMoveUp(idx)}
                      onMoveDown={() => handleMoveDown(idx)}
                      isFirst={idx === 0}
                      isLast={idx === customProps.length - 1}
                      t={t}
                    />
                  );
                })}
                {customProps.length === 0 && (
                  <TableRow>
                    <NoWrapCell colSpan={6} align="center">
                      <Typography variant="caption" color="text.secondary">
                        {t("application:setting.listEmpty")}
                      </Typography>
                    </NoWrapCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </DndProvider>
        </TableContainer>
      </Stack>
      <EditPropsDialog open={open} onClose={() => setOpen(false)} onSave={handleSave} isNew={isNew} props={editProps} />
    </Box>
  );
};

export default CustomPropsSetting;
