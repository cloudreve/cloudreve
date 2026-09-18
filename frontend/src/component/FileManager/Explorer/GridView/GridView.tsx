import { Box, Grid, Stack, styled, Typography } from "@mui/material";
import React, { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { FileResponse, FileType } from "../../../../api/explorer.ts";
import { useAppSelector } from "../../../../redux/hooks.ts";
import DndWrappedFile from "../../Dnd/DndWrappedFile.tsx";
import { FmIndexContext } from "../../FmIndexContext.tsx";
import GridFile from "./GridFile.tsx";
import { CrUriPrefix } from "../../../../util/uri.ts";

export interface GridViewProps {
  [key: string]: any;
}

export interface FmFile extends FileResponse {
  id: string;
  first?: boolean;
  placeholder?: boolean;
}

interface listComponents {
  Folders?: JSX.Element[];
  Files: JSX.Element[];
}

const AutoFillGrid = styled(Grid)(({ theme }) => ({
  [theme.breakpoints.down("md")]: {
    gridTemplateColumns: "repeat(auto-fill,minmax(160px,1fr))!important",
  },
  [theme.breakpoints.up("md")]: {
    gridTemplateColumns: "repeat(auto-fill,minmax(220px,1fr))!important",
  },
  gridGap: theme.spacing(2),
  display: "grid!important",
  padding: theme.spacing(1),
}));

const GridItem = styled(Grid)(() => ({
  flex: "1 1 220px!important",
}));

export const loadingPlaceHolderNumb = 3;

const GridView = React.forwardRef(({ ...rest }: GridViewProps, ref) => {
  const { t } = useTranslation("application");
  const fmIndex = useContext(FmIndexContext);
  const files = useAppSelector((state) => state.fileManager[fmIndex].list?.files);
  const mixedType = useAppSelector((state) => state.fileManager[fmIndex].list?.mixed_type);
  const pagination = useAppSelector((state) => state.fileManager[fmIndex].list?.pagination);
  const showThumb = useAppSelector((state) => state.fileManager[fmIndex].showThumb);
  const search_params = useAppSelector((state) => state.fileManager[fmIndex]?.search_params);
  const list = useMemo(() => {
    const list: listComponents = {
      Files: [],
    };
    if (files) {
      files.forEach((file, index) => {
        if (file.type === FileType.folder && !mixedType) {
          if (!list.Folders) {
            list.Folders = [];
          }
          list.Folders.push(
            <GridItem item key={`${file.id}`}>
              <DndWrappedFile
                component={GridFile}
                search={search_params}
                index={index}
                showThumb={mixedType}
                file={file}
              />
            </GridItem>,
          );
        } else {
          list.Files.push(
            <GridItem item key={`${file.id}`}>
              <DndWrappedFile
                component={GridFile}
                search={search_params}
                index={index}
                showThumb={showThumb}
                file={file}
              />
            </GridItem>,
          );
        }
      });

      // Add loading placeholder if there is next page
      if (pagination && pagination.next_token) {
        for (let i = 0; i < loadingPlaceHolderNumb; i++) {
          const id = `loadingPlaceholder-${pagination.next_token}-${i}`;
          const loadingPlaceholder = (
            <GridItem item key={id}>
              <DndWrappedFile
                component={GridFile}
                showThumb={list.Files.length > 0 ? showThumb : mixedType}
                file={{
                  ...files[0],
                  path: `${CrUriPrefix}${id}`,
                  id: id,
                  first: i == 0,
                  placeholder: true,
                }}
              />
            </GridItem>
          );
          const _ =
            list.Files.length > 0 ? list.Files.push(loadingPlaceholder) : list.Folders?.push(loadingPlaceholder);
        }
      }
    }
    return list;
  }, [files, mixedType, pagination, showThumb]);
  return (
    <Box
      ref={ref}
      {...rest}
      sx={{
        p: 1,
      }}
    >
      <Stack spacing={1}>
        {list.Folders && list.Folders.length > 0 && (
          <Box>
            <Typography fontWeight={"medium"} sx={{ p: 1 }} variant="body2">
              {t("fileManager.folders")}
            </Typography>
            <AutoFillGrid container alignItems="flex-start" spacing={0}>
              {list.Folders.map((f) => f)}
            </AutoFillGrid>
          </Box>
        )}
        {list.Files.length > 0 && (
          <Box>
            {!mixedType && (
              <Typography sx={{ p: 1 }} fontWeight={"medium"} variant="body2">
                {t("fileManager.files")}
              </Typography>
            )}
            <AutoFillGrid container alignItems="flex-start" spacing={0}>
              {list.Files.map((f) => f)}
            </AutoFillGrid>
          </Box>
        )}
      </Stack>
    </Box>
  );
});

export default GridView;
