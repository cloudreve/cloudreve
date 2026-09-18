/**
 * Generate a cryptographically random string for use as code verifier
 * @param length - Length of the random string (default: 64)
 * @returns Random string suitable for code verifier
 */
export function randomCryptoString(length: number = 64): string {
  const charset =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~";
  const randomValues = new Uint8Array(length);
  crypto.getRandomValues(randomValues);

  let result = "";
  for (let i = 0; i < length; i++) {
    result += charset[randomValues[i] % charset.length];
  }
  return result;
}

/**
 * Generate code challenge from code verifier using SHA-256
 * @param codeVerifier - The code verifier string
 * @returns Base64 URL-encoded SHA-256 hash of the code verifier
 */
export async function generateCodeChallenge(
  codeVerifier: string
): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(codeVerifier);
  const hashBuffer = await crypto.subtle.digest("SHA-256", data);
  const hashArray = new Uint8Array(hashBuffer);

  // Convert to base64url encoding
  let base64 = "";
  for (let i = 0; i < hashArray.length; i++) {
    base64 += String.fromCharCode(hashArray[i]);
  }
  return btoa(base64)
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=+$/, "");
}

/**
 * Generate both code verifier and code challenge
 * @returns Object containing both code verifier and code challenge
 */
export async function generatePKCEPair(): Promise<{
  codeVerifier: string;
  codeChallenge: string;
}> {
  const codeVerifier = randomCryptoString();
  const codeChallenge = await generateCodeChallenge(codeVerifier);
  return { codeVerifier, codeChallenge };
}
