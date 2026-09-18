import { Box, ListItemText, useMediaQuery, useTheme } from "@mui/material";
import { forwardRef, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { DenseSelect, NoWrapTypography } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu";
import ArrowLeft from "../../Icons/ArrowLeft";
import GlobeFilled from "../../Icons/GlobeFilled";
import Home from "../../Icons/Home";
import Person from "../../Icons/Person";
import Storage from "../../Icons/Storage";
import WindowApps from "../../Icons/WindowApps";
import { BorderedCard } from "../Common/AdminCard";

export interface TrafficDiagramProps {
  variant: "upload" | "download";
  proxyed?: boolean;
  cdn?: boolean;
  storageNodeTitle?: string;
  internalEndpoint?: boolean;
  proxyNodeTitle?: string;
}

enum Source {
  web = "web",
  dav = "dav",
  web_edit = "web_edit",
  wopi = "wopi",
  encrypted_file = "encrypted_file",
}

enum Node {
  cloudreve = "cloudreve",
  proxy = "proxy",
  storage_node = "storage_node",
  storage_node_internal = "storage_node_internal",
  user = "user",
  arrow = "arrow",
  wopi = "wopi",
}

interface NodeIconProps extends TrafficDiagramProps {
  type: Node;
  size?: number;
}

const NodeIcon = forwardRef(({ type, size = 40, storageNodeTitle, proxyNodeTitle, variant }: NodeIconProps, ref) => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const icon = useMemo(() => {
    switch (type) {
      case Node.cloudreve:
        return <Home sx={{ width: size, height: size, color: (t) => t.palette.action.active }} />;
      case Node.proxy:
        return <GlobeFilled sx={{ width: size, height: size, color: (t) => t.palette.action.active }} />;
      case Node.storage_node:
      case Node.storage_node_internal:
        return <Storage sx={{ width: size, height: size, color: (t) => t.palette.action.active }} />;
      case Node.user:
        return <Person sx={{ width: size, height: size, color: (t) => t.palette.action.active }} />;
      case Node.wopi:
        return <WindowApps sx={{ width: size, height: size, color: (t) => t.palette.action.active }} />;
      case Node.arrow:
        return (
          <ArrowLeft
            sx={{
              width: size - 10,
              mx: 1,
              height: size - 10,
              color: (t) => t.palette.action.disabled,
              transform:
                variant == "upload"
                  ? isMobile
                    ? "rotate(-90deg)"
                    : "rotate(180deg)"
                  : isMobile
                    ? "rotate(90deg)"
                    : "rotate(0deg)",
            }}
          />
        );
    }
  }, [type, isMobile]);

  const title = useMemo(() => {
    switch (type) {
      case Node.cloudreve:
        return "Cloudreve";
      case Node.proxy:
        return proxyNodeTitle ?? t("policy.customProxy");
      case Node.storage_node:
        return storageNodeTitle ?? t("policy.storageNode");
      case Node.storage_node_internal:
        return t("policy.storageNodeInternal");
      case Node.user:
        return t("nav.users");
      case Node.wopi:
        return t("settings.wopiViewer");
    }
  }, [type]);

  return (
    <Box sx={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 1 }} ref={ref}>
      {icon}
      {title && (
        <NoWrapTypography color="text.secondary" variant="body2">
          {title}
        </NoWrapTypography>
      )}
    </Box>
  );
});

export const TrafficDiagram = ({
  variant,
  cdn,
  proxyed,
  internalEndpoint,
  storageNodeTitle,
  proxyNodeTitle,
}: TrafficDiagramProps) => {
  const { t } = useTranslation("dashboard");
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down("sm"));
  const [source, setSource] = useState<Source>(Source.web);
  const nodes = useMemo(() => {
    const res: Node[] = [];
    if (source == Source.wopi) {
      res.push(Node.wopi);
    } else {
      res.push(Node.user);
    }
    if (variant == "upload") {
      if (proxyed || source == Source.dav || source == Source.web_edit) {
        res.push(Node.cloudreve);
      }
    } else {
      if (proxyed || source == Source.wopi || source == Source.encrypted_file) {
        res.push(Node.cloudreve);
      }

      if (cdn) {
        res.push(Node.proxy);
      }
    }

    if (variant == "upload" && internalEndpoint && (source == Source.dav || source == Source.web_edit || proxyed)) {
      res.push(Node.storage_node_internal);
    } else if (
      variant == "download" &&
      internalEndpoint &&
      (source == Source.wopi || proxyed || source == Source.encrypted_file) &&
      !cdn
    ) {
      res.push(Node.storage_node_internal);
    } else {
      res.push(Node.storage_node);
    }

    // join arrow between existing nodes
    const joinedNodes: Node[] = [];
    for (const node of res) {
      joinedNodes.push(node);
      joinedNodes.push(Node.arrow);
    }
    return joinedNodes.slice(0, -1);
  }, [proxyed, source, cdn, variant, internalEndpoint]);
  return (
    <BorderedCard sx={{ overflow: "auto" }}>
      <DenseSelect variant="filled" value={source} onChange={(e) => setSource(e.target.value as Source)}>
        <SquareMenuItem value={Source.web}>
          <ListItemText
            slotProps={{
              primary: { variant: "body2" },
            }}
          >
            {t("policy.sourceWeb")}
          </ListItemText>
        </SquareMenuItem>
        <SquareMenuItem value={Source.dav}>
          <ListItemText
            slotProps={{
              primary: { variant: "body2" },
            }}
          >
            {t("policy.sourceDav")}
          </ListItemText>
        </SquareMenuItem>
        {variant == "upload" && (
          <SquareMenuItem value={Source.web_edit}>
            <ListItemText
              slotProps={{
                primary: { variant: "body2" },
              }}
            >
              {t("policy.sourceWebEdit")}
            </ListItemText>
          </SquareMenuItem>
        )}
        {variant == "download" && (
          <SquareMenuItem value={Source.wopi}>
            <ListItemText
              slotProps={{
                primary: { variant: "body2" },
              }}
            >
              {t("settings.wopiViewer")}
            </ListItemText>
          </SquareMenuItem>
        )}
        {variant == "download" && (
          <SquareMenuItem value={Source.encrypted_file}>
            <ListItemText
              slotProps={{
                primary: { variant: "body2" },
              }}
            >
              {t("policy.encryptedFile")}
            </ListItemText>
          </SquareMenuItem>
        )}
      </DenseSelect>
      <Box
        sx={{
          mt: 2,
          display: "flex",
          justifyContent: "safe center",
          alignItems: "center",
          flexDirection: isMobile ? "column" : "row",
          gap: 2,
        }}
      >
        {nodes.map((node) => (
          <NodeIcon variant={variant} type={node} storageNodeTitle={storageNodeTitle} proxyNodeTitle={proxyNodeTitle} />
        ))}
      </Box>
    </BorderedCard>
  );
};
