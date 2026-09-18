import SessionManager, { UserSettings } from "./index.ts";
import { defaultColors, maxRecentColors, maxRecentTags } from "../constants";
import { Tag } from "../component/FileManager/Dialogs/Tags.tsx";
import Boolset from "../util/boolset.ts";
import { User } from "../api/user.ts";

export const addRecentUsedColor = (color: string | undefined, setting: string) => {
  if (!color || defaultColors.includes(color)) return;
  let colors = SessionManager.get(setting) as string[];
  if (!colors) colors = [];
  if (color && !colors.includes(color)) {
    colors.push(color);
    // Trim to maxRecentColors
    if (colors.length > maxRecentColors) {
      colors = colors.slice(colors.length - maxRecentColors);
    }
    SessionManager.set(setting, colors);
  }
};

export interface UsedTags {
  [key: string]: string | null;
}
export const addUsedTags = (tags: Tag[]) => {
  let existing = SessionManager.get(UserSettings.UsedTags) as UsedTags;
  if (!existing) existing = {};

  tags.forEach((tag) => {
    if (existing[tag.key]) {
      delete existing[tag.key];
    }

    existing[tag.key] = tag.color ?? null;
  });

  if (Object.keys(existing).length > maxRecentTags) {
    const removedKeys = Object.keys(existing).slice(0, Object.keys(existing).length - maxRecentTags);
    removedKeys.forEach((key) => {
      delete existing[key];
    });
  }

  SessionManager.set(UserSettings.UsedTags, existing);
};

export const GroupBS = (user?: User): Boolset => {
  return new Boolset(user?.group?.permission);
};

export const isTrueVal = (s: string): boolean => {
  return s === "true" || s === "1" || s === "yes" || s === "on";
};
