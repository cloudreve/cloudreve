import { alpha, Button, ButtonGroup, Grow, styled, useMediaQuery, useTheme } from "@mui/material";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { clearSearch, openAdvancedSearch } from "../../../redux/thunks/filemanager.ts";
import Dismiss from "../../Icons/Dismiss.tsx";
import Search from "../../Icons/Search.tsx";
import { FileManagerIndex } from "../FileManager.tsx";
import { FmIndexContext } from "../FmIndexContext.tsx";

export const StyledButtonGroup = styled(ButtonGroup)(({ theme }) => ({
  "& .MuiButtonGroup-firstButton, .MuiButtonGroup-lastButton": {
    "&:hover": {
      border: "none",
    },
  },
}));
export const StyledButton = styled(Button)(({ theme }) => ({
  border: "none",
  backgroundColor: alpha(theme.palette.primary.main, 0.1),
  "&:hover": {
    backgroundColor: alpha(theme.palette.primary.main, 0.2),
  },
  fontSize: theme.typography.caption.fontSize,
  minWidth: 0,
  "& .MuiButton-startIcon": {},
}));

export const SearchIndicator = () => {
  const { t } = useTranslation();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const dispatch = useAppDispatch();
  const fmIndex = useContext(FmIndexContext);

  const search_params = useAppSelector((state) => state.fileManager[fmIndex].search_params);

  const searchConditionsCount = useMemo(() => {
    if (!search_params) {
      return 0;
    }

    let count = 0;
    if (search_params.name) {
      count++;
    }
    if (search_params.metadata) {
      count += Object.keys(search_params.metadata).length;
    }
    if (search_params.type != undefined) {
      count++;
    }
    if (search_params.size_gte || search_params.size_lte) {
      count++;
    }
    if (search_params.created_at_gte || search_params.created_at_lte) {
      count++;
    }
    if (search_params.updated_at_gte || search_params.updated_at_lte) {
      count++;
    }
    if (search_params.metadata_strong_match) {
      count += Object.keys(search_params.metadata_strong_match).length;
    }
    return count;
  }, [search_params]);

  return (
    <Grow unmountOnExit in={searchConditionsCount > 0}>
      <StyledButtonGroup>
        <StyledButton
          disabled={fmIndex != FileManagerIndex.main}
          size={"small"}
          startIcon={<Search fontSize={"small"} />}
          onClick={() => dispatch(openAdvancedSearch(fmIndex))}
        >
          {isMobile
            ? searchConditionsCount
            : t("fileManager.searchConditions", {
                num: searchConditionsCount,
              })}
        </StyledButton>
        <StyledButton size={"small"} onClick={() => dispatch(clearSearch(fmIndex))}>
          <Dismiss fontSize={"small"} sx={{ width: 16, height: 16 }} />
        </StyledButton>
      </StyledButtonGroup>
    </Grow>
  );
};
