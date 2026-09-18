import { Alert, Box, OutlinedSelectProps, Typography } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { getNodeList } from "../../../api/api";
import { Node, NodeStatus, NodeType } from "../../../api/dashboard";
import { useAppDispatch } from "../../../redux/hooks";
import { DenseSelect } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu";

export interface NodeSelectionInputProps extends OutlinedSelectProps {
  value: number;
  onChange: (value: number) => void;
}

export const NodeStatusCondition = "node_status";

const NodeSelectionInput = ({ value, onChange, ...rest }: NodeSelectionInputProps) => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(true);
  const [nodes, setNodes] = useState<Node[]>([]);

  useEffect(() => {
    setLoading(true);
    dispatch(
      getNodeList({
        page_size: 1000,
        page: 1,
        order_by: "id",
        order_direction: "desc",
        conditions: {
          [NodeStatusCondition]: NodeStatus.active,
        },
      }),
    )
      .then((res) => {
        const filteredNodes = res.nodes.filter((n) => n.type != NodeType.master);
        setNodes(filteredNodes);
        if (!value && filteredNodes.length > 0) {
          onChange(filteredNodes[0].id);
        }
      })
      .finally(() => {
        setLoading(false);
      });
  }, []);

  if (!loading && nodes.length == 0) {
    return <Alert severity="warning">{t("settings.noNodes")}</Alert>;
  }
  return (
    <DenseSelect
      disabled={loading}
      value={value}
      onChange={(e) => onChange(e.target.value as number)}
      MenuProps={{
        PaperProps: { sx: { maxWidth: 230 } },
        MenuListProps: {
          sx: {
            "& .MuiMenuItem-root": {
              whiteSpace: "normal",
            },
          },
        },
      }}
      {...rest}
    >
      {nodes.map((g) => (
        <SquareMenuItem value={g.id}>
          <Box
            sx={{
              display: "flex",
              flexDirection: "column",
            }}
          >
            <Typography variant={"body2"} fontWeight={600}>
              {g.name}
            </Typography>
            <Typography variant={"caption"} color={"textSecondary"}>
              {g.server}
            </Typography>
          </Box>
        </SquareMenuItem>
      ))}
    </DenseSelect>
  );
};

export default NodeSelectionInput;
