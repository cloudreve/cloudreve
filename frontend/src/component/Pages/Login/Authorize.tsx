import { useQuery } from "../../../util";
import SignIn, { OAuthConsentProps } from "./Signin/SignIn.tsx";

const Authorize = () => {
  const query = useQuery();

  const oauthConsent: OAuthConsentProps = {
    clientId: query.get("client_id") || "",
    responseType: query.get("response_type") || "code",
    redirectUri: query.get("redirect_uri") || "",
    state: query.get("state") || "",
    scope: query.get("scope") || "",
    codeChallenge: query.get("code_challenge") || undefined,
    codeChallengeMethod: query.get("code_challenge_method") || undefined,
  };

  return <SignIn oauthConsent={oauthConsent} />;
};

export default Authorize;
