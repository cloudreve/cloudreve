import { SvgIcon, SvgIconProps } from "@mui/material";

export default function ChevronRight(props: SvgIconProps) {
  return (
    <SvgIcon {...props}>
      <path d="M8.293 4.293a1 1 0 0 0 0 1.414L14.586 12l-6.293 6.293a1 1 0 1 0 1.414 1.414l7-7a1 1 0 0 0 0-1.414l-7-7a1 1 0 0 0-1.414 0Z" />
    </SvgIcon>
  );
}
