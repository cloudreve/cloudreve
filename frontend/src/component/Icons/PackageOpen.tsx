import { SvgIcon, SvgIconProps } from "@mui/material";

export default function PackageOpen(props: SvgIconProps) {
  return (
    <SvgIcon {...props}>
      <path
        fill="none"
        stroke="currentColor"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="1.5"
        d="M12 22c-.818 0-1.6-.335-3.163-1.006C4.946 19.324 3 18.49 3 17.085V7.747M12 22c.818 0 1.6-.335 3.163-1.006C19.054 19.324 21 18.49 21 17.085V7.747M12 22v-9.83m9-4.422c0 .603-.802.984-2.405 1.747l-2.92 1.39C13.87 11.741 12.97 12.17 12 12.17m9-4.423c0-.604-.802-.985-2.405-1.748M3 7.747c0 .604.802.986 2.405 1.748l2.92 1.39c1.804.857 2.705 1.286 3.675 1.286M3 7.748c0-.604.802-.985 2.405-1.748m.927 7.311l1.994.948M12 2v2m4-1l-1.5 2M8 3l1.5 2"
        color="currentColor"
      />
    </SvgIcon>
  );
}
