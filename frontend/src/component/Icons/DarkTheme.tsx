import { SvgIcon, SvgIconProps } from "@mui/material";

export default function DarkTheme(props: SvgIconProps) {
  return (
    <SvgIcon {...props}>
      <path d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10Zm0-2V4a8 8 0 1 1 0 16Z" />
    </SvgIcon>
  );
}
