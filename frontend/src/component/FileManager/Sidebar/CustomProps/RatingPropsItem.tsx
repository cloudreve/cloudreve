import { Rating } from "@mui/material";
import { useTranslation } from "react-i18next";
import { PropsContentProps } from "./CustomPropsItem.tsx";

const RatingPropsItem = ({ prop, onChange, loading, readOnly }: PropsContentProps) => {
  const { t } = useTranslation();

  const handleChange = (_: any, value: number | null) => {
    onChange(value?.toString() ?? "");
  };

  return (
    <Rating
      readOnly={readOnly}
      disabled={loading}
      onChange={handleChange}
      value={parseInt(prop.value) ?? 0}
      max={prop.props.max ?? 5}
    />
  );
};

export default RatingPropsItem;
