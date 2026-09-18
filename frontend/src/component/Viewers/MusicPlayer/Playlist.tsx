import { ListItemIcon, ListItemText, MenuProps } from "@mui/material";
import { FileResponse } from "../../../api/explorer.ts";
import { SquareMenu, SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import FileIcon from "../../FileManager/Explorer/FileIcon.tsx";

export interface PlaylistProps extends MenuProps {
  file: FileResponse;
  playlist: FileResponse[];
  playIndex: (index: number, volume?: number) => void;
}

const Playlist = ({ file, playlist, playIndex, onClose, ...rest }: PlaylistProps) => {
  return (
    <SquareMenu
      MenuListProps={{
        dense: true,
      }}
      anchorOrigin={{
        vertical: "bottom",
        horizontal: "right",
      }}
      transformOrigin={{
        vertical: "top",
        horizontal: "right",
      }}
      onClose={onClose}
      {...rest}
    >
      {playlist.map((item, index) => (
        <SquareMenuItem key={item.id} onClick={() => playIndex(index)} selected={item.path == file.path}>
          <ListItemIcon>
            <FileIcon
              sx={{ px: 0, py: 0, height: "20px" }}
              file={item}
              variant={"small"}
              iconProps={{
                fontSize: "small",
              }}
            />
          </ListItemIcon>
          <ListItemText>{item.name}</ListItemText>
        </SquareMenuItem>
      ))}
    </SquareMenu>
  );
};

export default Playlist;
