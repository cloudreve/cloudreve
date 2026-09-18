import { useAppSelector } from "../../redux/hooks.ts";
import PinToSidebar from "../FileManager/Dialogs/PinToSidebar.tsx";
import BatchDownloadLog from "./BatchDownloadLog.tsx";
import Confirmation from "./Confirmation.tsx";
import SelectOption from "./SelectOption.tsx";

const GlobalDialogs = () => {
  const selectOptionOpen = useAppSelector((state) => state.globalState.selectOptionDialogOpen);
  const batchDownloadLogOpen = useAppSelector((state) => state.globalState.batchDownloadLogDialogOpen);
  return (
    <>
      <Confirmation />
      <PinToSidebar />
      {batchDownloadLogOpen != undefined && <BatchDownloadLog />}
      {selectOptionOpen != undefined && <SelectOption />}
    </>
  );
};

export default GlobalDialogs;
