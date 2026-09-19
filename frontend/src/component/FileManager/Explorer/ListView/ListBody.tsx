import { Box } from "@mui/material";
import { useCallback, useContext, useMemo } from "react";
import { Virtuoso } from "react-virtuoso";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import { setSelected } from "../../../../redux/fileManagerSlice.ts";
import DndWrappedFile from "../../Dnd/DndWrappedFile.tsx";
import { FmIndexContext } from "../../FmIndexContext.tsx";
import { FmFile, loadingPlaceHolderNumb } from "../GridView/GridView.tsx";
import { ListViewColumn } from "./Column.tsx";
import Row from "./Row.tsx";
import { CrUriPrefix } from "../../../../util/uri.ts";

export interface ListBodyProps {
  columns: ListViewColumn[];
}

const ListBody = ({ columns }: ListBodyProps) => {
  const fmIndex = useContext(FmIndexContext);
  const dispatch = useAppDispatch();
  const files = useAppSelector((state) => state.fileManager[fmIndex].list?.files);
  const mixedType = useAppSelector((state) => state.fileManager[fmIndex].list?.mixed_type);
  const pagination = useAppSelector((state) => state.fileManager[fmIndex].list?.pagination);
  const showThumb = useAppSelector((state) => state.fileManager[fmIndex].showThumb);
  const search_params = useAppSelector((state) => state.fileManager[fmIndex]?.search_params);

  const list = useMemo(() => {
    const list: FmFile[] = [];
    if (!files) {
      return list;
    }

    files.forEach((file) => {
      list.push(file);
    });

    // Add loading placeholder if there is next page
    if (pagination && pagination.next_token) {
      for (let i = 0; i < loadingPlaceHolderNumb; i++) {
        const id = `loadingPlaceholder-${pagination.next_token}-${i}`;
        list.push({
          ...files[0],
          path: `${CrUriPrefix}${id}`,
          id: id,
          first: i == 0,
          placeholder: true,
        });
      }
    }
    return list;
  }, [files, mixedType, pagination, search_params]);

  // Clicking empty space below/around rows clears the selection (#3223);
  // row clicks are excluded via the data-fm-row marker.
  const onBackgroundClick = useCallback(
    (e: React.MouseEvent<HTMLElement>) => {
      if (!(e.target as HTMLElement).closest("[data-fm-row]")) {
        dispatch(setSelected({ index: fmIndex, value: [] }));
      }
    },
    [dispatch, fmIndex],
  );

  return (
    <Box sx={{ height: "100%" }} onClick={onBackgroundClick}>
      <Virtuoso
        style={{
          height: "100%",
        }}
        increaseViewportBy={180}
        data={list}
        itemContent={(index, file) => (
          <DndWrappedFile
            columns={columns}
            key={file.id}
            component={Row}
            search={search_params}
            index={index}
            showThumb={showThumb}
            file={file}
          />
        )}
      />
    </Box>
  );
};

export default ListBody;
