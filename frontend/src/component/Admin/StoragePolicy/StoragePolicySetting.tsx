import { Box, Container, Link, ListItemText, Stack, Typography } from "@mui/material";
import Grid from "@mui/material/Grid2";
import { SelectChangeEvent } from "@mui/material/Select";
import { useQueryState } from "nuqs";
import { useCallback, useEffect, useState } from "react";
import { Trans, useTranslation } from "react-i18next";
import { getStoragePolicyList } from "../../../api/api";
import { StoragePolicy } from "../../../api/dashboard";
import { PolicyType } from "../../../api/explorer";
import { useAppDispatch } from "../../../redux/hooks";
import { DenseSelect, SecondaryButton } from "../../Common/StyledComponents";
import { SquareMenuItem } from "../../FileManager/ContextMenu/ContextMenu";
import Add from "../../Icons/Add";
import ArrowSync from "../../Icons/ArrowSync";
import PageContainer from "../../Pages/PageContainer";
import PageHeader from "../../Pages/PageHeader";
import { BorderedCardClickable } from "../Common/AdminCard";
import { Code } from "../../Common/Code.tsx";
import TablePagination from "../Common/TablePagination";
import AddWizardDialog, { AddWizardProps } from "./AddWizardDialog";
import SelectProvider from "./SelectProvider";
import StoragePolicyCard from "./StoragePolicyCard";
import CosWizard from "./Wizards/COS/CosWizard";
import LocalWizard from "./Wizards/Local/LocalWizard";
import ObsWizard from "./Wizards/OBS/ObsWizard";
import OneDriveWizard from "./Wizards/OneDrive/OneDriveWizard";
import OssWizard from "./Wizards/OSS/OssWizard";
import QiniuWizard from "./Wizards/Qiniu/QiniuWizard";
import RemoteWizard from "./Wizards/Remote/RemoteWizard";
import S3Wizard from "./Wizards/S3/S3Wizard";
import KS3Wizard from "./Wizards/KS3/KS3Wizard";
import UpyunWizard from "./Wizards/Upyun/UpyunWizard";

export const PageQuery = "page";
export const PageSizeQuery = "page_size";
export const OrderByQuery = "order_by";
export const OrderDirectionQuery = "order_direction";
export const PolicyTypeQuery = "policy_type";

export interface PolicyProps {
  name: string;
  img: string;
  wizardSize?: "sm" | "md" | "lg";
  wizard?: (props: AddWizardProps) => React.ReactNode;
  chunkSizeDes?: string;
  chunkSizeMin?: number;
  chunkSizeMax?: number;
  nativeThumbDes?: string;
  nativeThumbDoc?: string;
  nativeExtractorName?: string;
  nativeExtractorDoc?: string;
  nativeExtractorDes?: string;
  nativeExtractorDesDoc?: string;
  bucketName?: string;
  bucketNameDes?: React.ReactNode;
  bucketType?: string;
  endpointName?: string;
  endpointDes?: React.ReactNode;
  akName?: string;
  skName?: string;
  credentialDes?: React.ReactNode;
  corsExposedHeaders?: string[];
  endpointNotEnforcePrefix?: boolean;
  pro?: boolean;
  regionCode?: string;
  regionCodeDes?: React.ReactNode;
}

