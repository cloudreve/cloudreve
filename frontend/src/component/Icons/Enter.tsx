import { SvgIcon, SvgIconProps } from "@mui/material";

export default function Enter(props: SvgIconProps) {
  return (
    <SvgIcon {...props}>
      <path
        d="M21.25 4a.75.75 0 0 1 .75.75v6.5A3.75 3.75 0 0 1 18.25 15H4.586l3.72 3.72a.75.75 0 0 1 .072.976l-.073.084a.75.75 0 0 1-.976.073l-.084-.073l-5-5a.75.75 0 0 1-.073-.976l.073-.084l5-5a.75.75 0 0 1 1.133.976l-.073.084l-3.72 3.72H18.25a2.25 2.25 0 0 0 2.245-2.096l.005-.154v-6.5a.75.75 0 0 1 .75-.75z"
        fill-rule="nonzero"
      />
    </SvgIcon>
  );
}
