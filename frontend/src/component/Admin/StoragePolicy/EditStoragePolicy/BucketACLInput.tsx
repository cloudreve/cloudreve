import { Box, ListItemText, OutlinedSelectProps, Typography } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { PolicyType } from "../../../../api/explorer";
import { DenseSelect } from "../../../Common/StyledComponents";
import { SquareMenuItem } from "../../../FileManager/ContextMenu/ContextMenu";

export interface BucketACLInputProps extends OutlinedSelectProps {
  value?: boolean;
  phraseVariant?: PolicyType;
  onChange: (value: boolean) => void;
}

const BucketACLInput = ({ value, phraseVariant = PolicyType.oss, onChange, ...props }: BucketACLInputProps) => {
  const { t } = useTranslation("dashboard");
  const { privateLocale, publicLocale } = useMemo(() => {
    switch (phraseVariant) {
      case PolicyType.oss:
      case PolicyType.obs:
        return { privateLocale: "policy.privateBucket", publicLocale: "policy.publicBucket" };
      case PolicyType.cos:
        return { privateLocale: "policy.accessTypePrivate", publicLocale: "policy.accessTypePulic" };
      case PolicyType.upyun:
        return { privateLocale: "policy.tokenEnabled", publicLocale: "policy.tokenDisabled" };
      default:
        return { privateLocale: "policy.privateBucket", publicLocale: "policy.publicBucket" };
    }
  }, [phraseVariant]);
  return (
    <DenseSelect
      value={value ? "1" : "0"}
      renderValue={(value) => (
        <ListItemText
          slotProps={{
            primary: { variant: "body2" },
          }}
        >
          {value === "1" ? t(privateLocale) : t(publicLocale)}
        </ListItemText>
      )}
      onChange={(e) => onChange(e.target.value === "1")}
      {...props}
      MenuProps={{
        PaperProps: { sx: { maxWidth: 230 } },
        MenuListProps: {
          sx: {
            "& .MuiMenuItem-root": {
              whiteSpace: "normal",
            },
          },
        },
      }}
    >
      <SquareMenuItem value={"1"}>
        <Box
          sx={{
            display: "flex",
            flexDirection: "column",
          }}
        >
          <Typography variant={"body2"} fontWeight={600}>
            {t(privateLocale)}
          </Typography>
          <Typography variant={"caption"} color={"textSecondary"}>
            {t("policy.privateDes")}
          </Typography>
        </Box>
      </SquareMenuItem>
      <SquareMenuItem value={"0"}>
        <Box
          sx={{
            display: "flex",
            flexDirection: "column",
          }}
        >
          <Typography variant={"body2"} fontWeight={600}>
            {t(publicLocale)}
          </Typography>
          <Typography variant={"caption"} color={"textSecondary"}>
            {t("policy.publicDes")}
          </Typography>
        </Box>
      </SquareMenuItem>
    </DenseSelect>
  );
};

export default BucketACLInput;
