import { Icon as Iconify } from "@iconify/react";
import { Box, SvgIconProps, useTheme } from "@mui/material";
import SvgIcon from "@mui/material/SvgIcon/SvgIcon";
import { memo, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation, useNavigate } from "react-router-dom";
import { GroupPermission, isAnyAdmin } from "../../../api/user.ts";
import { useAppSelector } from "../../../redux/hooks.ts";
import SessionManager from "../../../session";
import { GroupBS } from "../../../session/utils.ts";
import BoxMultiple from "../../Icons/BoxMultiple.tsx";
import BoxMultipleFilled from "../../Icons/BoxMultipleFilled.tsx";
import BuildingShop from "../../Icons/BuildingShop.tsx";
import BuildingShopFilled from "../../Icons/BuildingShopFilled.tsx";
import CloudDownload from "../../Icons/CloudDownload.tsx";
import CloudDownloadOutlined from "../../Icons/CloudDownloadOutlined.tsx";
import CubeSync from "../../Icons/CubeSync.tsx";
import CubeSyncFilled from "../../Icons/CubeSyncFilled.tsx";
import CubeTree from "../../Icons/CubeTree.tsx";
import CubeTreeFilled from "../../Icons/CubeTreeFilled.tsx";
import DataHistogram from "../../Icons/DataHistogram.tsx";
import DataHistogramFilled from "../../Icons/DataHistogramFilled.tsx";
import Folder from "../../Icons/Folder.tsx";
import FolderOutlined from "../../Icons/FolderOutlined.tsx";
import Globe from "../../Icons/Globe.tsx";
import GlobeFilled from "../../Icons/GlobeFilled.tsx";
import HomeOutlined from "../../Icons/HomeOutlined.tsx";
import Payment from "../../Icons/Payment.tsx";
import PaymentFilled from "../../Icons/PaymentFilled.tsx";
import People from "../../Icons/People.tsx";
import PeopleFilled from "../../Icons/PeopleFilled.tsx";
import Person from "../../Icons/Person.tsx";
import PersonOutlined from "../../Icons/PersonOutlined.tsx";
import PhoneLaptop from "../../Icons/PhoneLaptop.tsx";
import PhoneLaptopOutlined from "../../Icons/PhoneLaptopOutlined.tsx";
import SendLogging from "../../Icons/SendLogging.tsx";
import SendLoggingFilled from "../../Icons/SendLoggingFilled.tsx";
import Server from "../../Icons/Server.tsx";
import ServerFilled from "../../Icons/ServerFilled.tsx";
import Setting from "../../Icons/Setting.tsx";
import SettingsOutlined from "../../Icons/SettingsOutlined.tsx";
import ShareAndroid from "../../Icons/ShareAndroid.tsx";
import ShareOutlined from "../../Icons/ShareOutlined.tsx";
import ShieldLock from "../../Icons/ShieldLock.tsx";
import ShieldLockFilled from "../../Icons/ShieldLockFilled.tsx";
import Storage from "../../Icons/Storage.tsx";
import StorageOutlined from "../../Icons/StorageOutlined.tsx";
import Warning from "../../Icons/Warning.tsx";
import WarningOutlined from "../../Icons/WarningOutlined.tsx";
import WrenchSettings from "../../Icons/WrenchSettings.tsx";
import NavIconTransition from "./NavIconTransition.tsx";
import SideNavItem from "./SideNavItem.tsx";

export interface NavigationItem {
  label: string;
  icon?: ((props: SvgIconProps) => JSX.Element)[] | (typeof SvgIcon)[];
  iconifyName?: string;
  path: string;
  // Delegated admin section bit required to see this item. Full admins
  // (is_admin) always pass; undefined means any admin.
  permission?: number;
}

let NavigationItems: NavigationItem[];
NavigationItems = [
  {
    label: "navbar.myShare",
    icon: [ShareAndroid, ShareOutlined],
    path: "/shares",
  },
];

