import i18n from "i18next";
import { initReactI18next } from "react-i18next";

import Backend from "i18next-http-backend";
// eslint-disable-next-line @typescript-eslint/ban-ts-ignore
// @ts-ignore
import LanguageDetector from "i18next-browser-languagedetector";
import ChainedBackend from "i18next-chained-backend";
import LocalStorageBackend from "i18next-localstorage-backend";

// eslint-disable-next-line @typescript-eslint/ban-ts-ignore
// @ts-ignore
i18n
  .use(ChainedBackend)
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    fallbackLng: "en-US",
    debug: true,
    ns: ["common", "application", "dashboard"],
    load: "currentOnly",
    defaultNS: "application",
    interpolation: {
      escapeValue: false,
    },
    backend: {
      backends: process.env.NODE_ENV === "development" ? [Backend] : [LocalStorageBackend, Backend],
      backendOptions: [
        {
          expirationTime: 7 * 24 * 60 * 60 * 1000, // 7 days
          loadPath: "/locales/{{lng}}/{{ns}}.json",
        },
      ],
    },
  });

i18n.on("languageChanged", (lng) => {
  document.documentElement.setAttribute("lang", lng);
});

export const languages = [
  {
    code: "en-US",
    displayName: "English",
  },
  {
    code: "zh-CN",
    displayName: "简体中文",
  },
  {
    code: "zh-TW",
    displayName: "繁體中文",
  },
  {
    code: "ja-JP",
    displayName: "日本語",
  },
  {
    code: "ru-RU",
    displayName: "Русский",
  },
  {
    code: "de-DE",
    displayName: "Deutsch",
  },
  {
    code: "fr-FR",
    displayName: "Français",
  },
  {
    code: "es-ES",
    displayName: "Español",
  },
  {
    code: "pt-BR",
    displayName: "Português",
  },
  {
    code: "it-IT",
    displayName: "Italiano",
  },
  {
    code: "ko-KR",
    displayName: "한국어",
  },
  {
    code: "pl-PL",
    displayName: "Polski",
  },
  {
    code: "ar-AR",
    displayName: "العربية",
  },
];

export default i18n;
