import { Box } from "@mui/material";
import { useEffect } from "react";
import { setHeadlessFrameLoading } from "../../../../redux/globalStateSlice.ts";
import { useAppDispatch } from "../../../../redux/hooks.ts";
import SessionManager from "../../../../session/index.ts";
import { useQuery } from "../../../../util/index.ts";

const MobileCallback = () => {
  const dispatch = useAppDispatch();
  const query = useQuery();

  useEffect(() => {
    dispatch(setHeadlessFrameLoading(true));
    const code = query.get("code");
    const state = query.get("state");
    const params: Record<string, string> = {
      state: state ?? "",
      code: code ?? "",
      user_id: SessionManager.currentLoginOrNull()?.user.id ?? "",
    };
    const search = new URLSearchParams(params);
    window.location.href = `cloudreve://mount?${search.toString()}`;
  }, []);

  return (
    <Box
      sx={{
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        pt: 7,
        pb: 9,
      }}
    ></Box>
  );
};

export default MobileCallback;
