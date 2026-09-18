import { SvgIcon, SvgIconProps } from "@mui/material";

export default function DocumentText(props: SvgIconProps) {
  return (
    <SvgIcon {...props}>
      <path d="M12 8V2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V10h-6a2 2 0 0 1-2-2Zm-3.75 3.5h7.5a.75.75 0 0 1 0 1.5h-7.5a.75.75 0 0 1 0-1.5Zm0 2.75h7.5a.75.75 0 0 1 0 1.5h-7.5a.75.75 0 0 1 0-1.5Zm0 2.75h7.5a.75.75 0 0 1 0 1.5h-7.5a.75.75 0 0 1 0-1.5Zm5.25-9V2.5l6 6H14a.5.5 0 0 1-.5-.5Z" />
    </SvgIcon>
  );
}
