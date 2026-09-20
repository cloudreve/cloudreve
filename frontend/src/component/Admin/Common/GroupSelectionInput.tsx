import { ListItemText } from "@mui/material";
import Checkbox from "@mui/material/Checkbox";
import FormControl from "@mui/material/FormControl";
import { useEffect, useState } from "react";
import { getGroupList } from "../../../api/api";
import { GroupEnt } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { DenseSelect } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu";

export interface GroupSelectionInputProps {
  value: string | string[];
  onChange: (value: string) => void;
  onChangeMulti?: (value: string[]) => void;
  onChangeGroup?: (group?: GroupEnt) => void;
  emptyValue?: string;
  emptyText?: string;
  fullWidth?: boolean;
  required?: boolean;
  multiple?: boolean;
}

const AnonymousGroupId = 3;

const GroupSelectionInput = ({
  value,
  onChange,
  onChangeMulti,
  onChangeGroup,
  emptyValue,
  emptyText,
  fullWidth,
  required,
  multiple,
}: GroupSelectionInputProps) => {
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(true);
  const [groups, setGroups] = useState<GroupEnt[]>([]);

  useEffect(() => {
    setLoading(true);
    dispatch(
      getGroupList({
        page_size: 1000,
        page: 1,
        order_by: "id",
        order_direction: "asc",
      }),
    )
      .then((res) => {
        setGroups(res.groups);
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  const handleChange = (v: string | string[]) => {
    if (multiple) {
      onChangeMulti?.(Array.isArray(v) ? v : [v]);
      return;
    }
    const sv = Array.isArray(v) ? v[0] : v;
    onChange(sv);
    onChangeGroup?.(groups.find((g) => g.id === parseInt(sv)));
  };

  const multiValue = Array.isArray(value) ? value : value ? [value] : [];

  return (
    <FormControl fullWidth={fullWidth}>
      <DenseSelect
        disabled={loading}
        multiple={multiple}
        value={multiple ? multiValue : Array.isArray(value) ? (value[0] ?? "") : value}
        onChange={(e) => handleChange(e.target.value as string | string[])}
        required={required}
        renderValue={
          multiple
            ? (selected) =>
                (selected as string[])
                  .map((id) => groups.find((g) => g.id === parseInt(id))?.name ?? id)
                  .join(", ")
            : undefined
        }
      >
        {groups
          .filter((g) => g.id != AnonymousGroupId)
          .map((g) => (
            <SquareMenuItem key={g.id} value={g.id.toString()}>
              {multiple && <Checkbox checked={multiValue.indexOf(g.id.toString()) > -1} size="small" />}
              <ListItemText
                slotProps={{
                  primary: { variant: "body2" },
                }}
              >
                {g.name}
              </ListItemText>
            </SquareMenuItem>
          ))}
        {emptyValue !== undefined && emptyText && (
          <SquareMenuItem value={emptyValue}>
            <ListItemText
              primary={<em>{emptyText}</em>}
              slotProps={{
                primary: { variant: "body2" },
              }}
            />
          </SquareMenuItem>
        )}
      </DenseSelect>
    </FormControl>
  );
};

export default GroupSelectionInput;
