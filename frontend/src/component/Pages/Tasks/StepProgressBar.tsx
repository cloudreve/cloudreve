import { Box, Skeleton, Typography } from "@mui/material";
import BorderLinearProgress from "../../Common/BorderLinearProgress.tsx";

export interface StepProgressBarProps {
  title?: string;
  secondary?: string;
  caption?: string;
  progress?: number;
  loading?: boolean;
  indeterminate?: boolean;
}

const StepProgressBar = ({ title, secondary, progress, loading, indeterminate }: StepProgressBarProps) => {
  return (
    <Box>
      <Box sx={{ display: "flex", justifyContent: "space-between" }}>
        <Typography variant={"caption"} noWrap>
          {loading ? <Skeleton variant={"text"} width={100} /> : title}
        </Typography>
        {secondary && (
          <Typography noWrap sx={{ ml: 1 }} variant={"caption"} color={"text.secondary"}>
            {secondary}
          </Typography>
        )}
      </Box>
      {loading ? (
        <Skeleton variant={"rectangular"} width={200} height={8} />
      ) : (
        <BorderLinearProgress variant={indeterminate ? "indeterminate" : "determinate"} value={progress} />
      )}
    </Box>
  );
};

export default StepProgressBar;
