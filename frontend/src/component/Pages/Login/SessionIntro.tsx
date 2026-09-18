import { useAppDispatch, useAppSelector } from "../../../redux/hooks.ts";
import { useEffect } from "react";
import { ConfigLoadState } from "../../../redux/siteConfigSlice.ts";
import { setHeadlessFrameLoading } from "../../../redux/globalStateSlice.ts";
import { loadSiteConfig } from "../../../redux/thunks/site.ts";
import { Outlet } from "react-router-dom";

const SessionIntro = () => {
  const dispatch = useAppDispatch();
  const loginConfigLoading = useAppSelector((state) => state.siteConfig.login.loaded);
  useEffect(() => {
    dispatch(loadSiteConfig("login"));
  }, []);
  useEffect(() => {
    if (loginConfigLoading == ConfigLoadState.NotLoaded) {
      dispatch(setHeadlessFrameLoading(true));
    } else {
      dispatch(setHeadlessFrameLoading(false));
    }
  }, [loginConfigLoading]);
  return <>{loginConfigLoading != ConfigLoadState.NotLoaded && <Outlet />}</>;
};

export default SessionIntro;
