import { Box, Button, Collapse, DialogActions, DialogContent } from "@mui/material";
import * as React from "react";
import { createContext, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { CSSTransition, SwitchTransition } from "react-transition-group";
import { getUserDetail, upsertUser } from "../../../../api/api.ts";
import { UpsertUserService, User } from "../../../../api/dashboard.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import AutoHeight from "../../../Common/AutoHeight.tsx";
import FacebookCircularProgress from "../../../Common/CircularProgress.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import UserForm from "./UserForm.tsx";

export interface UserDialogProps {
  open: boolean;
  onClose: () => void;
  userID?: number;
  onUpdated?: (user: User) => void;
}

export interface UserDialogContextProps {
  values: User;
  setUser: (f: (p: User) => User) => void;
  formRef?: React.RefObject<HTMLFormElement>;
}

const defaultUser: User = {
  id: 0,
  nick: "",
  email: "",
  edges: {},
};

export const UserDialogContext = createContext<UserDialogContextProps>({
  values: { ...defaultUser },
  setUser: () => {},
});

const UserDialog = ({ open, onClose, userID, onUpdated }: UserDialogProps) => {
  const dispatch = useAppDispatch();
  const { t } = useTranslation("dashboard");
  const [values, setValues] = useState<User>({
    ...defaultUser,
  });
  const [modifiedValues, setModifiedValues] = useState<User>({
    ...defaultUser,
  });
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);

  const showSaveButton = useMemo(() => {
    return JSON.stringify(modifiedValues) !== JSON.stringify(values);
  }, [modifiedValues, values]);

  const loadUser = useCallback(() => {
    if (!userID) {
      return;
    }
    setLoading(true);
    dispatch(getUserDetail(userID))
      .then((res) => {
        setValues(res);
        setModifiedValues(res);
      })
      .catch(() => {
        onClose();
      })
      .finally(() => {
        setLoading(false);
      });
  }, [userID]);

  useEffect(() => {
    loadUser();
  }, [userID]);

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

    const args: UpsertUserService = {
      user: { ...modifiedValues },
    };

    if (!args.user.two_fa_enabled) {
      args.two_fa = "clear";
    }

    if (args.user.password) {
      args.password = args.user.password;
    }

    setSubmitting(true);
    dispatch(upsertUser(args))
      .then((res) => {
        setValues(res);
        setModifiedValues(res);
        onUpdated?.(res);
      })
      .finally(() => {
        setSubmitting(false);
      });
  };

  return (
    <UserDialogContext.Provider
      value={{
        values: modifiedValues,
        setUser: setModifiedValues,
        formRef,
      }}
    >
      <DraggableDialog
        title={t("user.userDialogTitle")}
        dialogProps={{
          fullWidth: true,
          maxWidth: "md",
          open: open,
          onClose: onClose,
        }}
      >
        <DialogContent>
          <AutoHeight>
            <SwitchTransition>
              <CSSTransition
                addEndListener={(node, done) => node.addEventListener("transitionend", done, false)}
                classNames="fade"
                key={`${loading}`}
              >
                <Box>
                  {loading && (
                    <Box
                      sx={{
                        py: 15,
                        height: "100%",
                        display: "flex",
                        justifyContent: "center",
                        alignItems: "center",
                      }}
                    >
                      <FacebookCircularProgress />
                    </Box>
                  )}
                  {!loading && <UserForm setLoading={setLoading} reload={loadUser} />}
                </Box>
              </CSSTransition>
            </SwitchTransition>
          </AutoHeight>
        </DialogContent>
        <Collapse in={showSaveButton}>
          <DialogActions>
            <Button disabled={submitting} onClick={revert}>
              {t("settings.revert")}
            </Button>
            <Button loading={submitting} variant="contained" onClick={submit}>
              {t("settings.save")}
            </Button>
          </DialogActions>
        </Collapse>
      </DraggableDialog>
    </UserDialogContext.Provider>
  );
};

export default UserDialog;
