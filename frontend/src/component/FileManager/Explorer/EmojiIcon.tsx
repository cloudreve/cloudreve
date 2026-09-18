import { SvgIconProps, Typography } from "@mui/material";

export interface EmojiIconProps extends SvgIconProps {
  emoji: string;
}

const EmojiIcon = ({ sx, fontSize, emoji, ...rest }: EmojiIconProps) => {
  return (
    <Typography
      sx={{
        color: (theme) => theme.palette.text.primary,
        minWidth: "24px",
        pl: "4px",
        ...sx,
      }}
      fontSize={fontSize}
      {...rest}
    >
      {emoji}
    </Typography>
  );
};

export default EmojiIcon;
