use crate::models::user::User;
use crate::models::vas::{GroupSku, PaymentSetting, StorageProduct};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Captcha type
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum CaptchaType {
    Normal,
    Recaptcha,
    Tcaptcha,
    Turnstile,
    Cap,
}

/// Site configuration
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct SiteConfig {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub instance_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub title: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub login_captcha: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reg_captcha: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub forget_captcha: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub abuse_report_captcha: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub themes: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub default_theme: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub authn: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user: Option<User>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub captcha_re_captcha_key: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub site_notice: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub captcha_type: Option<CaptchaType>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub turnstile_site_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub captcha_cap_instance_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub captcha_cap_site_key: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub captcha_cap_secret_key: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub captcha_cap_asset_server: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub register_enabled: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub qq_enabled: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sso_enabled: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sso_display_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sso_icon: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub oidc_enabled: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub oidc_display_name: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub oidc_icon: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub logo: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub logo_light: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub tos_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub privacy_policy_url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub icons: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub emoji_preset: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub point_enabled: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub share_point_gain_rate: Option<f64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub map_provider: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub mapbox_ak: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub google_map_tile_type: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub max_batch_size: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub app_promotion: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub app_feedback: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub app_forum: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub payment: Option<PaymentSetting>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub anonymous_purchase: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub point_price: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub shop_nav_enabled: Option<bool>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub storage_products: Option<Vec<StorageProduct>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub group_skus: Option<Vec<GroupSku>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail_width: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumbnail_height: Option<i32>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub thumb_exts: Option<Vec<String>>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub show_encryption_status: Option<bool>,
}

/// Captcha response
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct CaptchaResponse {
    pub ticket: String,
    pub image: String,
}

/// Create abuse report service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreateAbuseReportService {
    #[serde(skip_serializing_if = "Option::is_none")]
    pub file_uri: Option<String>,
    pub category: i32,
    pub description: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub share_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user_id: Option<String>,
    #[serde(flatten)]
    pub captcha: Option<HashMap<String, serde_json::Value>>,
}
