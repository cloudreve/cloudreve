import { SvgIcon, SvgIconProps } from "@mui/material";

export default function Document(props: SvgIconProps) {
  return (
    <SvgIcon {...props}>
      <path d="M12 2v6a2 2 0 0 0 2 2h6v10a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h6Z" />
      <path d="M13.5 2.5V8a.5.5 0 0 0 .5.5h5.5l-6-6Z" />
    </SvgIcon>
  );
}
