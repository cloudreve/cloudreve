import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import Backend from "i18next-http-backend";
import LanguageDetector from "i18next-browser-languagedetector";

i18n
  .use(Backend)
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    fallbackLng: "en-US",
    debug: import.meta.env.DEV,
    ns: ["common"],
    load: "currentOnly",
    defaultNS: "common",
    interpolation: {
      escapeValue: false,
    },
    backend: {
      loadPath: "/locales/{{lng}}/{{ns}}.json",
    },
    supportedLngs: ['en-US', 'zh-CN', 'zh-TW', 'ja', 'de', 'fr', 'es', 'ko', 'ru', 'pl', 'it'],
    detection: {
      // Order of detection: URL query param first, then other methods
      order: ['querystring', 'navigator', 'htmlTag'],
      // Look for 'lng' in query string
      lookupQuerystring: 'lng',
      // Cache the detected language
      caches: [],
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
    code: "ja",
    displayName: "日本語",
  },
  {
    code: "de",
    displayName: "Deutsch",
  },
  {
    code: "fr",
    displayName: "Français",
  },
  {
    code: "es",
    displayName: "Español",
  },
  {
    code: "ko",
    displayName: "한국어",
  },
  {
    code: "ru",
    displayName: "Русский",
  },
  {
    code: "pl",
    displayName: "Polski",
  },
  {
    code: "it",
    displayName: "Italiano",
  },
];

export default i18n;
