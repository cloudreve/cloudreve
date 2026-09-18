import { useState, useEffect } from "react";
import { version } from "@tauri-apps/plugin-os";

/**
 * Hook to detect if the current OS is Windows 10.
 * Windows 10 version strings start with "10.0" but the build number is below 22000.
 * Windows 11 also reports as "10.0" but with build number >= 22000.
 */
export function useIsWindows10(): boolean | null {
  const [isWindows10, setIsWindows10] = useState<boolean | null>(null);

  useEffect(() => {
    // Windows version format: "10.0.19045" (Win10) or "10.0.22631" (Win11)
    // Windows 11 has build number >= 22000
    const parts = version().split(".");
    if (parts.length >= 3) {
      const buildNumber = parseInt(parts[2], 10);
      // Windows 10 has build numbers below 22000
      setIsWindows10(!isNaN(buildNumber) && buildNumber < 22000);
    } else {
      setIsWindows10(false);
    }
  }, []);

  return isWindows10;
}
