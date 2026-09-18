import { SvgIcon, SvgIconProps } from "@mui/material";

export default function CheckUnchecked(props: SvgIconProps) {
  return (
    <SvgIcon {...props}>
      <path d="M12 2c5.523 0 10 4.477 10 10s-4.477 10-10 10S2 17.523 2 12 6.477 2 12 2Zm0 1.5a8.5 8.5 0 1 0 0 17 8.5 8.5 0 0 0 0-17Z" />
    </SvgIcon>
  );
}
