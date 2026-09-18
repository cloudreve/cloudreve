import React, { useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { SubMenuItemsProps } from "./OrganizeMenuItems.tsx";
import { ViewersByID } from "../../../redux/siteConfigSlice.ts";
import { ListItemIcon, ListItemText } from "@mui/material";
import { ViewerIcon } from "../Dialogs/OpenWith.tsx";
import { SquareMenuItem } from "./ContextMenu.tsx";
import { CascadingContext, CascadingMenuItem, CascadingSubmenu } from "./CascadingMenu.tsx";
import { NewFileTemplate, Viewer } from "../../../api/explorer.ts";
import { createNew } from "../../../redux/thunks/file.ts";
import { CreateNewDialogType } from "../../../redux/globalStateSlice.ts";

interface MultiTemplatesMenuItemsProps {
  viewer: Viewer;
}

const MultiTemplatesMenuItems = ({ viewer }: MultiTemplatesMenuItemsProps) => {
  const { rootPopupState } = useContext(CascadingContext);
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const onClick = useCallback(
    (f: NewFileTemplate) => () => {
      if (rootPopupState) {
        rootPopupState.close();
      }

      dispatch(createNew(0, CreateNewDialogType.file, viewer, f));
    },
    [dispatch],
  );

  return (
    <>
      {viewer.templates?.map((template) => (
        <CascadingMenuItem key={template.ext} onClick={onClick(template)}>
          <ListItemText>
            {t("fileManager.newDocumentType", {
              display_name: t(template.display_name),
              ext: template.ext,
            })}
          </ListItemText>
        </CascadingMenuItem>
      ))}
    </>
  );
};

const NewFileTemplateMenuItems = (props: SubMenuItemsProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const onClick = useCallback(
    (viewer: Viewer, template: NewFileTemplate) => () => {
      dispatch(createNew(0, CreateNewDialogType.file, viewer, template));
    },
    [dispatch],
  );

  return (
    <>
      {Object.values(ViewersByID)
        .filter((viewer) => viewer.templates)
        .map((viewer) => {
          if (!viewer.templates) {
            return null;
          }

          if (viewer.templates.length == 1) {
            return (
              <SquareMenuItem key={viewer.id} onClick={onClick(viewer, viewer.templates[0])}>
                <ListItemIcon>
                  <ViewerIcon size={20} viewer={viewer} py={0} />
                </ListItemIcon>
                <ListItemText>
                  {t("fileManager.newDocumentType", {
                    display_name: t(viewer.templates[0].display_name),
                    ext: viewer.templates[0].ext,
                  })}
                </ListItemText>
              </SquareMenuItem>
            );
          } else {
            return (
              <CascadingSubmenu
                popupId={viewer.id}
                title={t(viewer.display_name)}
                icon={<ViewerIcon size={20} viewer={viewer} py={0} />}
              >
                <MultiTemplatesMenuItems viewer={viewer} />
              </CascadingSubmenu>
            );
          }
        })}
    </>
  );
};

export default NewFileTemplateMenuItems;
