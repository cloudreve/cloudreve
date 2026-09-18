use serde::{Deserialize, Serialize};

/// Payment status
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum PaymentStatus {
    Created,
    Paid,
    Fulfilled,
    FulfillFailed,
    Canceled,
}

/// Product type
#[derive(Debug, Clone, Copy, Serialize, Deserialize)]
pub enum ProductType {
    DoNotUse = 0,
    ShareLink = 1,
    Group = 2,
    Storage = 3,
    Points = 4,
}

/// Payment provider type
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum PaymentProviderType {
    Stripe,
    Weixin,
    Alipay,
    Points,
    Custom,
}

/// Payment information
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Payment {
    pub id: String,
    pub trade_no: String,
    pub name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub status: Option<PaymentStatus>,
    pub qyt: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub price_unit: Option<i64>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub price_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub price_one_unit: Option<i64>,
    pub created_at: String,
    pub updated_at: String,
    pub product_type: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub ticket: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub price_mark: Option<String>,
}

/// Payment setting
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaymentSetting {
    pub currency_code: String,
    pub currency_mark: String,
    pub currency_unit: i32,
    pub providers: Vec<PaymentProvider>,
}

/// Payment provider
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaymentProvider {
    pub id: String,
    pub name: String,
    #[serde(rename = "type")]
    pub provider_type: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub secret_key: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub app_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub public_key: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub merchant_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub certificate_serial: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub api_private_key: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub endpoint: Option<String>,
}

/// Product parameter
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ProductParameter {
    #[serde(rename = "type")]
    pub product_type: ProductType,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub share_link_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sku_id: Option<String>,
}

/// Create payment args
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreatePaymentArgs {
    pub product: ProductParameter,
    pub quantity: i32,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub provider_id: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub email: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub language: Option<String>,
}

/// Payment request
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PaymentRequest {
    pub payment_needed: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub url: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub qr_code_preferred: Option<bool>,
}

/// Create payment response
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CreatePaymentResponse {
    pub payment: Payment,
    pub request: PaymentRequest,
}

/// Storage product
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct StorageProduct {
    pub id: String,
    pub name: String,
    pub size: i64,
    pub time: i64,
    pub price: i64,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub chip: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub points: Option<i32>,
}

/// Group SKU
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GroupSku {
    pub id: String,
    pub name: String,
    pub price: i64,
    pub points: i32,
    pub time: i64,
    pub chip: String,
    pub des: Vec<String>,
}

/// Gift code
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GiftCode {
    pub id: i32,
    pub created_at: String,
    pub updated_at: String,
    pub code: String,
    pub used: bool,
    pub qyt: i32,
    pub used_by: i32,
    pub product_props: ProductParameter,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub user_hash_id: Option<String>,
}

/// Generate redeems service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct GenerateRedeemsService {
    pub num: i32,
    pub product: ProductParameter,
    pub qyt: i32,
}

/// Delete gift code service
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DeleteGiftCodeService {
    pub id: i32,
}

