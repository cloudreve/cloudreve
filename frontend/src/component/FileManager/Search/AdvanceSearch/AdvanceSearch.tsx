import { Collapse, DialogContent } from "@mui/material";
import { useSnackbar } from "notistack";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { TransitionGroup } from "react-transition-group";
import { Metadata } from "../../../../api/explorer.ts";
import { defaultPath } from "../../../../hooks/useNavigation.tsx";
import { closeAdvanceSearch } from "../../../../redux/globalStateSlice.ts";
import { useAppDispatch, useAppSelector } from "../../../../redux/hooks.ts";
import { advancedSearch } from "../../../../redux/thunks/filemanager.ts";
import { SearchParam } from "../../../../util/uri.ts";
import { DefaultCloseAction } from "../../../Common/Snackbar/snackbar.tsx";
import DraggableDialog from "../../../Dialogs/DraggableDialog.tsx";
import { FileManagerIndex } from "../../FileManager.tsx";
import AddCondition from "./AddCondition.tsx";
import ConditionBox, { Condition, ConditionType } from "./ConditionBox.tsx";

const searchParamToConditions = (search_params: SearchParam, base: string): Condition[] => {
  const applied: Condition[] = [
    {
      type: ConditionType.base,
      base_uri: base,
    },
  ];
  if (search_params.name) {
    applied.push({
      type: ConditionType.name,
      names: search_params.name,
      name_op_or: search_params.name_op_or,
      case_folding: search_params.case_folding,
    });
  }

  if (search_params.type != undefined) {
    applied.push({
      type: ConditionType.type,
      file_type: search_params.type,
    });
  }

  if (search_params.size_gte != undefined || search_params.size_lte != undefined) {
    applied.push({
      type: ConditionType.size,
      size_gte: search_params.size_gte,
      size_lte: search_params.size_lte,
    });
  }

  if (search_params.created_at_gte != undefined || search_params.created_at_lte != undefined) {
    applied.push({
      type: ConditionType.created,
      created_gte: search_params.created_at_gte,
      created_lte: search_params.created_at_lte,
    });
  }

  if (search_params.updated_at_gte != undefined || search_params.updated_at_lte != undefined) {
    applied.push({
      type: ConditionType.modified,
      updated_gte: search_params.updated_at_gte,
      updated_lte: search_params.updated_at_lte,
    });
  }

  const tags: string[] = [];
  if (search_params.metadata) {
    Object.entries(search_params.metadata).forEach(([key, value]) => {
      if (key.startsWith(Metadata.tag_prefix)) {
        tags.push(key.slice(Metadata.tag_prefix.length));
      } else {
        applied.push({
          type: ConditionType.metadata,
          metadata_key: key,
          metadata_value: value,
          id: key,
        });
      }
    });
  }

  if (search_params.metadata_strong_match) {
    Object.entries(search_params.metadata_strong_match).forEach(([key, value]) => {
      applied.push({
        type: ConditionType.metadata,
        metadata_key: key,
        metadata_value: value,
        id: key,
        metadata_strong_match: true,
      });
    });
  }

  if (tags.length > 0) {
    applied.push({
      type: ConditionType.tag,
      tags: tags,
    });
  }

  console.log(search_params);

  return applied;
};

const AdvanceSearch = () => {
  const { t } = useTranslation();
  const dispatch = useAppDispatch();
  const { enqueueSnackbar } = useSnackbar();

  const [conditions, setConditions] = useState<Condition[]>([]);
  const open = useAppSelector((state) => state.globalState.advanceSearchOpen);
  const base = useAppSelector((state) => state.globalState.advanceSearchBasePath);
  const initialNames = useAppSelector((state) => state.globalState.advanceSearchInitialNameCondition);
  const search_params = useAppSelector((state) => state.fileManager[FileManagerIndex.main].search_params);
  const current_base = useAppSelector((state) => state.fileManager[FileManagerIndex.main].pure_path);

  const onClose = useCallback(() => {
    dispatch(closeAdvanceSearch());
  }, [dispatch]);

  useEffect(() => {
    if (open) {
      if (initialNames && base) {
        setConditions([
          {
            type: ConditionType.base,
            base_uri: base,
          },
          {
            type: ConditionType.name,
            names: initialNames,
            case_folding: true,
          },
        ]);
        return;
      }

      if (search_params) {
        const existedConditions = searchParamToConditions(search_params, current_base ?? defaultPath);
        if (existedConditions.length > 0) {
          setConditions(existedConditions);
        }
      }
    }
  }, [open]);

  const onConditionRemove = (condition: Condition) => {
    setConditions(conditions.filter((c) => c !== condition));
  };

  const onConditionAdd = (condition: Condition) => {
    if (conditions.find((c) => c.type === condition.type && c.id === condition.id)) {
      enqueueSnackbar(t("application:navbar.conditionDuplicate"), {
        variant: "warning",
        action: DefaultCloseAction,
      });
      return;
    }

    setConditions([...conditions, condition]);
  };

  const submitSearch = useCallback(() => {
    dispatch(advancedSearch(FileManagerIndex.main, conditions));
  }, [dispatch, conditions]);

  return (
    <DraggableDialog
      title={t("application:navbar.advancedSearch")}
      showActions
      onAccept={submitSearch}
      showCancel
      secondaryAction={<AddCondition onConditionAdd={onConditionAdd} />}
      dialogProps={{
        open: open ?? false,
        onClose: onClose,
        fullWidth: true,
        maxWidth: "xs",
      }}
    >
      <DialogContent>
        <TransitionGroup>
          {conditions.map((condition, index) => (
            <Collapse key={`${condition.type} ${condition.id}`}>
              <ConditionBox
                index={index}
                onRemove={conditions.length > 2 && condition.type != ConditionType.base ? onConditionRemove : undefined}
                condition={condition}
                onChange={(condition) => {
                  const new_conditions = [...conditions];
                  new_conditions[index] = condition;
                  setConditions(new_conditions);
                }}
              />
            </Collapse>
          ))}
        </TransitionGroup>
      </DialogContent>
    </DraggableDialog>
  );
};

export default AdvanceSearch;
