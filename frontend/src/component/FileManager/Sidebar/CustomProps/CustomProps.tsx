import { Collapse, Stack, Typography } from "@mui/material";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { TransitionGroup } from "react-transition-group";
import { sendMetadataPatch } from "../../../../api/api.ts";
import { CustomProps as CustomPropsType, FileResponse } from "../../../../api/explorer.ts";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import { fileUpdated } from "../../../../redux/fileManagerSlice.ts";
import { FileManagerIndex } from "../../FileManager.tsx";
import { DisplayOption } from "../../ContextMenu/useActionDisplayOpt.ts";
import AddButton from "./AddButton.tsx";
import CustomPropsCard from "./CustomPropsItem.tsx";

export interface CustomPropsProps {
  file: FileResponse;
  setTarget: (target: FileResponse | undefined | null) => void;
  targetDisplayOptions?: DisplayOption;
}

export interface CustomPropsItem {
  props: CustomPropsType;
  id: string;
  value: string;
}

export const customPropsMetadataPrefix = "props:";

const CustomProps = ({ file, setTarget, targetDisplayOptions }: CustomPropsProps) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const dispatch = useAppDispatch();
  const custom_props = useAppSelector((state) => state.siteConfig.explorer?.config?.custom_props);
  // `file` is a snapshot taken when the sidebar opened; prefer live metadata
  // from the file list so props patched from list view appear immediately.
  const liveMetadata = useAppSelector(
    (state) => state.fileManager[FileManagerIndex.main]?.list?.files.find((f) => f.path === file.path)?.metadata,
  );
  const metadata = liveMetadata ?? file.metadata;

  const existingProps = useMemo(() => {
    if (!metadata) {
      return [];
    }
    return Object.keys(metadata)
      .filter((key) => key.startsWith(customPropsMetadataPrefix))
      .map((key) => {
        const propId = key.slice(customPropsMetadataPrefix.length);
        return {
          id: propId,
          props: custom_props?.find((prop) => prop.id === propId),
          value: metadata?.[key] ?? "",
        } as CustomPropsItem;
      });
  }, [metadata, custom_props]);

  const existingPropIds = useMemo(() => {
    return existingProps?.map((prop) => prop.id) ?? [];
  }, [existingProps]);

  const handlePropPatch = (remove?: boolean) => (props: CustomPropsItem[]) => {
    setLoading(true);
    dispatch(
      sendMetadataPatch({
        uris: [file.path],
        patches: props.map((prop) => ({
          key: customPropsMetadataPrefix + prop.id,
          value: prop.value,
          remove,
        })),
      }),
    )
      .then(() => {
        const newMetadata = { ...metadata };
        props.forEach((prop) => {
          if (remove) {
            delete newMetadata[customPropsMetadataPrefix + prop.id];
          } else {
            newMetadata[customPropsMetadataPrefix + prop.id] = prop.value;
          }
        });
        const updated = { ...file, metadata: newMetadata };
        setTarget(updated);
        dispatch(
          fileUpdated({
            index: FileManagerIndex.main,
            value: [{ file: updated, oldPath: file.path, includeMetadata: true }],
          }),
        );
      })
      .finally(() => {
        setLoading(false);
      });
  };

  if (existingProps.length === 0 && (!custom_props || custom_props.length === 0)) {
    return undefined;
  }

  return (
    <Stack spacing={1}>
      <Typography sx={{ pt: 1 }} color="textPrimary" fontWeight={500} variant={"subtitle1"}>
        {t("fileManager.customProps")}
      </Typography>
      <AddButton
        disabled={!targetDisplayOptions?.showCustomProps}
        loading={loading}
        options={custom_props ?? []}
        existingPropIds={existingPropIds}
        onPropAdd={(prop) => {
          handlePropPatch(false)([prop]);
        }}
      />
      <TransitionGroup>
        {existingProps.map((prop, index) => (
          <Collapse key={prop.id} sx={{ mb: index === existingProps.length - 1 ? 0 : 1 }}>
            <CustomPropsCard
              key={prop.id}
              prop={prop}
              loading={loading}
              onPropPatch={(prop) => {
                handlePropPatch(false)([prop]);
              }}
              onPropDelete={(prop) => {
                handlePropPatch(true)([prop]);
              }}
              readOnly={!targetDisplayOptions?.showCustomProps}
            />
          </Collapse>
        ))}
      </TransitionGroup>
    </Stack>
  );
};

export default CustomProps;
