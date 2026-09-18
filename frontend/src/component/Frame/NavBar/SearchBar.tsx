import { alpha, Button, IconButton, styled, Typography, useMediaQuery, useTheme } from "@mui/material";
import { useHotkeys } from "react-hotkeys-hook";
import { Trans, useTranslation } from "react-i18next";
import { setSearchPopup } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import Search from "../../Icons/Search.tsx";

export const KeyIndicator = styled("code")(({ theme }) => ({
  backgroundColor: theme.palette.mode === "light" ? theme.palette.grey[100] : theme.palette.grey[900],
  border: `1px solid ${theme.palette.divider}`,
  boxShadow:
    theme.palette.mode === "light"
      ? "0 1px 1px rgba(0, 0, 0, 0.2), 0 2px 0 0 rgba(255, 255, 255, 0.7) inset"
      : "0 1px 1px rgba(0, 0, 0, 0.2), 0 2px 0 0 #3d3e42 inset",
  padding: theme.spacing(0, 0.5),
  borderRadius: 4,
}));

const SearchButton = styled(Button)(({ theme }) => ({
  backgroundColor: theme.palette.background.default,
  color: theme.palette.text.disabled,
  border: `1px solid ${theme.palette.divider}`,
  pl: 2,
  pr: 8,
  " :hover": {
    border: `1px solid ${theme.palette.primary.main}`,
    backgroundColor: alpha(theme.palette.primary.main, 0.04),
  },
}));

const SearchBar = () => {
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const { t } = useTranslation();
  useHotkeys(
    "/",
    () => {
      dispatch(setSearchPopup(true));
    },
    { preventDefault: true },
  );

  if (isMobile) {
    return (
      <IconButton onClick={() => dispatch(setSearchPopup(true))}>
        <Search />
      </IconButton>
    );
  }

  return (
    <SearchButton
      sx={(theme) => ({
        backgroundColor: theme.palette.background.default,
        color: theme.palette.text.disabled,
        border: `1px solid ${theme.palette.divider}`,
        pl: 2,
        pr: 8,
        height: "100%",
      })}
      onClick={() => dispatch(setSearchPopup(true))}
      variant={"outlined"}
      startIcon={<Search color={"primary"} />}
    >
      <Trans
        ns={"application"}
        i18nKey={"navbar.searchPlaceholder"}
        components={[
          <KeyIndicator sx={{ mx: 0.5 }}>
            <Typography variant={"body2"} />
          </KeyIndicator>,
        ]}
      />
    </SearchButton>
  );
};

export default SearchBar;
