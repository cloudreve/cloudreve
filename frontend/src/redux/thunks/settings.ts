import { AppThunk } from "../store.ts";
import { sendPinFile, sendUnpinFile, sendUpdateUserSetting } from "../../api/api.ts";
import { updateSiteConfig } from "./site.ts";
import { increasePinedGeneration } from "../globalStateSlice.ts";
import i18next from "i18next";

export function pinToSidebar(uri: string, name?: string): AppThunk<Promise<void>> {
  return async (dispatch, _getState) => {
    await dispatch(
      sendPinFile({
        uri,
        name,
      }),
    );
    await dispatch(updateSiteConfig());
    dispatch(increasePinedGeneration());
  };
}

export function unPinFromSidebar(uri: string): AppThunk<Promise<void>> {
  return async (dispatch, _getState) => {
    await dispatch(
      sendUnpinFile({
        uri,
      }),
    );
    await dispatch(updateSiteConfig());
    dispatch(increasePinedGeneration());
  };
}

export function selectLanguage(lng: string): AppThunk<Promise<void>> {
  return async (dispatch, _getState) => {
    await i18next.changeLanguage(lng);
    await dispatch(
      sendUpdateUserSetting({
        language: lng,
      }),
    );
  };
}
