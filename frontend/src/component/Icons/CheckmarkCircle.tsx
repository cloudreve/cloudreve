import { SvgIcon, SvgIconProps } from "@mui/material";

export default function CheckmarkCircle(props: SvgIconProps) {
  return (
    <SvgIcon {...props}>
      <path d="M12 2c5.523 0 10 4.477 10 10s-4.477 10-10 10S2 17.523 2 12 6.477 2 12 2Zm3.22 6.97-4.47 4.47-1.97-1.97a.75.75 0 0 0-1.06 1.06l2.5 2.5a.75.75 0 0 0 1.06 0l5-5a.75.75 0 1 0-1.06-1.06Z" />
    </SvgIcon>
  );
}
