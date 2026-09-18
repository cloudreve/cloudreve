import React, { memo, useContext, useMemo } from "react";
import SessionManager from "../../../session";
import TreeFiles from "./TreeFiles.tsx";
import CrUri, { Filesystem } from "../../../util/uri.ts";
import { useAppSelector } from "../../../redux/hooks.ts";
import { FmIndexContext } from "../FmIndexContext.tsx";

interface PinnedItem {
  uri: string;
  name?: string;
  crUri: CrUri;
  useElements?: boolean;
}

export const usePinned = () => {
  const fmIndex = useContext(FmIndexContext);
  const generation = useAppSelector((state) => state.globalState.pinedGeneration);
  const path_root = useAppSelector((state) => state.fileManager[fmIndex].path_root);
  const pined = useMemo(() => {
    try {
      return SessionManager.currentLogin().user.pined?.map((p): PinnedItem => {
        return {
          uri: p.uri,
          name: p.name,
          crUri: new CrUri(p.uri),
          useElements: p.uri == path_root,
        };
      });
    } catch (e) {}
  }, [generation, path_root]);

  return pined;
};

const Pinned = memo(() => {
  const fmIndex = useContext(FmIndexContext);
  const elements = useAppSelector((state) => state.fileManager[fmIndex].path_elements);
  const pined = usePinned();

  return (
    <>
      {pined?.map((p, index) => (
        <TreeFiles
          sx={{ mt: index == 0 ? 2 : 0 }}
          flatten={
            p.crUri.fs() != Filesystem.share ||
            (p.crUri.fs() == Filesystem.share && (!p.crUri.is_root() || p.crUri.is_search()))
          }
          canDrop={(p.crUri.fs() == Filesystem.share || p.crUri.fs() == Filesystem.my) && !p.crUri.is_search()}
          level={0}
          labelOverwrite={p.name}
          path={p.uri}
          elements={p.useElements ? elements : undefined}
          key={"Pined" + p.uri}
          pinned
        />
      ))}
    </>
  );
});
export default Pinned;
