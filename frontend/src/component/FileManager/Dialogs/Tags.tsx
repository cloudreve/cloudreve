import { Autocomplete, DialogContent, Stack, useTheme } from "@mui/material";
import { enqueueSnackbar } from "notistack";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FileResponse, Metadata } from "../../../api/explorer.ts";
import { defaultColors } from "../../../constants";
import { closeTagsDialog } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { patchTags } from "../../../redux/thunks/file.ts";
import SessionManager, { UserSettings } from "../../../session";
import { addRecentUsedColor } from "../../../session/utils.ts";
import { FilledTextField } from "../../Common/StyledComponents.tsx";
import DialogAccordion from "../../Dialogs/DialogAccordion.tsx";
import DraggableDialog from "../../Dialogs/DraggableDialog.tsx";
import FileTag from "../Explorer/FileTag.tsx";
import CircleColorSelector, { customizeMagicColor } from "../FileInfo/ColorCircle/CircleColorSelector.tsx";
import { FileManagerIndex } from "../FileManager.tsx";

export interface Tag {
  key: string;
  color?: string;
}

export const getUniqueTagsFromFiles = (targets: FileResponse[]) => {
  const tags: {
    [key: string]: Tag;
  } = {};
  targets.forEach((target) => {
    if (target.metadata) {
      Object.keys(target.metadata).forEach((key: string) => {
        if (key.startsWith(Metadata.tag_prefix)) {
          // trim prefix for key
          const tagKey = key.slice(Metadata.tag_prefix.length);
          tags[tagKey] = {
            key: key.slice(Metadata.tag_prefix.length),
            color: target.metadata?.[key],
          };
        }
      });
    }
  });
  return Object.values(tags);
};

const Tags = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();

  const [hex, setHex] = useState<string | undefined>(undefined);
  const [tags, setTags] = useState<Tag[]>([]);
  const [name, setName] = useState("");
  const [loading, setLoading] = useState(false);

  const open = useAppSelector((state) => state.globalState.tagsDialogOpen);
  const targets = useAppSelector((state) => state.globalState.tagsDialogFile);

  const onClose = useCallback(() => {
    if (!loading) {
      dispatch(closeTagsDialog());
    }
  }, [dispatch, loading]);

  const onAccept = useCallback(
    async (e?: React.FormEvent<HTMLFormElement>) => {
      if (e) {
        e.preventDefault();
      }

      if (!targets) return;

      setLoading(true);
      try {
        await dispatch(patchTags(FileManagerIndex.main, targets, tags));
      } catch (e) {
      } finally {
        setLoading(false);
        dispatch(closeTagsDialog());
      }
    },
    [name, dispatch, targets, tags, setLoading],
  );

  const presetColors = useMemo(() => {
    const colors = new Set(defaultColors);

    const recentColors = SessionManager.get(UserSettings.UsedCustomizedTagColors) as string[] | undefined;

    if (recentColors) {
      recentColors.forEach((color) => {
        colors.add(color);
      });
    }

    return [...colors];
  }, [hex]);

  useEffect(() => {
    if (targets && open) {
      setTags(getUniqueTagsFromFiles(targets));
    }
  }, [targets, open]);

  const onColorChange = useCallback(
    (color: string | undefined) => {
      color = color == theme.palette.action.selected ? undefined : color;
      addRecentUsedColor(color, UserSettings.UsedCustomizedTagColors);
      setHex(color);
    },
    [theme, setHex],
  );

  const onTagAdded = useCallback(
    (_e: any, newValue: (string | Tag)[]) => {
      const duplicateMap: { [key: string]: boolean } = {};
      newValue = newValue.filter((tag) => {
        const tagKey = typeof tag === "string" ? tag : tag.key;
        if (!tagKey) {
          return false;
        }
        if (duplicateMap[tagKey]) {
          enqueueSnackbar(t("application:modals.duplicateTag", { tag: tagKey }), { variant: "warning" });
          return false;
        }
        duplicateMap[tagKey] = true;
        return true;
      });
      setTags(newValue.map((tag) => (typeof tag === "string" ? { key: tag, color: hex } : tag) as Tag));
    },
    [hex, setTags],
  );

  // const onNameChange = useCallback(
  //   (e: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
  //     setName(e.target.value);
  //   },
  //   [dispatch, setName],
  // );

  return (
    <DraggableDialog
      title={t("application:modals.manageTags")}
      showActions
      loading={loading}
      showCancel
      onAccept={onAccept}
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "sm",
      }}
    >
      <DialogContent>
        <Stack spacing={1}>
          <Autocomplete
            multiple
            id="tags-filled"
            options={[]}
            getOptionLabel={(o: any) => o?.key}
            value={tags}
            freeSolo
            autoSelect={true}
            onChange={onTagAdded}
            renderTags={(value: readonly Tag[], getTagProps) =>
              value.map((option: Tag, index: number) => (
                <FileTag
                  defaultStyle
                  openInNewTab
                  spacing={1}
                  label={option.key}
                  size={"medium"}
                  tagColor={option.color}
                  {...getTagProps({ index })}
                  key={option.key}
                />
              ))
            }
            renderInput={(params) => (
              <FilledTextField
                {...params}
                sx={{
                  mt: 2,
                  "& .MuiInputBase-root": {
                    pt: "28px",
                    pb: 1,
                  },
                }}
                variant="filled"
                autoFocus
                helperText={t("application:modals.enterForNewTag")}
                margin="dense"
                label={t("application:fileManager.tags")}
                type="text"
                fullWidth
              />
            )}
          />
          <DialogAccordion title={t("application:modals.colorForTag")}>
            <CircleColorSelector
              colors={[theme.palette.action.selected, ...presetColors, customizeMagicColor]}
              selectedColor={hex ?? theme.palette.action.selected}
              onChange={onColorChange}
            />
          </DialogAccordion>
        </Stack>
      </DialogContent>
    </DraggableDialog>
  );
};
export default Tags;
