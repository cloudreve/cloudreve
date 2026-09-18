import { createContext } from "react";
import type { DataType } from "./types";

export type UpdateItemType = (dataType: DataType) => void;

export interface PhotoContextType {
  show: (key: number) => void;
  update: UpdateItemType;
  remove: (key: number) => void;
  nextId: () => number;
}

export default createContext<PhotoContextType>(undefined as unknown as PhotoContextType);
