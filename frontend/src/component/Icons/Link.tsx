import { SvgIcon, SvgIconProps } from "@mui/material";

export default function Link(props: SvgIconProps) {
  return (
    <SvgIcon {...props}>
      <path d="M9 7a1 1 0 0 1 .117 1.993L9 9H7a3 3 0 0 0-.176 5.995L7 15h2a1 1 0 0 1 .117 1.993L9 17H7a5 5 0 0 1-.217-9.995L7 7h2Zm8 0a5 5 0 0 1 .217 9.995L17 17h-2a1 1 0 0 1-.117-1.993L15 15h2a3 3 0 0 0 .176-5.995L17 9h-2a1 1 0 0 1-.117-1.993L15 7h2ZM7 11h10a1 1 0 0 1 .117 1.993L17 13H7a1 1 0 0 1-.117-1.993L7 11h10H7Z" />
    </SvgIcon>
  );
}
