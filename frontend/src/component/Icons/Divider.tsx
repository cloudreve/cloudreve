import { SvgIcon, SvgIconProps } from "@mui/material";

export default function Divider(props: SvgIconProps) {
  return (
    <SvgIcon {...props}>
      <path d="M11.25 2.75v18.5a.75.75 0 0 0 1.5 0V2.75a.75.75 0 0 0-1.5 0" />
    </SvgIcon>
  );
}
