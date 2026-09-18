import { SearchOutlined } from "@mui/icons-material";
import {
  Box,
  debounce,
  Dialog,
  Divider,
  Grow,
  IconButton,
  styled,
  Tooltip,
  Typography,
  useMediaQuery,
  useTheme,
} from "@mui/material";
import { TransitionProps } from "@mui/material/transitions";
import Fuse from "fuse.js";
import { forwardRef, useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { sendFullTextSearch } from "../../../api/api.ts";
import {
  FileResponse,
  FullTextSearchResults,
  FullTextSearchResult as FullTextSearchResultType,
} from "../../../api/explorer.ts";
import { defaultPath } from "../../../hooks/useNavigation.tsx";
import { setSearchPopup } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { openAdvancedSearch, quickSearch } from "../../../redux/thunks/filemanager.ts";
import SessionManager from "../../../session";
import CrUri, { Filesystem } from "../../../util/uri.ts";
import AutoHeight from "../../Common/AutoHeight.tsx";
import { OutlineIconTextField } from "../../Common/Form/OutlineIconTextField.tsx";
import Options from "../../Icons/Options.tsx";
import { FileManagerIndex } from "../FileManager.tsx";
import FullSearchOption from "./FullSearchOptions.tsx";
import FullTextSearchResultList from "./FullTextSearchResult.tsx";
import FuzzySearchResult from "./FuzzySearchResult.tsx";

const StyledDialog = styled(Dialog)<{
  expanded?: boolean;
}>(({ theme, expanded }) => ({
  "& .MuiDialog-container": {
    alignItems: "flex-start",
    height: expanded ? "100%" : "initial",
  },
  zIndex: theme.zIndex.modal - 1,
}));

const StyledOutlinedIconTextFiled = styled(OutlineIconTextField)(() => ({
  "& .MuiOutlinedInput-notchedOutline": {
    border: "none",
  },
}));

export const GrowDialogTransition = forwardRef(function Transition(
  props: TransitionProps & {
    children: React.ReactElement<any, any>;
  },
  ref: React.Ref<unknown>,
) {
  return <Grow ref={ref} {...props} />;
});

const SearchPopup = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));

  const [keywords, setKeywords] = useState("");
  const [searchedKeyword, setSearchedKeyword] = useState("");
  const [treeSearchResults, setTreeSearchResults] = useState<FileResponse[]>([]);
  const [fullTextResults, setFullTextResults] = useState<FullTextSearchResultType[]>([]);
  const [fullTextTotal, setFullTextTotal] = useState(0);
  const [fullTextLoading, setFullTextLoading] = useState(false);

  const onClose = () => {
    dispatch(setSearchPopup(false));
    setKeywords("");
    setSearchedKeyword("");
    setFullTextResults([]);
    setFullTextTotal(0);
  };

  const open = useAppSelector((state) => state.globalState.searchPopupOpen);
  const tree = useAppSelector((state) => state.fileManager[FileManagerIndex.main]?.tree);
  const path = useAppSelector((state) => state.fileManager[FileManagerIndex.main]?.path);
  const single_file_view = useAppSelector((state) => state.fileManager[FileManagerIndex.main]?.list?.single_file_view);
  const fullTextSearchEnabled = useAppSelector((state) => state.siteConfig.explorer?.config?.full_text_search);
  const isLoggedIn = !!SessionManager.currentLoginOrNull();

  const searchTree = useMemo(
    () =>
      debounce((request: { input: string }, callback: (results?: FileResponse[]) => void) => {
        const options = {
          includeScore: true,
          // Search in `author` and in `tags` array
          keys: ["file.name"],
        };
        const fuse = new Fuse(Object.values(tree), options);
        const result = fuse.search(
          request.input
            .split(" ")
            .filter((k) => k != "")
            .join(" "),
          { limit: 50 },
        );
        const res: FileResponse[] = [];
        result
          .filter((r) => r.item.file != undefined)
          .forEach((r) => {
            if (r.item.file) {
              res.push(r.item.file);
            }
          });
        callback(res);
      }, 400),
    [tree],
  );

  useEffect(() => {
    let active = true;

    if (keywords === "" || keywords.length < 2) {
      setTreeSearchResults([]);
      setSearchedKeyword("");
      return undefined;
    }

    searchTree({ input: keywords }, (results?: FileResponse[]) => {
      if (active) {
        setTreeSearchResults(results ?? []);
        setSearchedKeyword(keywords);
      }
    });
    return () => {
      active = false;
    };
  }, [keywords, setSearchedKeyword, searchTree]);

  const fullTextSearchDebounced = useMemo(
    () =>
      debounce((query: string) => {
        if (!fullTextSearchEnabled || !isLoggedIn) return;
        setFullTextLoading(true);
        setFullTextResults([]);
        setFullTextTotal(0);
        dispatch(sendFullTextSearch(query))
          .then((res: FullTextSearchResults) => {
            setFullTextResults(res.hits ?? []);
            setFullTextTotal(res.total);
          })
          .catch(() => {
            setFullTextResults([]);
            setFullTextTotal(0);
          })
          .finally(() => {
            setFullTextLoading(false);
          });
      }, 500),
    [fullTextSearchEnabled, isLoggedIn, path, dispatch],
  );

  useEffect(() => {
    if (keywords === "" || keywords.length < 2 || !fullTextSearchEnabled || !isLoggedIn) {
      setFullTextResults([]);
      setFullTextTotal(0);
      return;
    }

    fullTextSearchDebounced(keywords);
    return () => {
      fullTextSearchDebounced.clear();
    };
  }, [keywords, fullTextSearchDebounced, fullTextSearchEnabled, isLoggedIn]);

  const loadMoreFullText = useCallback(() => {
    if (fullTextLoading) return;
    setFullTextLoading(true);
    dispatch(sendFullTextSearch(keywords, fullTextResults.length))
      .then((res: FullTextSearchResults) => {
        setFullTextResults((prev) => [...prev, ...(res.hits ?? [])]);
        setFullTextTotal(res.total);
      })
      .catch(() => {})
      .finally(() => {
        setFullTextLoading(false);
      });
  }, [keywords, fullTextResults.length, fullTextLoading, dispatch]);

  const fullSearchOptions = useMemo(() => {
    if (!open || !keywords) {
      return [];
    }

    const res: string[] = [];
    const current = new CrUri(path ?? defaultPath);
    // current folder - not currently in root
    if (!current.is_root()) {
      res.push(current.toString());
    }
    // current root - not in single file view
    if (!single_file_view) {
      res.push(current.base());
    }
    // my files - user login and not my fs
    if (SessionManager.currentLoginOrNull() && !(current.fs() == Filesystem.my)) {
      res.push(defaultPath);
    }
    return res;
  }, [open, path, single_file_view, keywords]);

  const onEnter = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && !e.nativeEvent.isComposing) {
        e.stopPropagation();
        e.preventDefault();
        if (fullSearchOptions.length > 0) {
          dispatch(quickSearch(FileManagerIndex.main, fullSearchOptions[0], keywords));
        }
      }
    },
    [fullSearchOptions, keywords],
  );

  return (
    <StyledDialog
      TransitionComponent={GrowDialogTransition}
      fullWidth
      expanded={!!keywords}
      maxWidth={"md"}
      open={!!open}
      onClose={onClose}
    >
      <Box
        sx={{
          display: "flex",
          alignItems: "center",
        }}
      >
        <StyledOutlinedIconTextFiled
          icon={<SearchOutlined />}
          variant="outlined"
          autoFocus
          onKeyDown={onEnter}
          value={keywords}
          onChange={(e) => setKeywords(e.target.value)}
          placeholder={t("navbar.searchFiles")}
          fullWidth
        />
        <Tooltip title={t("application:navbar.advancedSearch")}>
          <IconButton
            sx={{
              width: 40,
              height: 40,
              mr: 1.5,
            }}
            onClick={() => dispatch(openAdvancedSearch(FileManagerIndex.main, keywords))}
          >
            <Options />
          </IconButton>
        </Tooltip>
      </Box>

      {keywords && <Divider />}
      <Box
        sx={{
          flexGrow: 1,
          overflowY: "auto",
        }}
      >
        <AutoHeight>
          {fullSearchOptions.length > 0 && (
            <>
              <Typography
                variant={"body2"}
                color={"textSecondary"}
                sx={{
                  px: 3,
                  pt: 1.5,
                  fontWeight: 600,
                }}
              >
                {t("navbar.searchFilesTitle")}
              </Typography>
              <FullSearchOption keyword={keywords} options={fullSearchOptions} />
              {(fullTextResults.length > 0 || fullTextLoading || treeSearchResults.length > 0) && <Divider />}
            </>
          )}
          {fullTextSearchEnabled && isLoggedIn && (fullTextResults.length > 0 || fullTextLoading) && (
            <>
              <Typography
                variant={"body2"}
                color={"textSecondary"}
                sx={{
                  px: 3,
                  pt: 1.5,
                  fontWeight: 600,
                }}
              >
                {t("navbar.fullTextSearch")}
              </Typography>
              <FullTextSearchResultList
                results={fullTextResults}
                loading={fullTextLoading}
                hasMore={fullTextResults.length < fullTextTotal}
                onLoadMore={loadMoreFullText}
              />
              {treeSearchResults.length > 0 && <Divider />}
            </>
          )}
          {treeSearchResults.length > 0 && (
            <>
              <Typography
                variant={"body2"}
                color={"textSecondary"}
                sx={{
                  px: 3,
                  pt: 1.5,
                  fontWeight: 600,
                }}
              >
                {t("navbar.recentlyViewed")}
              </Typography>
              <FuzzySearchResult keyword={searchedKeyword} files={treeSearchResults} />
            </>
          )}
        </AutoHeight>
      </Box>
    </StyledDialog>
  );
};

export default SearchPopup;
