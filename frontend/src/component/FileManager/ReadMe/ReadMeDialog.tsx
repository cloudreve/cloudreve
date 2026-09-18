import { Dialog, Slide } from "@mui/material";
import { TransitionProps } from "@mui/material/transitions";
import { forwardRef } from "react";
import { closeShareReadme } from "../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import ReadMeContent from "./ReadMeContent.tsx";

const Transition = forwardRef(function Transition(
  props: TransitionProps & {
    children: React.ReactElement<unknown>;
  },
  ref: React.Ref<unknown>,
) {
  return <Slide direction="up" ref={ref} {...props} />;
});

const ReadMeDialog = () => {
  const dispatch = useAppDispatch();
  const readMeOpen = useAppSelector((state) => state.globalState.shareReadmeOpen);

  return (
    <Dialog
      fullScreen
      TransitionComponent={Transition}
      open={!!readMeOpen}
      onClose={() => {
        dispatch(closeShareReadme());
      }}
    >
      <ReadMeContent />
    </Dialog>
  );
};

export default ReadMeDialog;
