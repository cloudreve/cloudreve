import { Box } from "@mui/material";
import path from "path-browserify";
import React, { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useAppSelector } from "../../../redux/hooks.ts";
import CrUri, { Filesystem } from "../../../util/uri.ts";
import SideNavItem from "../../Frame/NavBar/SideNavItem.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";
import { useBreadcrumbButtons } from "../TopBar/BreadcrumbButton.tsx";
import TreeFile from "./TreeFile.tsx";

export interface TreeFilesProps {
  path: string;
  elements?: string[];
  notLoaded?: boolean;
  level: number;
  flatten?: boolean;
  labelOverwrite?: string;
  pinned?: boolean;
  canDrop?: boolean;
  [key: string]: any;
}

export const pinedPrefix = "Pined";

const TreeFiles = React.memo(
  React.forwardRef(
    (
      { path: p, level, elements, labelOverwrite, notLoaded, pinned, flatten, canDrop, ...rest }: TreeFilesProps,
      ref: React.Ref<HTMLLIElement>,
    ) => {
      const { t } = useTranslation();
      const fmIndex = useContext(FmIndexContext);
      const parentsCache = useAppSelector((state) => state.fileManager[fmIndex].tree[p]);
      const [limit, setLimit] = React.useState(50);
      const uri = useMemo(() => new CrUri(p), [p]);
      const nodeId = useMemo(() => {
        if (pinned && flatten) {
          return pinedPrefix + p;
        }

        return p;
      }, [pinned, p, flatten]);
      const [loading, displayName, startIcon, onClick] = useBreadcrumbButtons({
        name: parentsCache && parentsCache.file ? parentsCache.file.name : path.basename(uri.path()),
        is_latest: false,
        path: p,
      });

      const childTreeFiles = useMemo(() => {
        var res: TreeFilesProps[] = [];

        let newParent: string | null = null;
        let elementPushed = false;
        // Add current if loaded children exist
        if (elements && elements.length >= 1) {
          newParent = new CrUri(p).join(elements[0]).toString();
        }

        // load from store cache
        let currentIndex = 0;
        if (parentsCache && parentsCache.children) {
          parentsCache.children.forEach((child) => {
            let childElements: string[] | undefined = undefined;
            if (newParent && newParent == child) {
              childElements = elements?.slice(1);
              elementPushed = true;
              currentIndex = res.length;
            }
            res.push({
              level: level + 1,
              path: child,
              elements: childElements,
            });
          });
        }

        if (elements && elements.length >= 1 && !elementPushed) {
          const childElements = elements.slice(1);
          currentIndex = res.length;
          res.push({
            level: level + 1,
            path: newParent ?? "",
            elements: childElements,
            notLoaded: childElements.length > 0,
          });
        }

        if (currentIndex >= limit) {
          [res[currentIndex], res[0]] = [res[0], res[currentIndex]];
        }

        return res;
      }, [p, elements, parentsCache, limit]);

      const shadowChild = useMemo(() => {
        if (flatten || (parentsCache?.children && parentsCache.children.length == 0)) {
          return null;
        }
        return <Box />;
      }, [parentsCache, flatten]);

      return (
        <>
          <TreeFile
            canDrop={canDrop}
            key={p}
            level={level}
            ref={ref}
            file={parentsCache?.file}
            nodeId={nodeId}
            fileIcon={startIcon}
            label={labelOverwrite ?? displayName}
            loading={loading}
            notLoaded={notLoaded && !parentsCache?.children}
            pinned={pinned}
            {...rest}
          >
            {!flatten && childTreeFiles.length > 0
              ? childTreeFiles
                  .slice(0, limit)
                  .map((f) => (
                    <TreeFiles
                      flatten={uri.fs() == Filesystem.shared_with_me}
                      key={f.path}
                      level={f.level}
                      path={f.path}
                      notLoaded={f.notLoaded}
                      elements={f.elements}
                    />
                  ))
              : shadowChild}
          </TreeFile>
          {limit < childTreeFiles.length ? (
            <SideNavItem level={level + 1} label={t("navbar.showMore")} onClick={() => setLimit((l) => l + 50)} />
          ) : null}
        </>
      );
    },
  ),
);

export default TreeFiles;
