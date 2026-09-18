import { SvgIcon, SvgIconProps } from "@mui/material";

export default function Checkmark(props: SvgIconProps) {
  return (
    <SvgIcon {...props}>
      <path d="m8.5 16.586-3.793-3.793a1 1 0 0 0-1.414 1.414l4.5 4.5a1 1 0 0 0 1.414 0l11-11a1 1 0 0 0-1.414-1.414L8.5 16.586Z" />
    </SvgIcon>
  );
}
