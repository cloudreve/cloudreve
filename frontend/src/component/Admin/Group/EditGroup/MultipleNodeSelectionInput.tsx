import { Box, Checkbox, FormControl, ListItemText, SelectChangeEvent } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getNodeList } from "../../../../api/api";
import { Node } from "../../../../api/dashboard";
import { useAppDispatch } from "../../../../redux/hooks";
import { DenseSelect, SquareChip } from "../../../Common/StyledComponents";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu";

export interface MultipleNodeSelectionInputProps {
  value: number[];
  onChange: (value: number[]) => void;
}

// MultipleNodeSelectionInput picks the pool of nodes a group's tasks may run
// on. Empty means all nodes are eligible.
const MultipleNodeSelectionInput = ({ value, onChange }: MultipleNodeSelectionInputProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(false);
  const [nodeMap, setNodeMap] = useState<Record<number, Node>>({});

  const handleChange = (event: SelectChangeEvent<unknown>) => {
    const {
      target: { value: v },
    } = event;
    onChange(typeof v === "string" ? v.split(",").map((x) => parseInt(x)) : (v as number[]));
  };

  useEffect(() => {
    setLoading(true);
    dispatch(getNodeList({ page: 1, page_size: 1000, order_by: "id", order_direction: "asc" }))
      .then((res) => {
        setNodes(res.nodes);
        setNodeMap(
          res.nodes.reduce(
            (acc, n) => {
              acc[n.id] = n;
              return acc;
            },
            {} as Record<number, Node>,
          ),
        );
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  return (
    <FormControl fullWidth>
      <DenseSelect
        multiple
        value={value}
        onChange={handleChange}
        displayEmpty
        sx={{
          minHeight: 39,
        }}
        disabled={loading}
        MenuProps={{
          PaperProps: { sx: { maxWidth: 300 } },
        }}
        renderValue={(selected) =>
          (selected as number[]).length === 0 ? (
            <ListItemText
              primary={<em>{t("group.allNodes")}</em>}
              slotProps={{
                primary: { color: "textSecondary", variant: "body2" },
              }}
            />
          ) : (
            <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5, py: 0.5 }}>
              {(selected as number[]).map((id) => (
                <SquareChip key={id} label={nodeMap[id]?.name ?? id} size="small" />
              ))}
            </Box>
          )
        }
      >
        {nodes.map((n) => (
          <SquareMenuItem key={n.id} value={n.id}>
            <Checkbox size="small" checked={value.indexOf(n.id) > -1} />
            <ListItemText primary={n.name} secondary={`#${n.id} · ${n.type ?? ""}`} />
          </SquareMenuItem>
        ))}
      </DenseSelect>
    </FormControl>
  );
};

export default MultipleNodeSelectionInput;