const ConnectNavigationItem: NavigationItem = {
  label: "navbar.connect",
  icon: [PhoneLaptop, PhoneLaptopOutlined],
  path: "/connect",
};

const TaskNavigationItem: NavigationItem = {
  label: "navbar.taskQueue",
  icon: [CubeSyncFilled, CubeSync],
  path: "/tasks",
};

const RemoteDownloadNavigationItem: NavigationItem = {
  label: "navbar.remoteDownload",
  icon: [CloudDownload, CloudDownloadOutlined],
  path: "/downloads",
};

const ShopNavigationItem: NavigationItem = {
  label: "navbar.shop",
  icon: [BuildingShopFilled, BuildingShop],
  path: "/shop",
};

// Public share directory — rendered for anonymous visitors too.
const DiscoverNavigationItem: NavigationItem = {
  label: "navbar.discover",
  icon: [GlobeFilled, Globe],
  path: "/discover",
};

export const SideNavItemComponent = ({ item }: { item: NavigationItem }) => {
  const { t } = useTranslation("application");
  const navigate = useNavigate();
  const location = useLocation();
  const theme = useTheme();
  const active = useMemo(() => {
    return location.pathname == item.path || location.pathname.startsWith(item.path + "/");
  }, [location.pathname, item.path]);
  return (
    <>
      <SideNavItem
        key={item.label}
        onClick={() => (item.iconifyName ? window.open(item.path, "_blank") : navigate(item.path))}
        label={t(item.label)}
        active={active}
        icon={
          !item.icon ? (
            <Box
              sx={{
                width: 20,
                height: 20,
              }}
            >
              <Iconify
                icon={item.iconifyName ?? ""}
                height={20}
                style={{
                  color: theme.palette.action.active,
                }}
              />
            </Box>
          ) : (
            <NavIconTransition
              sx={{ px: 0, py: 0, pr: "14px", height: "20px" }}
              iconProps={{ fontSize: "small", color: "action" }}
              fileIcon={item.icon}
              active={active}
            />
          )
        }
      />
    </>
  );
};

let AdminNavigationItems: NavigationItem[];
AdminNavigationItems = [
  {
    label: "dashboard:nav.summary",
    icon: [DataHistogramFilled, DataHistogram],
    path: "/admin/home",
  },
  {
    label: "dashboard:nav.settings",
    icon: [Setting, SettingsOutlined],
    path: "/admin/settings",
    permission: GroupPermission.admin_settings,
  },
  {
    label: "dashboard:nav.fileSystem",
    icon: [CubeTreeFilled, CubeTree],
    path: "/admin/filesystem",
    permission: GroupPermission.admin_settings,
  },
  {
    label: "dashboard:nav.storagePolicy",
    icon: [Storage, StorageOutlined],
    path: "/admin/policy",
    permission: GroupPermission.admin_storage,
  },
  {
    label: "dashboard:nav.nodes",
    icon: [ServerFilled, Server],
    path: "/admin/node",
    permission: GroupPermission.admin_storage,
  },
  {
    label: "dashboard:nav.groups",
    icon: [PeopleFilled, People],
    path: "/admin/group",
    permission: GroupPermission.admin_groups,
  },
  {
    label: "dashboard:nav.users",
    icon: [Person, PersonOutlined],
    path: "/admin/user",
    permission: GroupPermission.admin_users,
  },
  {
    label: "dashboard:nav.files",
    icon: [Folder, FolderOutlined],
    path: "/admin/file",
    permission: GroupPermission.admin_files,
  },
  {
    label: "dashboard:nav.entities",
    icon: [BoxMultipleFilled, BoxMultiple],
    path: "/admin/blob",
    permission: GroupPermission.admin_files,
  },
  {
    label: "dashboard:nav.shares",
    icon: [ShareAndroid, ShareOutlined],
    path: "/admin/share",
    permission: GroupPermission.admin_shares,
  },
  {
    label: "dashboard:nav.tasks",
    icon: [CubeSyncFilled, CubeSync],
    path: "/admin/task",
    permission: GroupPermission.admin_queue,
  },
  {
    label: "dashboard:vas.orders",
    icon: [PaymentFilled, Payment],
    path: "/admin/payment",
    permission: GroupPermission.admin_payment,
  },
  {
    label: "dashboard:nav.events",
    icon: [SendLoggingFilled, SendLogging],
    path: "/admin/event",
    permission: GroupPermission.admin_events,
  },
  {
    label: "dashboard:nav.abuseReport",
    icon: [Warning, WarningOutlined],
    path: "/admin/abuse",
    permission: GroupPermission.admin_reports,
  },
  {
    label: "dashboard:nav.oauthClients",
    icon: [ShieldLockFilled, ShieldLock],
    path: "/admin/oauth",
    permission: GroupPermission.admin_settings,
  },
];

