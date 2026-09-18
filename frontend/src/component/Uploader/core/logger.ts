export type LogLevel = "INFO" | "WARN" | "ERROR" | "OFF";

export default class Logger {
  constructor(
    public level: LogLevel = "OFF",
    private prefix = "UPLOAD",
    private id: number = 1,
  ) {}

  private getPrintPrefix(level: LogLevel) {
    return `Cloudreve-Uploader [${level}][${this.prefix}#${this.id}]:`;
  }

  info(...args: unknown[]) {
    const allowLevel: LogLevel[] = ["INFO"];
    if (allowLevel.includes(this.level)) {
      // eslint-disable-next-line no-console
      console.log(this.getPrintPrefix("INFO"), ...args);
    }
  }

  warn(...args: unknown[]) {
    const allowLevel: LogLevel[] = ["INFO", "WARN"];
    if (allowLevel.includes(this.level)) {
      // eslint-disable-next-line no-console
      console.warn(this.getPrintPrefix("WARN"), ...args);
    }
  }

  error(...args: unknown[]) {
    const allowLevel: LogLevel[] = ["INFO", "WARN", "ERROR"];
    if (allowLevel.includes(this.level)) {
      // eslint-disable-next-line no-console
      console.error(this.getPrintPrefix("ERROR"), ...args);
    }
  }
}
