import { CSSTransition, SwitchTransition } from "react-transition-group";
import { useAppSelector } from "../../../redux/hooks.ts";
import { useMemo } from "react";
import { Box } from "@mui/material";
import SearchBar from "./SearchBar.tsx";
import FileSelectedActions from "./FileSelectedActions.tsx";
import { FileManagerIndex } from "../../FileManager/FileManager.tsx";

const NavBarMainActions = () => {
  const selected = useAppSelector((state) => state.fileManager[FileManagerIndex.main].selected);
  const targets = useMemo(() => {
    return Object.keys(selected).map((key) => selected[key]);
  }, [selected]);
  return (
    <>
      <SwitchTransition>
        <CSSTransition
          addEndListener={(node, done) => node.addEventListener("transitionend", done, false)}
          classNames="fade"
          key={`${targets.length > 0}`}
        >
          <Box sx={{ height: "100%" }}>
            {targets.length == 0 && <SearchBar />}
            {targets.length > 0 && <FileSelectedActions targets={targets} />}
          </Box>
        </CSSTransition>
      </SwitchTransition>
    </>
  );
};

export default NavBarMainActions;