export const AdminPageNavigation = memo(() => {
  const user = SessionManager.currentLoginOrNull();
  const permission = useMemo(() => GroupBS(user?.user), [user?.user?.group?.permission]);
  const isFullAdmin = permission.enabled(GroupPermission.is_admin);
  const visibleItems = AdminNavigationItems.filter(
    (item) => item.permission === undefined || isFullAdmin || permission.enabled(item.permission),
  );
  return (
    <>
      <SideNavItemComponent key={visibleItems[0].label} item={visibleItems[0]} />
      <Box>
        {visibleItems.slice(1).map((item) => (
          <SideNavItemComponent key={item.label} item={item} />
        ))}
      </Box>
      <SideNavItemComponent
        item={{
          label: "navbar.backToHomepage",
          icon: [HomeOutlined, HomeOutlined],
          path: "/home",
        }}
      />
    </>
  );
});

const PageNavigation = () => {
  const appPromotionEnabled = useAppSelector((state) => state.siteConfig.basic.config.app_promotion);
  const user = SessionManager.currentLoginOrNull();
  const isAdmin = useMemo(() => {
    return isAnyAdmin(GroupBS(user?.user));
  }, [user?.user?.group?.permission]);
  const remoteDownloadEnabled = useMemo(() => {
    return GroupBS(user?.user).enabled(GroupPermission.remote_download);
  }, [user?.user?.group?.permission]);
  const connectEnabled = useMemo(() => {
    return GroupBS(user?.user).enabled(GroupPermission.webdav) || appPromotionEnabled;
  }, [user?.user?.group?.permission, appPromotionEnabled]);
  const isLogin = !!user;
  const customNavItems = useAppSelector((state) => state.siteConfig.basic.config.custom_nav_items);

  return (
    <>
      <SideNavItemComponent item={DiscoverNavigationItem} />
      {isLogin && (
        <Box>
          <>
            {NavigationItems.map((item) => (
              <SideNavItemComponent key={item.label} item={item} />
            ))}
            {connectEnabled && <SideNavItemComponent item={ConnectNavigationItem} />}
            <SideNavItemComponent item={TaskNavigationItem} />
            {remoteDownloadEnabled && <SideNavItemComponent item={RemoteDownloadNavigationItem} />}
            <SideNavItemComponent item={ShopNavigationItem} />
          </>
        </Box>
      )}
      {customNavItems && customNavItems.length > 0 && (
        <Box>
          {customNavItems
            .filter((item) => isLogin || item.scope !== "user")
            .map((item) => (
              <SideNavItemComponent
                key={item.name}
                item={{
                  label: item.name,
                  iconifyName: item.icon,
                  path: item.url,
                }}
              />
            ))}
        </Box>
      )}
      {isLogin && isAdmin && (
        <SideNavItemComponent
          item={{
            label: "navbar.dashboard",
            icon: [WrenchSettings, WrenchSettings],
            path: "/admin/home",
          }}
        />
      )}
    </>
  );
};

export default PageNavigation;
