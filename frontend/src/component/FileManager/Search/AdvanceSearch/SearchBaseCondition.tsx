import { PathSelectorForm } from "../../../Common/Form/PathSelectorForm.tsx";
import { defaultPath } from "../../../../hooks/useNavigation.tsx";
import { Condition } from "./ConditionBox.tsx";

export const SearchBaseCondition = ({
  condition,
  onChange,
}: {
  onChange: (condition: Condition) => void;
  condition: Condition;
}) => {
  return (
    <PathSelectorForm
      onChange={(path) => onChange({ ...condition, base_uri: path })}
      path={condition.base_uri ?? defaultPath}
      variant={"searchIn"}
      textFieldProps={{
        sx: {
          "& .MuiOutlinedInput-input": {
            paddingTop: "15.5px",
            paddingBottom: "15.5px",
          },
        },
      }}
    />
  );
};
