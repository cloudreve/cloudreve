import { StoragePolicy } from "../../../../api/explorer.ts";
import { FileValidateError } from "../errors";

interface Validator {
  (file: File, policy: StoragePolicy): void;
}

// validators
const checkers: Array<Validator> = [
  function checkExt(file: File, policy: StoragePolicy) {
    const ext = file?.name.split(".").pop();
    if (policy.allowed_suffix != undefined && policy.allowed_suffix.length > 0) {
      if (ext === null || !ext || !policy.allowed_suffix.includes(ext))
        throw new FileValidateError("File suffix not allowed in policy.", "suffix", policy);
    }
    if (policy.denied_suffix != undefined && policy.denied_suffix.length > 0) {
      if (ext && policy.denied_suffix.includes(ext))
        throw new FileValidateError("File suffix not allowed in policy.", "suffix_denied", policy);
    }
  },

  function checkRegexp(file: File, policy: StoragePolicy) {
    if (policy.allowed_name_regexp != undefined && policy.allowed_name_regexp.length > 0) {
      if (!new RegExp(policy.allowed_name_regexp).test(file.name))
        throw new FileValidateError("File name must match the allowed regexp.", "regexp", policy);
    }
    if (policy.denied_name_regexp != undefined && policy.denied_name_regexp.length > 0) {
      if (new RegExp(policy.denied_name_regexp).test(file.name))
        throw new FileValidateError("File name must not match the regexp.", "regexp", policy);
    }
  },

  function checkSize(file: File, policy: StoragePolicy) {
    if (policy.max_size > 0) {
      if (file.size > policy.max_size) {
        throw new FileValidateError("File size exceeds maximum limit.", "size", policy);
      }
    }
  },
];

/* 将每个 Validator 执行
   失败返回 Error
 */
export function validate(file: File, policy: StoragePolicy) {
  checkers.forEach((c) => c(file, policy));
}
