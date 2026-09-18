import * as React from "react";
import { useEffect, useMemo } from "react";
import { IconButton, ListItemText, Menu, Skeleton, TableRow } from "@mui/material";
import { useTranslation } from "react-i18next";
import { useAppDispatch } from "../../../redux/hooks.ts";
import { NoWrapCell, SquareChip } from "../../Common/StyledComponents.tsx";
import { DavAccount, DavAccountOption } from "../../../api/setting.ts";
import FileBadge from "../../FileManager/FileBadge.tsx";
import { FileType } from "../../../api/explorer.ts";
import TimeBadge from "../../Common/TimeBadge.tsx";
import Boolset from "../../../util/boolset.ts";
import { defaultPath } from "../../../hooks/useNavigation.tsx";
import { useInView } from "react-intersection-observer";
import Eye from "../../Icons/Eye.tsx";
import { Edit } from "@mui/icons-material";
import CloudFilled from "../../Icons/CloudFilled.tsx";
import { bindMenu, bindTrigger, usePopupState } from "material-ui-popup-state/hooks";
import MoreVertical from "../../Icons/MoreVertical.tsx";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu.tsx";
import { sendDeleteDavAccount } from "../../../api/api.ts";

export interface DavAccountRowProps {
  account?: DavAccount;
  onLoad?: () => void;
  loading?: boolean;
  onAccountDeleted: (id: string) => void;
  onClick?: () => void;
  onEditClicked?: (account: DavAccount) => void;
}

const DavAccountRow = ({ account, onAccountDeleted, onLoad, loading, onClick, onEditClicked }: DavAccountRowProps) => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();

  const { ref, inView } = useInView({
    rootMargin: "200px 0px",
    triggerOnce: true,
    skip: !loading || !onLoad,
  });

  const actionMenuState = usePopupState({
    variant: "popover",
    popupId: "action" + account?.id,
  });

  useEffect(() => {
    if (!inView) {
      return;
    }

    if (onLoad) {
      onLoad();
    }
  }, [inView]);

  const [readOnly, proxy] = useMemo(() => {
    if (!account?.options) return [false, false] as const;
    const bs = new Boolset(account.options);
    return [bs.enabled(DavAccountOption.readonly), bs.enabled(DavAccountOption.proxy)] as const;
  }, [account?.options]);

  const { onClose, ...rest } = bindMenu(actionMenuState);

  const onEdit = () => {
    if (account) {
      onEditClicked?.(account);
    }
    onClose && onClose();
  };

  const onDelete = () => {
    if (!account?.id) {
      return;
    }

    onClose();
    dispatch(sendDeleteDavAccount(account.id));
    onAccountDeleted(account.id);
  };

  return (
    <TableRow ref={ref} hover sx={{ cursor: "pointer" }} onClick={onClick}>
      <NoWrapCell component="th" scope="row">
        {loading ? <Skeleton variant={"text"} width={200} sx={{ my: "6px" }} /> : account?.name}
      </NoWrapCell>
      <NoWrapCell>
        {loading ? (
          <Skeleton variant={"text"} width={100} />
        ) : (
          <FileBadge
            variant={"outlined"}
            clickable
            sx={{ px: 1 }}
            simplifiedFile={{
              path: account?.uri ?? defaultPath,
              type: FileType.folder,
            }}
          />
        )}
      </NoWrapCell>
      <NoWrapCell>
        {loading ? (
          <Skeleton variant={"text"} width={100} />
        ) : readOnly ? (
          <SquareChip icon={<Eye />} size={"small"} label={t("setting.readonlyOn")} />
        ) : (
          <SquareChip icon={<Edit />} size={"small"} label={t("setting.readonlyOff")} />
        )}
      </NoWrapCell>
      <NoWrapCell>
        {loading ? (
          <Skeleton variant={"text"} width={100} />
        ) : proxy ? (
          <SquareChip
            icon={<CloudFilled />}
            sx={{
              backgroundColor: (theme) => theme.palette.primary.light,
            }}
            color={"success"}
            size={"small"}
            label={t("application:setting.proxied")}
          />
        ) : (
          t("setting.none")
        )}
      </NoWrapCell>
      <NoWrapCell>
        {loading || !account ? (
          <Skeleton variant={"text"} width={100} />
        ) : (
          <TimeBadge datetime={account.created_at} variant={"inherit"} />
        )}
      </NoWrapCell>
      <NoWrapCell onClick={(e) => e.stopPropagation()}>
        {account && (
          <IconButton size={"small"} {...bindTrigger(actionMenuState)}>
            <MoreVertical fontSize={"small"} />
          </IconButton>
        )}
      </NoWrapCell>
      <Menu
        onClick={(e) => e.stopPropagation()}
        onClose={onClose}
        slotProps={{
          paper: {
            sx: {
              minWidth: 150,
            },
          },
        }}
        {...rest}
      >
        <SquareMenuItem dense onClick={onEdit}>
          <ListItemText>{t(`fileManager.edit`)}</ListItemText>
        </SquareMenuItem>
        <SquareMenuItem dense onClick={onDelete}>
          <ListItemText>{t(`fileManager.delete`)}</ListItemText>
        </SquareMenuItem>
      </Menu>
    </TableRow>
  );
};

export default DavAccountRow;
