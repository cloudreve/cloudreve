import { Box } from "@mui/material";
import * as React from "react";
import { createContext, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import { getOAuthClientDetail, upsertOAuthClient } from "../../../../api/api";
import { GetOAuthClientResponse, OAuthClient } from "../../../../api/dashboard";
import { useAppDispatch } from "../../../../redux/hooks";
import FacebookCircularProgress from "../../../Common/CircularProgress";
import { SavingFloat } from "../../Settings/SettingWrapper";

export interface OAuthClientSettingWrapperProps {
  clientID: number;
  children: React.ReactNode;
  onClientChange: (client: GetOAuthClientResponse) => void;
}

export interface OAuthClientSettingContextProps {
  values: GetOAuthClientResponse;
  setClient: (f: (c: GetOAuthClientResponse) => GetOAuthClientResponse) => void;
  formRef?: React.RefObject<HTMLFormElement>;
  isNew: boolean;
}

const defaultClient: GetOAuthClientResponse = {
  id: 0,
  name: "",
  redirect_uris: [],
  scopes: ["openid"],
  is_enabled: true,
  props: {
    refresh_token_ttl: 2592000, // 30 days default
  },
};

export const OAuthClientSettingContext = createContext<OAuthClientSettingContextProps>({
  values: { ...defaultClient },
  setClient: () => {},
  isNew: false,
});

const OAuthClientSettingWrapper = ({ clientID, children, onClientChange }: OAuthClientSettingWrapperProps) => {
  const dispatch = useAppDispatch();
  const navigate = useNavigate();
  const [values, setValues] = useState<GetOAuthClientResponse>({
    ...defaultClient,
  });
  const [modifiedValues, setModifiedValues] = useState<GetOAuthClientResponse>({
    ...defaultClient,
  });
  const [loading, setLoading] = useState(clientID !== 0);
  const [submitting, setSubmitting] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);
  const isNew = clientID === 0;

  const showSaveButton = useMemo(() => {
    return JSON.stringify(modifiedValues) !== JSON.stringify(values);
  }, [modifiedValues, values]);

  useEffect(() => {
    if (clientID === 0) {
      // New client, use defaults
      setValues({ ...defaultClient });
      setModifiedValues({ ...defaultClient });
      onClientChange({ ...defaultClient });
      return;
    }

    setLoading(true);
    dispatch(getOAuthClientDetail(clientID))
      .then((res) => {
        setValues(res);
        setModifiedValues(res);
        onClientChange(res);
      })
      .finally(() => {
        setLoading(false);
      });
  }, [clientID]);

  const revert = useCallback(() => {
    setModifiedValues(values);
  }, [values]);

  const submit = useCallback(() => {
    if (formRef.current) {
      if (!formRef.current.checkValidity()) {
        formRef.current.reportValidity();
        return;
      }
    }

    setSubmitting(true);
    const clientToSubmit: OAuthClient = {
      id: modifiedValues.id,
      guid: modifiedValues.guid,
      secret: modifiedValues.secret,
      name: modifiedValues.name,
      homepage_url: modifiedValues.homepage_url,
      redirect_uris: modifiedValues.redirect_uris,
      scopes: modifiedValues.scopes,
      props: modifiedValues.props,
      is_enabled: modifiedValues.is_enabled,
    };

    dispatch(upsertOAuthClient({ client: clientToSubmit }))
      .then((res) => {
        setValues(res);
        setModifiedValues(res);
        onClientChange(res);
        if (isNew) {
          // Navigate to the edit page after creating
          navigate(`/admin/oauth/${res.id}`, { replace: true });
        }
      })
      .finally(() => {
        setSubmitting(false);
      });
  }, [dispatch, modifiedValues, isNew, navigate, onClientChange]);

  return (
    <OAuthClientSettingContext.Provider
      value={{
        values: modifiedValues,
        setClient: setModifiedValues,
        formRef,
        isNew,
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
    </OAuthClientSettingContext.Provider>
  );
};

export default OAuthClientSettingWrapper;
