import { Box, Typography } from "@mui/material";

export interface StyledFormControlProps {
  title?: React.ReactNode;
  children: React.ReactNode;
}

const StyledFormControl = ({ title, children }: StyledFormControlProps) => {
  return (
    <Box>
      {title && (
        <Typography fontWeight={600} sx={{ mb: 0.5 }} variant={"body2"}>
          {title}
        </Typography>
      )}
      {children}
    </Box>
  );
};

export default StyledFormControl;
