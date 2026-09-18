import { SvgIcon, SvgIconProps } from "@mui/material";

export default function Raw(props: SvgIconProps) {
  return (
    <SvgIcon {...props}>
      <path d="M6.5 9H3v6h1.5v-2h1.1l.9 2H8l-.9-2.1c.5-.3.9-.8.9-1.4v-1C8 9.7 7.3 9 6.5 9m0 2.5h-2v-1h2zM10.25 9l-1.5 6h1.5l.38-1.5h1.75l.37 1.5h1.5l-1.5-6zm.75 3l.25-1h.5l.25 1zm8.98-3l-.74 3l-.74-3h-1.52l-.74 3l-.74-3H14l1.5 6h1.48l.76-3.04l.76 3.04h1.48l1.5-6z" />
    </SvgIcon>
  );
}
