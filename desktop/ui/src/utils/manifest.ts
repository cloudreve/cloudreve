import { fetch } from "./fetch";

interface ManifestIcon {
  sizes: string;
  src: string;
  type?: string;
}

interface Manifest {
  icons?: ManifestIcon[];
}

/**
 * Check if a string is a valid URL
 */
export function isValidUrl(url: string): boolean {
  try {
    new URL(url);
    return true;
  } catch {
    return false;
  }
}

/**
 * Fetch manifest.json and find the largest icon (preferably 512x512)
 * @param siteUrl - The base URL of the site
 * @returns The URL of the largest icon, or null if not found
 */
export async function fetchSiteIcon(siteUrl: string): Promise<string | null> {
  const manifestUrl = new URL("/manifest.json", siteUrl);
  const response = await fetch(manifestUrl.toString());

  if (!response.ok) {
    throw new Error(`Failed to fetch manifest: ${response.status}`);
  }

  const manifest: Manifest = await response.json();

  if (!manifest.icons || manifest.icons.length === 0) {
    return null;
  }

  // Find the largest icon, preferring 512x512
  let bestIcon: ManifestIcon | null = null;
  let bestSize = 0;

  for (const icon of manifest.icons) {
    // Parse sizes like "512x512" or "64x64 32x32 24x24 16x16"
    const sizeMatches = icon.sizes.match(/(\d+)x(\d+)/g);
    if (sizeMatches) {
      for (const sizeStr of sizeMatches) {
        const [width] = sizeStr.split("x").map(Number);
        if (width > bestSize) {
          bestSize = width;
          bestIcon = icon;
        }
      }
    }
  }

  if (!bestIcon) {
    return null;
  }

  // Resolve the icon URL relative to the site URL
  const iconUrl = new URL(bestIcon.src, siteUrl);
  return iconUrl.toString();
}
