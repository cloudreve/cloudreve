import Add from "../Icons/Add.tsx";
import Delete from "../Icons/Delete.tsx";
import DeleteOutlined from "../Icons/DeleteOutlined.tsx";
import { Button, IconButton, useMediaQuery, useTheme } from "@mui/material";
import { useTranslation } from "react-i18next";
import { useAppDispatch, useAppSelector } from "../../redux/hooks.ts";
import { openNewContextMenu } from "../../redux/thunks/filemanager.ts";
import { emptyTrash } from "../../redux/thunks/file.ts";
import { FileManagerIndex } from "./FileManager.tsx";
import { Filesystem } from "../../util/uri.ts";

const NewButton = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const currentFs = useAppSelector((s) => s.fileManager[FileManagerIndex.main].current_fs);
  const isTrash = currentFs === Filesystem.trash;

  if (isTrash) {
    if (isMobile) {
      return (
        <IconButton onClick={() => dispatch(emptyTrash(FileManagerIndex.main))}>
          <Delete />
        </IconButton>
      );
    }

    return (
      <Button
        variant={"contained"}
        onClick={() => dispatch(emptyTrash(FileManagerIndex.main))}
        startIcon={<DeleteOutlined />}
        color={"primary"}
      >
        {t("fileManager.emptyTrash")}
      </Button>
    );
  }

  if (isMobile) {
    return (
      <IconButton onClick={(e) => dispatch(openNewContextMenu(FileManagerIndex.main, e))}>
        <Add />
      </IconButton>
    );
  }

  return (
    <Button
      variant={"contained"}
      onClick={(e) => dispatch(openNewContextMenu(FileManagerIndex.main, e))}
      startIcon={<Add />}
      color={"primary"}
    >
      {t("fileManager.new")}
    </Button>
  );
};

export default NewButton;
