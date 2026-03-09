/**
 * Crypto Module - Utilities
 */

export * from './pgp.js';

/**
 * Validate file name to prevent path traversal
 */
export function validateFileName(name: string): boolean {
  if (!name || name.length === 0) {
    return false;
  }

  // Check for path separators
  if (name.includes('/') || name.includes('\\') || name.includes('\0')) {
    return false;
  }

  // Check for relative path components
  if (name === '.' || name === '..' || name.startsWith('./') || name.startsWith('../')) {
    return false;
  }

  return true;
}

/**
 * Normalize fingerprint to lowercase hex without separators
 */
export function normalizeFingerprint(fingerprint: string): string {
  return fingerprint.toLowerCase().replace(/[^0-9a-f]/g, '');
}

/**
 * Format fingerprint with spaces for readability
 */
export function formatFingerprint(fingerprint: string): string {
  const normalized = normalizeFingerprint(fingerprint);
  return normalized.match(/.{1,4}/g)?.join(' ') || normalized;
}
