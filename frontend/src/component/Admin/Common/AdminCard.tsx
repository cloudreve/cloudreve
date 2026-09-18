import { Box, styled } from "@mui/material";

export const BorderedCard = styled(Box)(({ theme }) => ({
  border: `1px solid ${theme.palette.divider}`,
  borderRadius: theme.shape.borderRadius,
  padding: theme.spacing(2),
  backgroundColor: theme.palette.background.paper,
}));

export const BorderedCardClickable = styled(BorderedCard)(({ theme }) => ({
  cursor: "pointer",
  "&:hover": {
    backgroundColor: theme.palette.action.hover,
  },
  transition: "background-color 0.3s ease",
}));

export const BorderedCardClickableBaImg = styled(BorderedCardClickable)<{ img?: string }>(({ theme, img }) => ({
  position: "relative",
  overflow: "hidden",
  "&::before": {
    content: '""',
    position: "absolute",
    top: "-20px",
    right: "-20px",
    width: "150px",
    height: "150px",
    backgroundImage: `url(${img})`,
    backgroundSize: "cover",
    backgroundPosition: "center",
    transform: "rotate(-10deg)",
    opacity: 0.1,
    maskImage: "radial-gradient(circle at center, black 30%, transparent 80%)",
    pointerEvents: "none",
    zIndex: 0,
  },
  "& > *": {
    position: "relative",
    zIndex: 1,
  },
}));
