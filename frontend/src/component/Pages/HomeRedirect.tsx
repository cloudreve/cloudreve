import SessionManager from "../../session";
import { useNavigate } from "react-router-dom";
import { useEffect } from "react";

export const HomeRedirect = () => {
  const navigate = useNavigate();
  useEffect(() => {
    if (SessionManager.currentLoginOrNull()) {
      navigate("/home");
    } else {
      navigate("/session");
    }
  }, []);

  return <div></div>;
};
