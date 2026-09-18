import { DenseFilledTextField } from "../../Common/StyledComponents.tsx";
import { InputAdornment } from "@mui/material";
import CircleColorSelector, {
  customizeMagicColor,
} from "../../FileManager/FileInfo/ColorCircle/CircleColorSelector.tsx";
import * as React from "react";

export interface HexColorInputProps {
  currentColor: string;
  onColorChange: (color: string) => void;
  required?: boolean;
}

const HexColorInput = ({ currentColor, onColorChange, ...rest }: HexColorInputProps) => {
  return (
    <DenseFilledTextField
      value={currentColor}
      onChange={(e) => {
        onColorChange(e.target.value);
      }}
      type="text"
      InputProps={{
        endAdornment: (
          <InputAdornment position="end">
            <CircleColorSelector
              showColorValueInCustomization
              colors={[customizeMagicColor]}
              selectedColor={currentColor}
              onChange={(color) => onColorChange(color)}
            />
          </InputAdornment>
        ),
      }}
      {...rest}
    />
  );
};

export default HexColorInput;
