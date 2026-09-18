import { Box, IconButton, Link, Typography } from "@mui/material";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { NoLabelFilledTextField } from "../../../Common/StyledComponents.tsx";
import Edit from "../../../Icons/Edit.tsx";
import { PropsContentProps } from "./CustomPropsItem.tsx";

const LinkPropsContent = ({ prop, onChange, loading, fullSize, readOnly }: PropsContentProps) => {
  const { t } = useTranslation();
  const [value, setValue] = useState(prop.value);
  const [isEditing, setIsEditing] = useState(false);
  const [isHovered, setIsHovered] = useState(false);

  useEffect(() => {
    setValue(prop.value);
  }, [prop.value]);

  const handleEditClick = () => {
    setIsEditing(true);
  };

  const handleBlur = () => {
    setIsEditing(false);
    if (value !== prop.value) {
      onChange(value);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleBlur();
    }
    if (e.key === "Escape") {
      setValue(prop.value);
      setIsEditing(false);
    }
  };

  if (readOnly) {
    if (!value) {
      return null;
    }
    return (
      <Link href={value} target="_blank" rel="noopener noreferrer" variant="body2" sx={{ wordBreak: "break-all" }}>
        {value}
      </Link>
    );
  }

  if (isEditing) {
    return (
      <NoLabelFilledTextField
        variant="filled"
        fullSize={fullSize}
        placeholder={t("application:fileManager.enterUrl")}
        disabled={loading}
        fullWidth
        autoFocus
        onClick={(e) => e.stopPropagation()}
        onChange={(e) => setValue(e.target.value)}
        value={value ?? ""}
        onBlur={handleBlur}
        onKeyDown={handleKeyDown}
      />
    );
  }

  if (!value) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ cursor: "pointer" }} onClick={handleEditClick}>
        {t("application:fileManager.clickToEdit")}
      </Typography>
    );
  }

  return (
    <Box
      sx={{ position: "relative", width: "100%" }}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <Link
        href={value}
        target="_blank"
        rel="noopener noreferrer"
        variant="body2"
        sx={{ wordBreak: "break-all", pr: isHovered ? 4 : 0 }}
      >
        {value}
      </Link>
      {isHovered && (
        <IconButton
          size="small"
          sx={{
            position: "absolute",
            right: 0,
            top: "50%",
            transform: "translateY(-50%)",
            opacity: 0.7,
            ...(!fullSize
              ? {
                  padding: 0,
                }
              : {}),
            "&:hover": {
              opacity: 1,
            },
          }}
          onClick={(e) => {
            e.stopPropagation();
            handleEditClick();
          }}
        >
          <Edit fontSize="small" />
        </IconButton>
      )}
    </Box>
  );
};

export default LinkPropsContent;
