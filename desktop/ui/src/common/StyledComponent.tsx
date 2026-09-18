import { Button, TextField, alpha, styled, type ButtonProps } from "@mui/material";

export const DefaultButton = styled(({ variant, ...rest }: ButtonProps) => <Button variant={variant} {...rest} />)(
  ({ variant, theme }) => ({
    color: theme.palette.text.primary,
    minHeight: theme.spacing(4),
    "& .MuiButton-startIcon": {
      marginLeft: 0,
    },
    border: variant == "outlined" ? `1px solid ${theme.palette.divider}` : "none",
  }),
);

export const SecondaryButton = styled(Button)(({ theme }) => ({
  color: theme.palette.text.secondary,
  paddingLeft: theme.spacing(1.5),
  paddingRight: theme.spacing(1.5),
  backgroundColor: theme.palette.action.hover,
  "&:hover": {
    backgroundColor: theme.palette.action.focus,
  },
}));

export const SecondaryErrorButton = styled(Button)(({ theme }) => ({
  color: theme.palette.error.main,
  backgroundColor: alpha(theme.palette.error.light, 0.1),
  paddingLeft: theme.spacing(1.5),
  paddingRight: theme.spacing(1.5),
  "&:hover": {
    backgroundColor: alpha(theme.palette.error.light, 0.2),
  },
}));


export const FilledTextField = styled(TextField)(() => ({
  "& .Mui-disabled:before": {
    border: "none",
  },
}));

export const DenseFilledTextField = styled(FilledTextField)(({ theme }) => ({
  "& .MuiOutlinedInput-input": {
    paddingTop: theme.spacing(1.2),
    paddingBottom: theme.spacing(1.2),
    fontSize: theme.typography.body2.fontSize,
  },
  "& .MuiInputBase-root.MuiOutlinedInput-root": {
    paddingTop: 0,
    paddingBottom: 0,
    fontSize: theme.typography.body2.fontSize,
  },
  "& .MuiInputLabel-root": {
    fontSize: theme.typography.body2.fontSize,
    // no class .Mui-focused
    "&:not(.Mui-focused):not(.MuiInputLabel-shrink)": {
      transform: "translate(14px, 10px) scale(1)",
    },
  },
}));
