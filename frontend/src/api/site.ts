import { CustomProps, ViewerGroup } from "./explorer.ts";
import { User } from "./user.ts";

export enum CaptchaType {
  NORMAL = "normal",
  RECAPTCHA = "recaptcha",
  // Deprecated
  TCAPTCHA = "tcaptcha",
  TURNSTILE = "turnstile",
  CAP = "cap",
}

export interface SiteConfig {
  instance_id?: string;
  title?: string;
  login_captcha?: boolean;
  reg_captcha?: boolean;
  forget_captcha?: boolean;
  themes?: string;
  default_theme?: string;
  authn?: boolean;
  user?: User;
  captcha_ReCaptchaKey?: string;
  captcha_type?: CaptchaType;
  turnstile_site_id?: string;
  captcha_cap_instance_url?: string;
  captcha_cap_site_key?: string;
  captcha_cap_secret_key?: string;
  captcha_cap_asset_server?: string;
  register_enabled?: boolean;
  logo?: string;
  logo_light?: string;
  tos_url?: string;
  privacy_policy_url?: string;
  icons?: string;
  emoji_preset?: string;
  map_provider?: string;
  mapbox_ak?: string;
  google_map_tile_type?: string;
  file_viewers?: ViewerGroup[];
  default_viewer_mapping?: {
    [ext: string]: string;
  };
  max_batch_size?: number;
  app_promotion?: boolean;
  desktop_app_promotion?: boolean;
  thumbnail_width?: number;
  thumbnail_height?: number;
  custom_props?: CustomProps[];
  custom_nav_items?: CustomNavItem[];
  custom_html?: CustomHTML;
  thumb_exts?: string[];
  show_encryption_status?: boolean;
  full_text_search?: boolean;
}

export interface CaptchaResponse {
  ticket: string;
  image: string;
}

export interface CustomNavItem {
  name: string;
  url: string;
  icon: string;
}

export interface CustomHTML {
  headless_footer?: string;
  headless_bottom?: string;
  sidebar_bottom?: string;
}
