import path from "path-browserify";
import { FileType, Metadata } from "../api/explorer.ts";
import SessionManager from "../session";

export const CrUriPrefix = "cloudreve://";
const HttpUriPrefix = "http://";

export const Filesystem = {
  my: "my",
  share: "share",
  shared_by_me: "shared_by_me",
  shared_with_me: "shared_with_me",
  trash: "trash",
};

export const UriQuery = {
  name: "name",
  name_op_or: "name_op_or",
  metadata_prefix: "meta_",
  metadata_strong_match: "exact_meta_",
  case_folding: "case_folding",
  type: "type",
  category: "category",
  size_gte: "size_gte",
  size_lte: "size_lte",
  created_gte: "created_gte",
  created_lte: "created_lte",
  updated_gte: "updated_gte",
  updated_lte: "updated_lte",
};

export const UriSearchCategory = {
  image: "image",
  video: "video",
  audio: "audio",
  document: "document",
};

export interface SearchParam {
  name?: string[];
  name_op_or?: boolean;
  metadata?: {
    [key: string]: string;
  };
  metadata_strong_match?: {
    [key: string]: string;
  };
  case_folding?: boolean;
  category?: string;
  type?: number;
  size_gte?: number;
  size_lte?: number;
  created_at_gte?: number;
  created_at_lte?: number;
  updated_at_gte?: number;
  updated_at_lte?: number;
}

export default class CrUri {
  private url: URL;
  constructor(u: string) {
    if (!u.startsWith(CrUriPrefix)) {
      throw new Error("Invalid cloudreve uri");
    }

    // replacing prefix with standard HTTP for compatibility
    u = u.replace(CrUriPrefix, HttpUriPrefix);
    this.url = new URL(u);
    // remove ending slash if presented
    this.url.pathname = this.url.pathname.replace(/\/$/, "");
  }

  public id(): string {
    return this.url.username;
  }

  public password(): string {
    return this.url.password;
  }

  public is_search(): boolean {
    return !!this.url.searchParams.keys().next().value;
  }

  public query(key: string): string[] {
    return this.url.searchParams.getAll(key);
  }

  public addQuery(key: string, value: string): this {
    this.url.searchParams.append(key, value);
    return this;
  }

  public setSearchParam(param: SearchParam): this {
    this.url.searchParams.forEach((_, k) => {
      this.url.searchParams.delete(k);
    });
    if (param.name) {
      if (this.fs() == Filesystem.trash) {
        this.addQuery(UriQuery.metadata_prefix + Metadata.restore_uri, encodeURI(param.name.join(" ")));
      } else {
        param.name.forEach((name) => this.addQuery(UriQuery.name, name));
      }
    }
    if (param.name_op_or) {
      this.addQuery(UriQuery.name_op_or, "");
    }
    if (param.case_folding) {
      this.addQuery(UriQuery.case_folding, "");
    }
    if (param.category) {
      this.addQuery(UriQuery.category, param.category);
    }
    if (param.type !== undefined) {
      this.addQuery(UriQuery.type, param.type == FileType.folder ? "folder" : "file");
    }
    if (param.metadata) {
      Object.entries(param.metadata).forEach(([k, v]) => {
        this.addQuery(UriQuery.metadata_prefix + k, v);
      });
    }
    if (param.metadata_strong_match) {
      Object.entries(param.metadata_strong_match).forEach(([k, v]) => {
        this.addQuery(UriQuery.metadata_strong_match + k, v);
      });
    }
    if (param.size_gte) {
      this.addQuery(UriQuery.size_gte, param.size_gte.toString());
    }
    if (param.size_lte) {
      this.addQuery(UriQuery.size_lte, param.size_lte.toString());
    }
    if (param.created_at_gte) {
      this.addQuery(UriQuery.created_gte, param.created_at_gte.toString());
    }
    if (param.created_at_lte) {
      this.addQuery(UriQuery.created_lte, param.created_at_lte.toString());
    }
    if (param.updated_at_gte) {
      this.addQuery(UriQuery.updated_gte, param.updated_at_gte.toString());
    }
    if (param.updated_at_lte) {
      this.addQuery(UriQuery.updated_lte, param.updated_at_lte.toString());
    }
    return this;
  }

