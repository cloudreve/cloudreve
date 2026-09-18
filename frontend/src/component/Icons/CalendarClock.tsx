import { SvgIcon, SvgIconProps } from "@mui/material";

export default function CalendarClock(props: SvgIconProps) {
  return (
    <SvgIcon {...props}>
      <path d="M17.75 3A3.25 3.25 0 0 1 21 6.25v5.772a6.471 6.471 0 0 0-1.5-.709V8.5h-15v9.25c0 .966.784 1.75 1.75 1.75h5.063c.173.534.412 1.037.709 1.5H6.25A3.25 3.25 0 0 1 3 17.75V6.25A3.25 3.25 0 0 1 6.25 3zm0 1.5H6.25A1.75 1.75 0 0 0 4.5 6.25V7h15v-.75a1.75 1.75 0 0 0-1.75-1.75m5.25 13a5.5 5.5 0 1 0-11 0a5.5 5.5 0 0 0 11 0M17.5 14a.5.5 0 0 1 .5.5V17h2a.5.5 0 0 1 0 1h-2.5a.5.5 0 0 1-.5-.5v-3a.5.5 0 0 1 .5-.5" />
    </SvgIcon>
  );
}
