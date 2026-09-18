import { useTranslation } from "react-i18next";
import { useRouteError } from "react-router-dom";

function ErrorBoundary() {
  let error = useRouteError();
  const { t } = useTranslation();
  const loader = document.getElementById("app-loader");
  if (loader) loader.style.display = "none";
  // Uncaught ReferenceError: path is not defined
  return (
    <div style={{ padding: 16 }}>
      <h1 style={{ color: "#a4a4a4", margin: "5px 0px" }}>:(</h1>
      <h2 style={{ margin: "15px 0px" }}>{t("common:renderError")}</h2>
      {!!error && (
        <details>
          <summary>{t("common:errorDetails")}</summary>
          <pre>
            <code>{error.toString()}</code>
          </pre>
          {error.stack && (
            <pre>
              <code>{error.stack}</code>
            </pre>
          )}
        </details>
      )}
    </div>
  );
}

export default ErrorBoundary;
