import { Icon } from "@iconify/react/dist/iconify.js";
import { Box, Grow, IconButton, Typography } from "@mui/material";
import { forwardRef, useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useAppSelector } from "../../../../redux/hooks.ts";
import CalendarClock from "../../../Icons/CalendarClock.tsx";
import Dismiss from "../../../Icons/Dismiss.tsx";
import FolderOutlined from "../../../Icons/FolderOutlined.tsx";
import HardDriveOutlined from "../../../Icons/HardDriveOutlined.tsx";
import Numbers from "../../../Icons/Numbers.tsx";
import Search from "../../../Icons/Search.tsx";
import Tag from "../../../Icons/Tag.tsx";
import TextCaseTitle from "../../../Icons/TextCaseTitle.tsx";
import { customPropsMetadataPrefix } from "../../Sidebar/CustomProps/CustomProps.tsx";
import { CustomPropsConditon } from "./CustomPropsConditon.tsx";
import { DateTimeCondition } from "./DateTimeCondition.tsx";
import { FileNameCondition, StyledBox } from "./FileNameCondition.tsx";
import { FileTypeCondition } from "./FileTypeCondition.tsx";
import { MetadataCondition } from "./MetadataCondition.tsx";
import { SearchBaseCondition } from "./SearchBaseCondition.tsx";
import { SizeCondition } from "./SizeCondition.tsx";
import { TagCondition } from "./TagCondition.tsx";

export interface Condition {
  type: ConditionType;
  case_folding?: boolean;
  names?: string[];
  name_op_or?: boolean;
  file_type?: number;
  size_gte?: number;
  size_lte?: number;
  time?: number;
  metadata_key?: string;
  metadata_value?: string;
  metadata_strong_match?: boolean;
  base_uri?: string;
  tags?: string[];
  id?: string;
  metadata_key_readonly?: boolean;
  created_gte?: number;
  created_lte?: number;
  updated_gte?: number;
  updated_lte?: number;
}

export enum ConditionType {
  name,
  size,
  created,
  modified,
  type,
  metadata,
  base,
  tag,
}

export interface ConditionProps {
  condition: Condition;
  onChange: (condition: Condition) => void;
  onRemove?: (condition: Condition) => void;
  index: number;
}

const ConditionBox = forwardRef((props: ConditionProps, ref) => {
  const { condition, index, onRemove, onChange } = props;
  const customPropsOptions = useAppSelector((state) => state.siteConfig.explorer?.config?.custom_props);
  const { t } = useTranslation();
  const [hovered, setHovered] = useState(false);

  const onNameConditionAdded = useCallback(
    (_e: any, newValue: string[]) => {
      onChange({
        ...condition,
        names: newValue,
      });
    },
    [onChange],
  );

  const customPropsOption = useMemo(() => {
    if (
      condition.type !== ConditionType.metadata ||
      !condition.metadata_key ||
      !condition.metadata_key.startsWith(customPropsMetadataPrefix)
    ) {
      return undefined;
    }
    return customPropsOptions?.find(
      (option) => option.id === condition?.metadata_key?.slice(customPropsMetadataPrefix.length),
    );
  }, [customPropsOptions, condition.type, condition.metadata_key]);

  const title = useMemo(() => {
    switch (condition.type) {
      case ConditionType.base:
        return t("application:navbar.searchBase");
      case ConditionType.name:
        return t("application:modals.fileName");
      case ConditionType.type:
        return t("application:navbar.fileType");
      case ConditionType.tag:
        return t("application:fileManager.tags");
      case ConditionType.metadata:
        if (customPropsOption) {
          return t(customPropsOption.name);
        }
        return t("application:fileManager.metadata");
      case ConditionType.size:
        return t("application:navbar.fileSize");
      case ConditionType.modified:
        return t("application:fileManager.updatedDate");
      case ConditionType.created:
        return t("application:fileManager.createDate");
      default:
        return "Unknown";
    }
  }, [t, condition, customPropsOption]);

  const ConditionIcon = useMemo(() => {
    switch (condition.type) {
      case ConditionType.base:
        return Search;
      case ConditionType.type:
        return FolderOutlined;
      case ConditionType.tag:
        return Tag;
      case ConditionType.metadata:
        if (customPropsOption?.icon) {
          return customPropsOption?.icon;
        }
        return Numbers;
      case ConditionType.size:
        return HardDriveOutlined;
      case ConditionType.modified:
      case ConditionType.created:
        return CalendarClock;

      default:
        return TextCaseTitle;
    }
  }, [condition.type, customPropsOption]);

  return (
    <StyledBox
      ref={ref}
      sx={{
        mt: index > 0 ? 1 : 0,
      }}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
    >
      <Typography
        sx={{
          gap: 1,
          mb: 1,
          display: "flex",
          alignItems: "center",
        }}
        variant={"body2"}
        fontWeight={600}
      >
        {typeof ConditionIcon !== "string" ? (
          <ConditionIcon sx={{ width: "20px", height: "20px" }} />
        ) : (
          <Icon icon={ConditionIcon} width={20} height={20} />
        )}
        <Box sx={{ flexGrow: 1 }}>{title}</Box>
        <Grow in={hovered && !!onRemove}>
          <IconButton onClick={onRemove ? () => onRemove(condition) : undefined}>
            <Dismiss fontSize={"small"} />
          </IconButton>
        </Grow>
      </Typography>
      <Box>
        {condition.type == ConditionType.name && (
          <FileNameCondition condition={condition} onChange={onChange} onNameConditionAdded={onNameConditionAdded} />
        )}
        {condition.type == ConditionType.type && <FileTypeCondition condition={condition} onChange={onChange} />}
        {condition.type == ConditionType.base && <SearchBaseCondition condition={condition} onChange={onChange} />}
        {condition.type == ConditionType.tag && <TagCondition onChange={onChange} condition={condition} />}
        {condition.type == ConditionType.metadata && !customPropsOption && (
          <MetadataCondition onChange={onChange} condition={condition} />
        )}
        {condition.type == ConditionType.metadata && customPropsOption && (
          <CustomPropsConditon onChange={onChange} condition={condition} option={customPropsOption} />
        )}
        {condition.type == ConditionType.size && <SizeCondition condition={condition} onChange={onChange} />}
        {condition.type == ConditionType.created && (
          <DateTimeCondition condition={condition} onChange={onChange} field={"created"} />
        )}
        {condition.type == ConditionType.modified && (
          <DateTimeCondition condition={condition} onChange={onChange} field={"updated"} />
        )}
      </Box>
    </StyledBox>
  );
});

export default ConditionBox;
