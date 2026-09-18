import { Box, createFilterOptions, debounce, useTheme } from "@mui/material";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getSearchUser } from "../../../api/api.ts";
import { User } from "../../../api/user.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import {
  DenseAutocomplete,
  DenseFilledTextField,
  NoWrapBox,
  NoWrapTypography,
} from "../../Common/StyledComponents.tsx";
import UserAvatar from "../../Common/User/UserAvatar.tsx";

export interface UserSearchInputProps {
  onUserSelected: (user: User) => void;
  label?: string;
  required?: boolean;
}

const UserSearchInput = (props: UserSearchInputProps) => {
  const theme = useTheme();
  const { t } = useTranslation(["dashboard", "application"]);
  const [value, setValue] = useState<User | null>(null);
  const [inputValue, setInputValue] = useState("");
  const [options, setOptions] = useState<readonly User[]>([]);
  const [loading, setLoading] = useState(false);

  const dispatch = useAppDispatch();

  const fetch = useMemo(
    () =>
      debounce((request: { input: string }, callback: (results?: readonly User[]) => void) => {
        setLoading(true);
        dispatch(getSearchUser(request.input))
          .then((results: readonly User[]) => {
            callback(results);
          })
          .finally(() => {
            setLoading(false);
          });
      }, 400),
    [dispatch],
  );

  useEffect(() => {
    let active = true;

    if (inputValue === "" || inputValue.length < 2) {
      setOptions([]);
      return undefined;
    }

    fetch({ input: inputValue }, (results?: readonly User[]) => {
      if (active) {
        setOptions(results ?? []);
      }
    });

    return () => {
      active = false;
    };
  }, [value, inputValue, fetch]);

  const filterOptions = useMemo(() => {
    return createFilterOptions<User>({
      stringify: (option) => option.nickname + " " + option.email,
    });
  }, []);

  return (
    <DenseAutocomplete
      value={value}
      filterOptions={filterOptions}
      options={options}
      loading={loading}
      blurOnSelect
      onChange={(_event: any, newValue: User | null) => {
        setValue(newValue);
        if (newValue) {
          props.onUserSelected(newValue);
        }
      }}
      onInputChange={(_event, newInputValue) => {
        setInputValue(newInputValue);
      }}
      getOptionLabel={(option) => (typeof option === "string" ? option : `${option.nickname} <${option.email}>`)}
      noOptionsText={t("application:modals.noResults")}
      renderOption={(props, option) => {
        return (
          <li {...props}>
            <Box sx={{ display: "flex", width: "100%", alignItems: "center" }}>
              <Box>
                <UserAvatar user={option} />
              </Box>
              <NoWrapBox
                sx={{
                  width: "100%",
                  ml: 2,
                }}
              >
                {option.nickname}
                {option.email && (
                  <NoWrapTypography
                    sx={{
                      width: "100%",
                    }}
                    variant="body2"
                    color="text.secondary"
                  >
                    {option.email}
                  </NoWrapTypography>
                )}
              </NoWrapBox>
            </Box>
          </li>
        );
      }}
      renderInput={(params) => (
        <DenseFilledTextField
          {...params}
          sx={{
            "& .MuiInputBase-root": {},
            "& .MuiInputBase-root.MuiOutlinedInput-root": {
              paddingTop: theme.spacing(0.6),
              paddingBottom: theme.spacing(0.6),
            },
            mt: 0,
          }}
          required={props.required}
          variant="outlined"
          margin="dense"
          placeholder={props.label || t("dashboard:file.selectUser")}
          type="text"
          fullWidth
        />
      )}
    />
  );
};

export default UserSearchInput;
