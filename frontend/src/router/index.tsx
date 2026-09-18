import { createBrowserRouter, Navigate } from "react-router-dom";
import { App } from "../App.tsx";
import ErrorBoundary from "../component/Common/ErrorBoundary.tsx";
import HeadlessFrame from "../component/Frame/HeadlessFrame.tsx";
import { HomeRedirect } from "../component/Pages/HomeRedirect.tsx";
import Activate from "../component/Pages/Login/Activate.tsx";
import Authorize from "../component/Pages/Login/Authorize.tsx";
import Reset from "../component/Pages/Login/Reset.tsx";
import SessionIntro from "../component/Pages/Login/SessionIntro.tsx";
import DesktopCallback from "../component/Pages/Login/Signin/DesktopCallback.tsx";
import MobileCallback from "../component/Pages/Login/Signin/MobileCallback.tsx";
import SignIn from "../component/Pages/Login/Signin/SignIn.tsx";
import SignUp from "../component/Pages/Login/Signup.tsx";
import NoMatch from "../component/Pages/NoMatch.tsx";

export const router = createBrowserRouter([
  {
    path: "/",
    errorElement: <ErrorBoundary />,
    element: <App />,
    children: [
      { path: "/", element: <HomeRedirect /> },
      {
        path: "/",
        element: <HeadlessFrame />,
        children: [
          {
            path: "/session",
            element: <SessionIntro />,
            children: [
              {
                path: "/session",
                element: <SignIn />,
              },
              {
                path: "signup",
                element: <SignUp />,
              },
              {
                path: "activate",
                element: <Activate />,
              },
              {
                path: "reset",
                element: <Reset />,
              },
              {
                path: "authorize",
                element: <Authorize />,
              },
            ],
          },
          {
            path: "*",
            element: <NoMatch />,
          },
        ],
      },
      {
        path: "/callback",
        element: <HeadlessFrame />,
        children: [
          {
            path: "desktop",
            element: <DesktopCallback />,
          },
          {
            path: "ios",
            element: <MobileCallback />,
          },
        ],
      },
      {
        path: "/",
        async lazy() {
          let { AutoNavbarFrame } = await import("../component/Frame/FrameManagerBundle.tsx");
          return { Component: AutoNavbarFrame };
        },
        children: [
          {
            path: "/admin",
            children: [
              {
                path: "/admin",
                element: <Navigate to={"/admin/home"} />,
              },
              {
                path: "home",
                async lazy() {
                  let { Home } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: Home };
                },
              },
              {
                path: "settings",
                async lazy() {
                  let { Settings } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: Settings };
                },
              },

              {
                path: "policy",
                async lazy() {
                  let { StoragePolicySetting } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: StoragePolicySetting };
                },
              },
              {
                path: "policy/:id",
                async lazy() {
                  let { EditStoragePolicy } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: EditStoragePolicy };
                },
              },
              {
                path: "policy/oauth",
                async lazy() {
                  let { OauthCallback } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: OauthCallback };
                },
              },
              {
                path: "oauth",
                async lazy() {
                  let { OAuthClientSetting } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: OAuthClientSetting };
                },
              },
              {
                path: "oauth/new",
                async lazy() {
                  let { EditOAuthClient } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: EditOAuthClient };
                },
              },
              {
                path: "oauth/:id",
                async lazy() {
                  let { EditOAuthClient } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: EditOAuthClient };
                },
              },
              {
                path: "group",
                async lazy() {
                  let { GroupSetting } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: GroupSetting };
                },
              },
              {
                path: "group/:id",
                async lazy() {
                  let { EditGroup } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: EditGroup };
                },
              },
              {
                path: "node",
                async lazy() {
                  let { NodeSetting } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: NodeSetting };
                },
              },
              {
                path: "node/:id",
                async lazy() {
                  let { EditNode } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: EditNode };
                },
              },
              {
                path: "user",
                async lazy() {
                  let { UserSetting } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: UserSetting };
                },
              },
              {
                path: "file",
                async lazy() {
                  let { FileSetting } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: FileSetting };
                },
              },
              {
                path: "blob",
                async lazy() {
                  let { EntitySetting } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: EntitySetting };
                },
              },
              {
                path: "task",
                async lazy() {
                  let { TaskList } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: TaskList };
                },
              },
              {
                path: "share",
                async lazy() {
                  let { ShareList } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: ShareList };
                },
              },
              {
                path: "filesystem",
                async lazy() {
                  let { FileSystem } = await import("../component/Admin/AdminBundle.tsx");
                  return { Component: FileSystem };
                },
              },
            ],
          },
          {
            path: "/home",
            async lazy() {
              let { FileManager } = await import("../component/Frame/FrameManagerBundle.tsx");
              return { Component: FileManager };
            },
          },
          {
            path: "/tasks",
            async lazy() {
              let { TaskList } = await import("../component/Pages/Pages");
              return { Component: TaskList };
            },
          },
          {
            path: "/downloads",
            async lazy() {
              let { DownloadList } = await import("../component/Pages/Pages");
              return { Component: DownloadList };
            },
          },
          {
            path: "/shares",
            async lazy() {
              let { ShareList } = await import("../component/Pages/Pages");
              return { Component: ShareList };
            },
          },
          {
            path: "/connect",
            async lazy() {
              let { Devices } = await import("../component/Pages/Pages");
              return { Component: Devices };
            },
          },
          {
            path: "/settings",
            async lazy() {
              let { Setting } = await import("../component/Pages/Pages");
              return { Component: Setting };
            },
          },
          {
            path: "/profile/:id",
            async lazy() {
              let { Profile } = await import("../component/Pages/Pages");
              return { Component: Profile };
            },
          },
        ],
      },
    ],
  },
]);