export const PolicyPropsMap: Record<PolicyType, PolicyProps> = {
  [PolicyType.local]: {
    name: "policy.local",
    img: "/static/img/local.png",
    wizardSize: "sm",
    wizard: LocalWizard,
    chunkSizeDes: "policy.chunkSizeDes",
  },
  [PolicyType.load_balance]: {
    name: "policy.load_balance",
    img: "/static/img/lb.svg",
    wizardSize: "sm",
    pro: true,
  },
  [PolicyType.remote]: {
    name: "policy.remote",
    img: "/static/img/remote.png",
    wizardSize: "sm",
    wizard: RemoteWizard,
    nativeThumbDes: "policy.nativeThumbNailsGeneralRemote",
    nativeExtractorName: "policy.mediaExtractorNative",
    nativeExtractorDes: "policy.nativeMediaMetaExtsRemote",
    chunkSizeDes: "policy.chunkSizeDes",
  },
  [PolicyType.s3]: {
    name: "policy.s3",
    img: "/static/img/s3.png",
    wizardSize: "sm",
    wizard: S3Wizard,
    bucketName: "policy.bucketName",
    bucketType: "policy.bucketType",
    endpointName: "policy.policyEndpoint",
    endpointDes: <Trans i18nKey="policy.s3EndpointDes" ns="dashboard" components={[<Code />]} />,
    akName: "Access Key",
    skName: "Secret Key",
    chunkSizeMin: 5 * 1024 * 1024, //5MB
    chunkSizeMax: 5 * 1024 * 1024 * 1024, //5GB
    chunkSizeDes: "policy.chunkSizeDesS3",
    regionCode: "policy.s3Region",
    regionCodeDes: <Trans i18nKey={"policy.selectRegionDes"} ns="dashboard" components={[<Code />]} />,
  },
  [PolicyType.ks3]: {
    name: "policy.ks3",
    img: "/static/img/ks3.png",
    wizardSize: "sm",
    wizard: KS3Wizard,
    bucketName: "policy.bucketName",
    bucketType: "policy.bucketType",
    endpointName: "policy.policyEndpoint",
    endpointDes: <Trans i18nKey="policy.ks3selectRegionDes" ns="dashboard" components={[<Code />]} />,
    akName: "Access Key",
    skName: "Secret Key",
    corsExposedHeaders: ["ETag"],
    chunkSizeMin: 5 * 1024 * 1024, //5MB
    chunkSizeMax: 5 * 1024 * 1024 * 1024, //5GB
    chunkSizeDes: "policy.chunkSizeDesS3",
    regionCode: "policy.s3Region",
    regionCodeDes: <Trans i18nKey={"policy.ks3selectRegionDes"} ns="dashboard" components={[<Code />]} />,
  },
  [PolicyType.cos]: {
    name: "policy.cos",
    img: "/static/img/cos.png",
    wizardSize: "sm",
    wizard: CosWizard,
    bucketName: "policy.cosObsBucketName",
    bucketNameDes: (
      <Trans
        i18nKey="policy.createCOSBucketDes"
        ns="dashboard"
        components={[<Link href="https://console.cloud.tencent.com/cos/bucket" target="_blank" />, <Code />]}
      />
    ),
    bucketType: "policy.accessType",
    endpointName: "policy.accessDomain",
    endpointDes: <Trans i18nKey="policy.cosAccessDomainDes" ns="dashboard" components={[<Code />, <Code />]} />,
    chunkSizeMin: 1024 * 1024, //1MB
    chunkSizeMax: 5 * 1024 * 1024 * 1024, //5GB
    chunkSizeDes: "policy.chunkSizeDesQiniuCos",
    nativeThumbDes: "policy.nativeThumbNailsGeneralCos",
    nativeThumbDoc: "https://cloud.tencent.com/document/product/436/113312",
    nativeExtractorName: "policy.mediaExtractorCos",
    nativeExtractorDes: "policy.nativeMediaMetaExtCos",
    nativeExtractorDesDoc: "https://console.cloud.tencent.com/ci",
    nativeExtractorDoc: "https://console.cloud.tencent.com/ci",
    akName: "SecretId",
    skName: "SecretKey",
    corsExposedHeaders: ["ETag"],
    credentialDes: (
      <Trans
        i18nKey="policy.cosCredentialDes"
        ns="dashboard"
        components={[
          <Link href="https://console.cloud.tencent.com/cam/capi" target="_blank" />,
          <Code />,
          <Link href="https://console.cloud.tencent.com/cam" target="_blank" />,
        ]}
      />
    ),
  },
  [PolicyType.oss]: {
    name: "policy.oss",
    img: "/static/img/oss.png",
    wizardSize: "sm",
    wizard: OssWizard,
    chunkSizeMin: 100 * 1024, //100KB
    chunkSizeMax: 5 * 1024 * 1024 * 1024, //5GB
    chunkSizeDes: "policy.chunkSizeDesOssObs",
    nativeThumbDes: "policy.nativeThumbNailsGeneralOss",
    nativeThumbDoc: "https://help.aliyun.com/zh/oss/user-guide/overview-17/",
    nativeExtractorName: "policy.mediaExtractorOss",
    nativeExtractorDes: "policy.nativeMediaMetaExtOss",
    nativeExtractorDoc: "https://help.aliyun.com/zh/oss/user-guide/quick-start-2326698",
    nativeExtractorDesDoc: "https://help.aliyun.com/zh/oss/user-guide/quick-start-2326698",
    bucketName: "policy.bucketName",
    bucketNameDes: (
      <Trans
        i18nKey="policy.createOSSBucketDes"
        ns="dashboard"
        components={[<Link href="https://oss.console.aliyun.com/bucket" target="_blank" />, <Code />, <Code />]}
      />
    ),
    bucketType: "policy.bucketType",
    akName: "AccessKey ID",
    skName: "AccessKey Secret",
    credentialDes: (
      <Trans
        i18nKey="policy.ossAKDes"
        ns="dashboard"
        components={[
          <Link href="https://ram.console.aliyun.com/profile/access-keys" target="_blank" />,
          <Link href="https://ram.console.aliyun.com/ram/overview" target="_blank" />,
          <Code />,
        ]}
      />
    ),
    regionCode: "policy.s3Region",
    regionCodeDes: (
      <Trans
        i18nKey={"policy.ossRegionDes"}
        ns="dashboard"
        components={[
          <Code />,
          <Link href="https://www.alibabacloud.com/help/oss/regions-and-endpoints" target="_blank" />,
          <Code />,
        ]}
      />
    ),
  },
  [PolicyType.obs]: {
    name: "policy.obs",
    img: "/static/img/obs.png",
    wizardSize: "sm",
    wizard: ObsWizard,
    bucketName: "policy.cosObsBucketName",
    bucketNameDes: (
      <Trans
        i18nKey="policy.obsBucketDes"
        ns="dashboard"
        components={[
          <Link href="https://console.huaweicloud.com/console/#/obs/create" target="_blank" />,
          <Code />,
          <Code />,
          <Code />,
        ]}
      />
    ),
    bucketType: "policy.bucketPolicy",
    endpointName: "policy.policyEndpoint",
    endpointDes: <Trans i18nKey="policy.obsEndpointDes" ns="dashboard" components={[<Code />, <Code />]} />,
    endpointNotEnforcePrefix: true,
    akName: "Access Key Id",
    skName: "Secret Access Key",
    credentialDes: (
      <Trans
        i18nKey="policy.obsCredentialDes"
        ns="dashboard"
        components={[
          <Link href="https://console.huaweicloud.com/iam/#/mine/accessKey" target="_blank" />,
          <Code />,
          <Code />,
          <Code />,
        ]}
      />
    ),
    corsExposedHeaders: ["ETag"],
    chunkSizeMin: 100 * 1024, //100KB
    chunkSizeMax: 5 * 1024 * 1024 * 1024, //5GB
    chunkSizeDes: "policy.chunkSizeDesOssObs",
    nativeThumbDes: "policy.nativeThumbNailsGeneralObs",
    nativeThumbDoc: "https://support.huaweicloud.com/intl/zh-cn/usermanual-obs/obs_01_0001.html",
    nativeExtractorName: "policy.mediaExtractorObs",
    nativeExtractorDes: "policy.nativeMediaMetaExtObs",
    nativeExtractorDoc: "https://support.huaweicloud.com/intl/zh-cn/usermanual-obs/obs_01_0410.html",
    nativeExtractorDesDoc: "https://support.huaweicloud.com/intl/zh-cn/usermanual-obs/obs_01_0410.html",
  },
  [PolicyType.qiniu]: {
    name: "policy.qiniu",
    img: "/static/img/qiniu.png",
    wizardSize: "sm",
    wizard: QiniuWizard,
    bucketName: "policy.qiniuBucketName",
    bucketNameDes: (
      <Trans
        i18nKey="policy.enterQiniuBucket"
        ns="dashboard"
        components={[<Link href="https://portal.qiniu.com/kodo/bucket" target="_blank" />]}
      />
    ),
    bucketType: "policy.aclType",
    akName: "AK",
    skName: "SK",
    credentialDes: <Trans i18nKey="policy.qiniuCredentialDes" ns="dashboard" />,
    nativeThumbDes: "policy.nativeThumbNailsGeneralQiniu",
    nativeThumbDoc: "https://developer.qiniu.com/dora/api/basic-processing-images-imageview2",
    nativeExtractorName: "policy.mediaExtractorQiniu",
    nativeExtractorDes: "policy.nativeMediaMetaExtQiniu",
    nativeExtractorDoc: "https://www.qiniu.com/products/dora",
    chunkSizeMin: 1024 * 1024, //1 MB
    chunkSizeMax: 1024 * 1024 * 1024, //1GB
    chunkSizeDes: "policy.chunkSizeDesQiniuCos",
  },
  [PolicyType.upyun]: {
    name: "policy.upyun",
    img: "/static/img/upyun.png",
    wizardSize: "sm",
    wizard: UpyunWizard,
    bucketName: "policy.storageServiceName",
    bucketNameDes: (
      <Trans
        i18nKey="policy.createUpyunBucketDes"
        ns="dashboard"
        components={[<Link href="https://console.upyun.com/console/service" target="_blank" />]}
      />
    ),
    bucketType: "policy.tokenStatus",
    akName: "policy.operatorName",
    skName: "policy.operatorPassword",
    nativeThumbDes: "policy.nativeThumbNailsGeneralUpyun",
    nativeThumbDoc: "https://help.upyun.com/knowledge-base/image/",
    nativeExtractorName: "policy.mediaExtractorUpyun",
    nativeExtractorDes: "policy.nativeMediaMetaExtUpyun",
    nativeExtractorDesDoc: "https://help.upyun.com/knowledge-base/image/#e58583e695b0e68daee88eb7e58f96",
    nativeExtractorDoc: "https://help.upyun.com/knowledge-base/image/",
  },
  [PolicyType.onedrive]: {
    name: "policy.onedrive",
    img: "/static/img/onedrive.png",
    wizardSize: "sm",
    wizard: OneDriveWizard,
    chunkSizeMin: 5 * 1024 * 1024, //5MB
    chunkSizeMax: 5 * 1024 * 1024 * 1024, //5GB
    chunkSizeDes: "policy.chunkSizeDesOd",
    bucketName: "policy.storageServiceName",
    bucketNameDes: (
      <Trans
        i18nKey="policy.createUpyunBucketDes"
        ns="dashboard"
        components={[<Link href="https://console.upyun.com/console/service" target="_blank" />]}
      />
    ),
    bucketType: "policy.tokenStatus",
  },
};

