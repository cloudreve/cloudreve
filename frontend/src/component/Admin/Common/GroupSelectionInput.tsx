import { ListItemText } from "@mui/material";
import FormControl from "@mui/material/FormControl";
import { useEffect, useState } from "react";
import { getGroupList } from "../../../api/api";
import { GroupEnt } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { DenseSelect } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu";

export interface GroupSelectionInputProps {
  value: string;
  onChange: (value: string) => void;
  onChangeGroup?: (group?: GroupEnt) => void;
  emptyValue?: string;
  emptyText?: string;
  fullWidth?: boolean;
  required?: boolean;
}

const AnonymousGroupId = 3;

const GroupSelectionInput = ({
  value,
  onChange,
  onChangeGroup,
  emptyValue,
  emptyText,
  fullWidth,
  required,
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

  const handleChange = (value: string) => {
    onChange(value);
    onChangeGroup?.(groups.find((g) => g.id === parseInt(value)));
  };

  return (
    <FormControl fullWidth={fullWidth}>
      <DenseSelect
        disabled={loading}
        value={value}
        onChange={(e) => handleChange(e.target.value as string)}
        required={required}
      >
        {groups
          .filter((g) => g.id != AnonymousGroupId)
          .map((g) => (
            <SquareMenuItem value={g.id.toString()}>
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
