import { InputAdornment, TextField, TextFieldProps, useMediaQuery, useTheme } from "@mui/material";

export interface OutlineIconTextFieldProps extends TextFieldProps<"outlined"> {
  icon: React.ReactNode;
}

export const OutlineIconTextField = ({ icon, ...rest }: OutlineIconTextFieldProps) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  return (
    <TextField
      {...rest}
      slotProps={{
        input: {
          startAdornment: !isMobile && <InputAdornment position="start">{icon}</InputAdornment>,
          ...rest.InputProps,
        },
      }}
    />
  );
};