const StoragePolicySetting = () => {
  const { t } = useTranslation("dashboard");
  const dispatch = useAppDispatch();
  const [loading, setLoading] = useState(true);
  const [policies, setPolicies] = useState<StoragePolicy[]>([]);
  const [page, setPage] = useQueryState(PageQuery, { defaultValue: "1" });
  const [pageSize, setPageSize] = useQueryState(PageSizeQuery, {
    defaultValue: "11",
  });
  const [orderBy, setOrderBy] = useQueryState(OrderByQuery, {
    defaultValue: "",
  });
  const [orderDirection, setOrderDirection] = useQueryState(OrderDirectionQuery, { defaultValue: "desc" });
  const [policyType, setPolicyType] = useQueryState(PolicyTypeQuery, {
    defaultValue: " ",
  });
  const [count, setCount] = useState(0);
  const [selectProviderOpen, setSelectProviderOpen] = useState(false);
  const [newPolicyType, setNewPolicyType] = useState<PolicyType | null>(null);
  const [createNewOpen, setCreateNewOpen] = useState(false);

  const [open, setOpen] = useState(false);
  const pageInt = parseInt(page) ?? 1;
  const pageSizeInt = parseInt(pageSize) ?? 11;

  useEffect(() => {
    fetchPolicies();
  }, [page, pageSize, orderBy, orderDirection, policyType]);

  const fetchPolicies = () => {
    setLoading(true);
    dispatch(
      getStoragePolicyList({
        page: pageInt,
        page_size: pageSizeInt,
        order_by: orderBy ?? "",
        order_direction: orderDirection ?? "desc",
        conditions: {
          policy_type: policyType == " " ? "" : policyType,
        },
      }),
    )
      .then((res) => {
        setPolicies(res.policies);
        setPageSize(res.pagination.page_size.toString());
        setCount(res.pagination.total_items ?? 0);
        setLoading(false);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  const onAddNewPolcyWithType = useCallback((type: PolicyType) => {
    setNewPolicyType(type);
    setSelectProviderOpen(false);
    setCreateNewOpen(true);
  }, []);

  const onPolicyTypeChange = useCallback((e: SelectChangeEvent<unknown>) => {
    setPolicyType(e.target.value as PolicyType);
    setPage("1");
  }, []);

  return (
    <PageContainer>
      <SelectProvider
        open={selectProviderOpen}
        onClose={() => setSelectProviderOpen(false)}
        onSelect={onAddNewPolcyWithType}
      />
      {newPolicyType && (
        <AddWizardDialog open={createNewOpen} onClose={() => setCreateNewOpen(false)} type={newPolicyType} />
      )}
      <Container maxWidth="xl">
        <PageHeader title={t("dashboard:nav.storagePolicy")} />
        <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
          <SecondaryButton onClick={fetchPolicies} disabled={loading} variant={"contained"} startIcon={<ArrowSync />}>
            {t("node.refresh")}
          </SecondaryButton>
          <DenseSelect
            value={policyType}
            onChange={onPolicyTypeChange}
            renderValue={(v) => (
              <Typography variant="body2">
                {v == " " ? t("policy.all") : t(PolicyPropsMap[v as PolicyType].name)}
              </Typography>
            )}
          >
            <SquareMenuItem value={" "}>
              <ListItemText
                slotProps={{
                  primary: { variant: "body2" },
                }}
              >
                {t("policy.all")}
              </ListItemText>
            </SquareMenuItem>
            {Object.values(PolicyType).map((type) => (
              <SquareMenuItem key={type} value={type}>
                <ListItemText
                  slotProps={{
                    primary: { variant: "body2" },
                  }}
                >
                  {t(PolicyPropsMap[type].name)}
                </ListItemText>
              </SquareMenuItem>
            ))}
          </DenseSelect>
        </Stack>
        <Grid container spacing={2}>
          <Grid
            size={{
              xs: 12,
              md: 6,
              lg: 4,
            }}
          >
            <BorderedCardClickable
              onClick={() => setSelectProviderOpen(true)}
              sx={{
                height: "100%",
                borderStyle: "dashed",
                display: "flex",
                alignItems: "center",
                gap: 1,
                justifyContent: "center",
                color: (t) => t.palette.text.secondary,
              }}
            >
              <Add />
              <Typography variant="h6">{t("policy.newStoragePolicy")}</Typography>
            </BorderedCardClickable>
          </Grid>
          {!loading && policies.map((p) => <StoragePolicyCard key={p.id} policy={p} onRefresh={fetchPolicies} />)}
          {loading &&
            policies.length > 0 &&
            policies.map((p) => <StoragePolicyCard key={`loading-${p.id}`} loading={true} />)}
          {loading &&
            policies.length === 0 &&
            Array.from(Array(5)).map((_, index) => <StoragePolicyCard key={`loading-placeholder-${index}`} loading={true} />)}
        </Grid>
        {count > 0 && (
          <Box sx={{ mt: 1 }}>
            <TablePagination
              page={pageInt}
              totalItems={count}
              rowsPerPage={pageSizeInt}
              rowsPerPageOptions={[11, 25, 50, 100, 200, 500, 1000]}
              onRowsPerPageChange={(value) => setPageSize(value.toString())}
              onChange={(_, value) => setPage(value.toString())}
            />
          </Box>
        )}
      </Container>
    </PageContainer>
  );
};

export default StoragePolicySetting;
