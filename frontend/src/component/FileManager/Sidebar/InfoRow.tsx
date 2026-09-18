import { Box, Typography } from "@mui/material";
import React from "react";

export interface InfoRowProps {
  title: string;
  content: React.ReactNode | string;
}

const InfoRow = ({ title, content }: InfoRowProps) => {
  return (
    <Box>
      <Typography variant={"body2"} color="textPrimary" fontWeight={500}>
        {title}
      </Typography>
      <Typography variant={"body2"} color={"text.secondary"}>
        {content}
      </Typography>
    </Box>
  );
};

export default InfoRow;
