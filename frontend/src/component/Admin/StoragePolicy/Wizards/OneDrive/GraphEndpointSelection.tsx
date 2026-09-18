import { Box, OutlinedSelectProps, Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import { DenseSelect } from "../../../../Common/StyledComponents";
import { SquareMenuItem } from "../../../../FileManager/ContextMenu/ContextMenu";

export interface GraphEndpointSelectionProps extends OutlinedSelectProps {
  value: string;
  onChange: (value: string) => void;
}

interface GraphEndpoint {
  name: string;
  endpoint: string;
}

const graphEndpoints: GraphEndpoint[] = [
  { name: "policy.multiTenant", endpoint: "https://graph.microsoft.com/v1.0" },
  { name: "policy.gallatin", endpoint: "https://microsoftgraph.chinacloudapi.cn/v1.0" },
];

const GraphEndpointSelection = ({ value, onChange, ...rest }: GraphEndpointSelectionProps) => {
  const { t } = useTranslation("dashboard");

  return (
    <DenseSelect
      value={value}
      onChange={(e) => onChange(e.target.value as string)}
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
      {graphEndpoints.map((endpoint) => (
        <SquareMenuItem value={endpoint.endpoint}>
          <Box
            sx={{
              display: "flex",
              flexDirection: "column",
            }}
          >
            <Typography variant={"body2"} fontWeight={600}>
              {t(endpoint.name)}
            </Typography>
            <Typography variant={"caption"} color={"textSecondary"}>
              {endpoint.endpoint}
            </Typography>
          </Box>
        </SquareMenuItem>
      ))}
    </DenseSelect>
  );
};

export default GraphEndpointSelection;
