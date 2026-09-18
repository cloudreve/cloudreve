import { Box } from "@mui/material";
import * as React from "react";
import { createContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import { getNodeDetail, upsertNode } from "../../../../api/api.ts";
import { Node, StoragePolicy } from "../../../../api/dashboard.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import FacebookCircularProgress from "../../../Common/CircularProgress.tsx";
import { SavingFloat } from "../../Settings/SettingWrapper.tsx";

export interface NodeSettingWrapperProps {
  nodeID: number;
  children: React.ReactNode;
  onNodeChange: (node: Node) => void;
}

export interface NodeSettingContextProps {
  values: Node;
  setNode: (f: (p: Node) => Node) => void;
  formRef?: React.RefObject<HTMLFormElement>;
}

const defaultNode: Node = {
  id: 0,
  name: "",
  status: undefined,
  type: undefined,
  server: "",
  slave_key: "",
  capabilities: "",
  weight: 1,
  settings: {},
  edges: {
    storage_policy: [],
  },
};

export const NodeSettingContext = createContext<NodeSettingContextProps>({
  values: { ...defaultNode },
  setNode: () => {},
});

const nodeValueFilter = (node: Node): Node => {
  return {
    ...node,
    edges: {
      storage_policy: node.edges.storage_policy?.map(
        (p): StoragePolicy =>
          ({
            id: p.id,
          }) as StoragePolicy,
      ),
    },
  };
};

const NodeSettingWrapper = ({ nodeID, children, onNodeChange }: NodeSettingWrapperProps) => {
  const dispatch = useAppDispatch();
  const { t } = useTranslation("dashboard");
  const [values, setValues] = useState<Node>({
    ...defaultNode,
  });
  const [modifiedValues, setModifiedValues] = useState<Node>({
    ...defaultNode,
  });
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);

  const showSaveButton = useMemo(() => {
    return JSON.stringify(modifiedValues) !== JSON.stringify(values);
  }, [modifiedValues, values]);

  useEffect(() => {
    setLoading(true);
    dispatch(getNodeDetail(nodeID))
      .then((res) => {
        setValues(nodeValueFilter(res));
        setModifiedValues(nodeValueFilter(res));
        onNodeChange(nodeValueFilter(res));
      })
      .finally(() => {
        setLoading(false);
      });
  }, [nodeID]);

  const revert = () => {
    setModifiedValues(values);
  };

  const submit = () => {
    if (formRef.current) {
      if (!formRef.current.checkValidity()) {
        formRef.current.reportValidity();
        return;
      }
    }

    setSubmitting(true);
    dispatch(
      upsertNode({
        node: { ...modifiedValues },
      }),
    )
      .then((res) => {
        setValues(nodeValueFilter(res));
        setModifiedValues(nodeValueFilter(res));
        onNodeChange(nodeValueFilter(res));
      })
      .finally(() => {
        setSubmitting(false);
      });
  };

  return (
    <NodeSettingContext.Provider
      value={{
        values: modifiedValues,
        setNode: setModifiedValues,
        formRef,
      }}
    >
      <SwitchTransition>
        <CSSTransition
          addEndListener={(node, done) => node.addEventListener("transitionend", done, false)}
          classNames="fade"
          key={`${loading}`}
        >
          <Box sx={{ mt: 3 }}>
            {loading && (
              <Box
                sx={{
                  pt: 20,
                  height: "100%",
                  display: "flex",
                  justifyContent: "center",
                  alignItems: "center",
                }}
              >
                <FacebookCircularProgress />
              </Box>
            )}
            {!loading && (
              <Box>
                {children}
                <SavingFloat in={showSaveButton} submitting={submitting} revert={revert} submit={submit} />
              </Box>
            )}
          </Box>
        </CSSTransition>
      </SwitchTransition>
    </NodeSettingContext.Provider>
  );
};

export default NodeSettingWrapper;
