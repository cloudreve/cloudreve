import { Box, styled } from "@mui/material";
import { grey } from "@mui/material/colors";

const StyledCode = styled(Box)(({ theme }) => ({
  backgroundColor: grey[100],
  ...theme.applyStyles("dark", {
    backgroundColor: grey[900],
  }),
  border: `1px solid ${theme.palette.divider}`,
  borderRadius: "4px",
  padding: "1px",
  paddingLeft: "4px",
  paddingRight: "4px",
}));

export const Code = ({ children }: { children?: React.ReactNode }) => {
  return <StyledCode as="code">{children}</StyledCode>;
};
