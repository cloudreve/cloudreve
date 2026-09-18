import * as React from "react";
import { useCallback, useState } from "react";
import { Box, Table, TableBody, TableContainer, TableHead, TableRow, Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import { getDavAccounts } from "../../../api/api.ts";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { DavAccount } from "../../../api/setting.ts";
import DavAccountRow from "./DavAccountRow.tsx";
import { NoWrapTableCell } from "../../Common/StyledComponents.tsx";
import { CellHeaderWithPadding } from "../../FileManager/Dialogs/LockConflictDetails.tsx";
import CreateDAVAccountDialog from "./CreateDAVAccountDialog.tsx";
import ConnectionInfoDialog from "./ConnectionInfoDialog.tsx";

const defaultPageSize = 50;

export interface DavAccountListProps {
  creatAccountDialog: boolean;
  setCreateAccountDialog: (value: boolean) => void;
}

const DavAccountList = ({ creatAccountDialog, setCreateAccountDialog }: DavAccountListProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const [nextPageToken, setNextPageToken] = useState<string | undefined>("");
  const [accounts, setAccounts] = useState<DavAccount[]>([]);
  const [loading, setLoading] = useState(false);
  const [accountInfoTarget, setAccountInfoTarget] = useState<DavAccount | undefined>();
  const [editTarget, setEditTarget] = useState<DavAccount | undefined>();
  const [editOpen, setEditOpen] = useState(false);

  const loadNextPage = useCallback(
    (originAccounts: DavAccount[], token?: string) => () => {
      setLoading(true);
      dispatch(
        getDavAccounts({
          page_size: defaultPageSize,
          next_page_token: token,
        }),
      )
        .then((res) => {
          setAccounts([...originAccounts, ...res.accounts]);
          if (res.pagination?.next_token) {
            setNextPageToken(res.pagination.next_token);
          } else {
            setNextPageToken(undefined);
          }
        })
        .catch(() => {
          setNextPageToken(undefined);
        })
        .finally(() => {
          setLoading(false);
        });
    },
    [dispatch],
  );

  const refresh = () => {
    loadNextPage([], "")();
  };

  const onAccountDeleted = useCallback(
    (id: string) => {
      setAccounts((accounts) => accounts.filter((account) => account.id !== id));
    },
    [setAccounts],
  );

  const onAccountAdded = (account: DavAccount) => {
    setAccounts([account, ...accounts]);
    setAccountInfoTarget(account);
  };

  const onEditOpen = (account: DavAccount) => {
    setEditOpen(true);
    setEditTarget(account);
  };

  return (
    <Box>
      <CreateDAVAccountDialog
        open={creatAccountDialog}
        onClose={() => setCreateAccountDialog(false)}
        onAccountAdded={onAccountAdded}
      />
      <CreateDAVAccountDialog
        open={editOpen}
        editTarget={editTarget}
        onClose={() => setEditOpen(false)}
        onAccountUpdated={(account) => setAccounts(accounts.map((a) => (a.id == account.id ? account : a)))}
      />
      <ConnectionInfoDialog
        open={accountInfoTarget != undefined}
        account={accountInfoTarget}
        onClose={() => setAccountInfoTarget(undefined)}
      />
      <TableContainer sx={{ mt: 1 }}>
        <Table sx={{ width: "100%", tableLayout: "fixed" }} size="small">
          <TableHead>
            <TableRow>
              <NoWrapTableCell width={200}>{t("setting.annotation")}</NoWrapTableCell>
              <NoWrapTableCell width={200}>
                <CellHeaderWithPadding>{t("setting.rootFolder")}</CellHeaderWithPadding>
              </NoWrapTableCell>
              <NoWrapTableCell width={150}>{t("fileManager.permissions")}</NoWrapTableCell>
              <NoWrapTableCell width={150}>{t("setting.proxy")}</NoWrapTableCell>
              <NoWrapTableCell width={200}>{t("fileManager.createdAt")}</NoWrapTableCell>
              <NoWrapTableCell width={100}>{t("fileManager.actions")}</NoWrapTableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {accounts.map((account) => (
              <DavAccountRow
                onClick={() => setAccountInfoTarget(account)}
                key={account.id}
                account={account}
                onAccountDeleted={onAccountDeleted}
                onEditClicked={() => onEditOpen(account)}
              />
            ))}
            {nextPageToken != undefined && (
              <>
                {[...Array(4)].map((_, i) => (
                  <DavAccountRow
                    onLoad={i == 0 ? loadNextPage(accounts, nextPageToken) : undefined}
                    loading={true}
                    key={i}
                    onAccountDeleted={onAccountDeleted}
                  />
                ))}
              </>
            )}
          </TableBody>
        </Table>
        {nextPageToken == undefined && accounts.length == 0 && (
          <Box sx={{ p: 1, width: "100%", textAlign: "center" }}>
            <Typography variant={"caption"} color={"text.secondary"}>
              {t("application:setting.listEmpty")}
            </Typography>
          </Box>
        )}
      </TableContainer>
    </Box>
  );
};

export default DavAccountList;
