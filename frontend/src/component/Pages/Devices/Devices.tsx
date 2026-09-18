import PageHeader, { PageTabQuery } from "../PageHeader.tsx";
import { Button, Container, Grow } from "@mui/material";
import * as React from "react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSearchParams } from "react-router-dom";
import ResponsiveTabs from "../../Common/ResponsiveTabs.tsx";
import DavAccountList from "./DavAccountList.tsx";
import Add from "../../Icons/Add.tsx";
import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { loadSiteConfig } from "../../../redux/thunks/site.ts";
import Nothing from "../../Common/Nothing.tsx";
import SessionManager from "../../../session";
import Boolset from "../../../util/boolset.ts";
import { GroupPermission } from "../../../api/user.ts";
import AppPromotion from "./AppPromotion.tsx";
import DesktopAppPromotion from "./DesktopAppPromotion.tsx";
import PageContainer from "../PageContainer.tsx";

export enum DevicePageTab {
  Dav = "dav",
  App = "app",
  DesktopApp = "desktop",
}

const Devices = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const [searchParams, setSearchParams] = useSearchParams();
  const [creatAccountDialog, setCreateAccountDialog] = useState(false);
  const appPromotion = useAppSelector((state) => state.siteConfig.app.config?.app_promotion);
  const desktopAppPromotion = useAppSelector((state) => state.siteConfig.app.config?.desktop_app_promotion);

  const webDavEnabled = useMemo(() => {
    const user = SessionManager.currentLoginOrNull();
    let enabled = false;
    if (user && user.user.group?.permission) {
      const bs = new Boolset(user.user.group.permission);
      enabled = bs.enabled(GroupPermission.webdav);
    }

    return enabled;
  }, []);

  const tabs = useMemo(() => {
    const res = [];
    if (webDavEnabled) {
      res.push({
        label: t("application:setting.webdavAccounts"),
        value: DevicePageTab.Dav,
      });
    }
    if (appPromotion) {
      res.push({
        label: t("application:setting.iOSApp"),
        value: DevicePageTab.App,
      });
    }
    if (desktopAppPromotion) {
      res.push({
        label: t("application:setting.desktopApp"),
        value: DevicePageTab.DesktopApp,
      });
    }
    return res;
  }, [webDavEnabled, appPromotion, desktopAppPromotion]);

  const [tab, setTab] = useState(
    searchParams.get(PageTabQuery) ?? (webDavEnabled ? DevicePageTab.Dav : DevicePageTab.App),
  );

  useEffect(() => {
    dispatch(loadSiteConfig("app"));
  }, []);

  return (
    <PageContainer>
      <Container maxWidth="lg">
        <PageHeader
          secondaryAction={
            <Grow in={tab == DevicePageTab.Dav}>
              <Button variant={"contained"} startIcon={<Add />} onClick={() => setCreateAccountDialog(true)}>
                {t("setting.createNewAccount")}
              </Button>
            </Grow>
          }
          title={t("application:navbar.connect")}
        />
        <ResponsiveTabs value={tab} onChange={(_e, newValue) => setTab(newValue)} tabs={tabs} />
        {tab == DevicePageTab.Dav && webDavEnabled && (
          <DavAccountList creatAccountDialog={creatAccountDialog} setCreateAccountDialog={setCreateAccountDialog} />
        )}
        {tab == DevicePageTab.App && appPromotion && <AppPromotion />}
        {tab == DevicePageTab.DesktopApp && desktopAppPromotion && <DesktopAppPromotion />}

        {!webDavEnabled && !appPromotion && !desktopAppPromotion && <Nothing primary={t("setting.deviceNothing")} />}
      </Container>
    </PageContainer>
  );
};

export default Devices;
