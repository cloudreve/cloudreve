import { MagicVar } from "../../../Common/MagicVarDialog";

export const commonMagicVars: MagicVar[] = [
  { name: "{randomkey16}", value: "policy.magicVar.16digitsRandomString", example: "a1b2c3d4e5f6g7h8" },
  { name: "{randomkey8}", value: "policy.magicVar.8digitsRandomString", example: "a1b2c3d4" },
  { name: "{timestamp}", value: "policy.magicVar.secondTimestamp", example: "1609459200" },
  { name: "{timestamp_nano}", value: "policy.magicVar.nanoTimestamp", example: "1609459200000000000" },
  { name: "{randomnum2}", value: "policy.magicVar.randomNumber", example: "0-1" },
  { name: "{randomnum3}", value: "policy.magicVar.randomNumber", example: "0-2" },
  { name: "{randomnum4}", value: "policy.magicVar.randomNumber", example: "0-3" },
  { name: "{randomnum8}", value: "policy.magicVar.randomNumber", example: "0-7" },
  { name: "{uid}", value: "policy.magicVar.uid", example: "1" },
  { name: "{datetime}", value: "policy.magicVar.dateAndTime", example: "20220101120000" },
  { name: "{date}", value: "policy.magicVar.date", example: "20220101" },
  { name: "{year}", value: "policy.magicVar.year", example: "2022" },
  { name: "{month}", value: "policy.magicVar.month", example: "01" },
  { name: "{day}", value: "policy.magicVar.day", example: "01" },
  { name: "{hour}", value: "policy.magicVar.hour", example: "12" },
  { name: "{minute}", value: "policy.magicVar.minute", example: "00" },
  { name: "{second}", value: "policy.magicVar.second", example: "00" },
  { name: "{uuid}", value: "policy.magicVar.uuidV4", example: "550e8400-e29b-41d4-a716-446655440000" },
  { name: "{ext}", value: "policy.magicVar.extension", example: ".jpg" },
  { name: "{originname}", value: "policy.magicVar.originalFileName", example: "example.jpg" },
  { name: "{originname_without_ext}", value: "policy.magicVar.originFileNameNoext", example: "example" },
];

export const pathMagicVars: MagicVar[] = [
  ...commonMagicVars,
  { name: "{path}", value: "policy.magicVar.path", example: "/path/to/" },
];

export const fileMagicVars: MagicVar[] = [...commonMagicVars];

export const thumbnailMagicVars: MagicVar[] = [
  ...pathMagicVars,
  { name: "{blob_name}", value: "policy.magicVar.blobName", example: "a1b2c3d4_example.mp4" },
  { name: "{blob_name_without_ext}", value: "policy.magicVar.blobNameNoext", example: "abcd1234_example" },
  { name: "{blob_path}", value: "policy.magicVar.blobPath", example: "/data/i-m-physical-path/" },
];
