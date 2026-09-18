import { useEffect } from "react";
import { useLocation } from "react-router-dom";
import { useAppSelector } from "../redux/hooks.ts";

const PageTitle = ({ title }: { title?: string }) => {
  const location = useLocation();
  const siteTitle = useAppSelector((state) => state.siteConfig.basic.config.title);

  useEffect(() => {
    const titles: string[] = [];
    if (title) {
      titles.push(title);
    }

    if (siteTitle) {
      titles.push(siteTitle);
    }

    document.title = titles.join(" - ");
  }, [location, title, siteTitle]);

  return null;
};

export default PageTitle;