  public searchParams(): SearchParam | undefined {
    if (!this.is_search()) {
      return undefined;
    }

    const res: SearchParam = {};
    this.url.searchParams.forEach((v, k) => {
      switch (k) {
        case UriQuery.name:
          res.name = this.url.searchParams.getAll(k);
          break;
        case UriQuery.name_op_or:
          res.name_op_or = true;
          break;
        case UriQuery.case_folding:
          res.case_folding = true;
          break;
        case UriQuery.category:
          res.category = v;
          break;
        case UriQuery.type:
          if (v === "file") {
            res.type = FileType.file;
          } else {
            res.type = FileType.folder;
          }
          break;
        case UriQuery.size_gte:
          res.size_gte = parseInt(v);
          break;
        case UriQuery.size_lte:
          res.size_lte = parseInt(v);
          break;
        case UriQuery.created_gte:
          res.created_at_gte = parseInt(v);
          break;
        case UriQuery.created_lte:
          res.created_at_lte = parseInt(v);
          break;
        case UriQuery.updated_gte:
          res.updated_at_gte = parseInt(v);
          break;
        case UriQuery.updated_lte:
          res.updated_at_lte = parseInt(v);
          break;
        default:
          if (k.startsWith(UriQuery.metadata_prefix)) {
            if (!res.metadata) {
              res.metadata = {};
            }
            res.metadata[k.slice(UriQuery.metadata_prefix.length)] = v;
          } else if (k.startsWith(UriQuery.metadata_strong_match)) {
            if (!res.metadata_strong_match) {
              res.metadata_strong_match = {};
            }
            res.metadata_strong_match[k.slice(UriQuery.metadata_strong_match.length)] = v;
          }
      }
    });

    return res;
  }

  public path(): string {
    return decodeURIComponent(this.url.pathname);
  }

  public setPath(path: string): this {
    this.url.pathname = path
      .split("/")
      .map((p) => encodeURIComponent(p))
      .join("/");
    return this;
  }

  public setUsername(username: string): this {
    this.url.username = username;
    return this;
  }

  public setPassword(password: string): this {
    this.url.password = password;
    return this;
  }

  // path without leading slash
  public path_trimmed(): string {
    return this.url.pathname.slice(1);
  }

  public join(...paths: string[]): this {
    this.url.pathname = path.join(this.url.pathname, ...paths.map((p) => encodeURIComponent(p)));
    return this;
  }

  public join_raw(rawPath: string): this {
    if (rawPath.startsWith("/")) {
      // Absolute path - replace the entire pathname
      this.url.pathname = rawPath;
    } else {
      // Relative path - resolve it against the current path
      this.url.pathname = path.resolve(this.url.pathname, rawPath);
    }
    return this;
  }

  public elements(): string[] {
    const res = this.path_trimmed()
      .split("/")
      .map((p) => decodeURIComponent(p));
    if (res.length == 1 && res[0] == "") {
      return [];
    }

    return res;
  }

  public is_root(): boolean {
    return this.url.pathname == "" || this.url.pathname == "/";
  }

  public fs(): string {
    return this.url.hostname;
  }

  public root_id(): string {
    return `${this.fs()}/${this.url.username}/${SessionManager.currentLoginOrNull()?.user.id ?? "0"}`;
  }

  public base(excludeSearch: boolean = true): string {
    const newUri = new URL(this.url.toString());
    newUri.pathname = "";
    if (excludeSearch) {
      newUri.search = "";
    }

    // remove ending slash

    return newUri.toString().replace(HttpUriPrefix, CrUriPrefix).replace(/\/$/, "");
  }

  // pure_uri returns the uri without searching query string, with exceptions.
  public pure_uri(...exceptions: string[]): CrUri {
    const newUri = new CrUri(this.toString());
    let keysForDel: string[] = [];
    newUri.url.searchParams.forEach((_v, k) => {
      if (exceptions && exceptions.includes(k)) {
        return;
      }
      keysForDel.push(k);
    });
    keysForDel.forEach((k) => {
      newUri.url.searchParams.delete(k);
    });

    return newUri;
  }

  public parent(): CrUri {
    const newUri = new CrUri(this.toString());
    const path = newUri.elements();
    path.pop();
    newUri.setPath((path.length > 0 ? "/" : "") + path.join("/"));
    return newUri;
  }

  public toString(): string {
    return this.url.toString().replace(HttpUriPrefix, CrUriPrefix).replace(/\/$/, "");
  }
}

export const newMyUri = (uid?: string): CrUri => {
  return uid ? new CrUri(`cloudreve://${uid}@my`) : new CrUri("cloudreve://my");
};
