import { Box, FormControlLabel } from "@mui/material";
import { useTranslation } from "react-i18next";
import { isTrueVal } from "../../../../session/utils.ts";
import { StyledCheckbox } from "../../../Common/StyledComponents.tsx";
import { PropsContentProps } from "./CustomPropsItem.tsx";

const BooleanPropsItem = ({ prop, onChange, loading, readOnly, fullSize }: PropsContentProps) => {
  const { t } = useTranslation();

  const handleChange = (_: any, checked: boolean) => {
    onChange(checked.toString());
  };

  return (
    <Box sx={{ pl: "10px" }}>
      <FormControlLabel
        slotProps={{
          typography: {
            variant: "inherit",
            pl: 1,
          },
        }}
        control={<StyledCheckbox size={"small"} checked={isTrueVal(prop.value)} onChange={handleChange} />}
        label={fullSize ? t(prop.props.name) : undefined}
        disabled={readOnly || loading}
      />
    </Box>
  );
};

export default BooleanPropsItem;
