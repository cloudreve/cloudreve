import { Box, Checkbox, FormControl, ListItemText, SelectChangeEvent } from "@mui/material";
import { useEffect, useState } from "react";
import { getStoragePolicyList } from "../../../../api/api";
import { StoragePolicy } from "../../../../api/dashboard";
import { useAppDispatch } from "../../../../redux/hooks";
import { DenseSelect, SquareChip } from "../../../Common/StyledComponents";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu";

export interface PolicyMultiSelectionInputProps {
  value: number[];
  onChange: (value: number[]) => void;
}

// PolicyMultiSelectionInput picks the pool of storage policies members of a
// group may freely switch between. The group's default policy is always
// available implicitly and need not be selected here.
const PolicyMultiSelectionInput = ({ value, onChange }: PolicyMultiSelectionInputProps) => {
  const dispatch = useAppDispatch();
  const [policies, setPolicies] = useState<StoragePolicy[]>([]);
  const [loading, setLoading] = useState(false);
  const [policyMap, setPolicyMap] = useState<Record<number, StoragePolicy>>({});

  const handleChange = (event: SelectChangeEvent<unknown>) => {
    const {
      target: { value: v },
    } = event;
    onChange(typeof v === "string" ? v.split(",").map((x) => parseInt(x)) : (v as number[]));
  };

  useEffect(() => {
    setLoading(true);
    dispatch(getStoragePolicyList({ page: 1, page_size: 1000, order_by: "id", order_direction: "asc" }))
      .then((res) => {
        setPolicies(res.policies);
        setPolicyMap(
          res.policies.reduce(
            (acc, policy) => {
              acc[policy.id] = policy;
              return acc;
            },
            {} as Record<number, StoragePolicy>,
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
        sx={{
          minHeight: 39,
        }}
        disabled={loading}
        MenuProps={{
          PaperProps: { sx: { maxWidth: 300 } },
        }}
        renderValue={(selected) => (
          <Box sx={{ display: "flex", flexWrap: "wrap", gap: 0.5, py: 0.5 }}>
            {(selected as number[]).map((id) => (
              <SquareChip key={id} label={policyMap[id]?.name ?? id} size="small" />
            ))}
          </Box>
        )}
      >
        {policies.map((p) => (
          <SquareMenuItem key={p.id} value={p.id}>
            <Checkbox size="small" checked={value.indexOf(p.id) > -1} />
            <ListItemText primary={p.name} secondary={`#${p.id} · ${p.type}`} />
          </SquareMenuItem>
        ))}
      </DenseSelect>
    </FormControl>
  );
};

export default PolicyMultiSelectionInput;
