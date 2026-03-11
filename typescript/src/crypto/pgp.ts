/**
 * Crypto Module - PGP Operations
 * 
 * Handles OpenPGP key generation, signing, encryption, and verification.
 */

import * as openpgp from 'openpgp';
import type {
  Fingerprint,
  AccountOptions,
  CertificateInfo,
} from '../types/index.js';
import { MauError } from '../types/index.js';

/** Key pair with public and private keys */
export interface KeyPair {
  publicKey: openpgp.PublicKey;
  privateKey: openpgp.PrivateKey;
  fingerprint: Fingerprint;
}

/**
 * Generate a new PGP key pair
 * 
 * @param options Account options including algorithm choice
 * @throws {MauError} If passphrase is missing or RSA key is too weak
 */
export async function generateKeyPair(options: AccountOptions): Promise<KeyPair> {
  if (!options.passphrase) {
    throw new MauError('Passphrase required', 'PASSPHRASE_REQUIRED');
  }

  const keyOptions: openpgp.GenerateKeyOptions = {
    userIDs: [{ name: options.name, email: options.email }],
    passphrase: options.passphrase,
  };
  
  // Set key expiration (default: 2 years)
  const expirationYears = options.expirationYears ?? 2;
  if (expirationYears > 0) {
    // Convert years to seconds
    keyOptions.keyExpirationTime = expirationYears * 365 * 24 * 60 * 60;
  }

  // Use Ed25519 by default (modern, fast, secure), RSA if specified
  if (options.algorithm === 'rsa') {
    keyOptions.type = 'rsa';
    const rsaBits = options.rsaBits || 4096;
    
    // Validate key strength (2048 bits minimum per NIST recommendations)
    if (rsaBits < 2048) {
      throw new MauError(
        'RSA keys must be at least 2048 bits for security. Recommended: 4096 bits.',
        'WEAK_KEY'
      );
    }
    
    keyOptions.rsaBits = rsaBits;
  } else {
    keyOptions.type = 'ecc';
    keyOptions.curve = 'ed25519';
  }

  const result = await openpgp.generateKey({
    ...keyOptions,
    format: 'armored',
  });
  
  // Keys are returned as armored strings
  const privateKeyArmored = result.privateKey;
  const publicKeyArmored = result.publicKey;

  // Read keys
  let privKey = await openpgp.readPrivateKey({ armoredKey: privateKeyArmored });
  const pubKey = await openpgp.readKey({ armoredKey: publicKeyArmored });

  // Generated keys are encrypted with passphrase, decrypt for immediate use
  if (!privKey.isDecrypted()) {
    privKey = await openpgp.decryptKey({
      privateKey: privKey,
      passphrase: options.passphrase,
    });
  }

  return {
    publicKey: pubKey,
    privateKey: privKey,
    fingerprint: pubKey.getFingerprint(),
  };
}

/**
 * Serialize private key to armored format
 * 
 * @param privateKey The private key to serialize (must be decrypted)
 * @param passphrase Passphrase to encrypt the key with
 * @throws {MauError} If the key is already encrypted
 */
export async function serializePrivateKey(
  privateKey: openpgp.PrivateKey,
  passphrase: string
): Promise<string> {
  // Require decrypted key - prevents accidental passphrase mismatch
  if (!privateKey.isDecrypted()) {
    throw new MauError(
      'Private key must be decrypted before serialization. Use deserializePrivateKey() to decrypt first.',
      'KEY_ALREADY_ENCRYPTED'
    );
  }
  
  // Encrypt the key with the provided passphrase
  const encrypted = await openpgp.encryptKey({
    privateKey,
    passphrase,
  });
  return encrypted.armor();
}

/**
 * Serialize public key to armored format
 */
export function serializePublicKey(publicKey: openpgp.PublicKey): string {
  return publicKey.armor();
}

/**
 * Deserialize private key from armored format
 */
export async function deserializePrivateKey(
  armoredKey: string,
  passphrase: string
): Promise<openpgp.PrivateKey> {
  const privateKey = await openpgp.readPrivateKey({ armoredKey });
  
  if (!privateKey.isDecrypted()) {
    try {
      const decrypted = await openpgp.decryptKey({
        privateKey,
        passphrase,
      });
      return decrypted;
    } catch (err) {
      throw new MauError('Incorrect passphrase', 'INCORRECT_PASSPHRASE');
    }
  }

  return privateKey;
}

/**
 * Deserialize public key from armored format
 */
export async function deserializePublicKey(armoredKey: string): Promise<openpgp.PublicKey> {
  return await openpgp.readKey({ armoredKey });
}

/**
 * Extract fingerprint from public key
 */
export function getFingerprint(key: openpgp.PublicKey | openpgp.PrivateKey): Fingerprint {
  return key.getFingerprint();
}

/**
 * Sign and encrypt data
 */
export async function signAndEncrypt(
  data: Uint8Array | string,
  privateKey: openpgp.PrivateKey,
  publicKeys: openpgp.PublicKey[]
): Promise<string> {
  const message = await openpgp.createMessage({
    binary: typeof data === 'string' ? new TextEncoder().encode(data) : data,
  });

  const encrypted = await openpgp.encrypt({
    message,
    encryptionKeys: publicKeys,
    signingKeys: privateKey,
    format: 'armored',
  });

  return encrypted as string;
}

/**
 * Decrypt and verify data
 */
export async function decryptAndVerify(
  armoredMessage: string,
  privateKey: openpgp.PrivateKey,
  publicKeys: openpgp.PublicKey[]
): Promise<{ data: Uint8Array; verified: boolean; signedBy?: Fingerprint }> {
  const message = await openpgp.readMessage({ armoredMessage });

  const { data, signatures } = await openpgp.decrypt({
    message,
    decryptionKeys: privateKey,
    verificationKeys: publicKeys,
    format: 'binary',
  });

  let verified = false;
  let signedBy: Fingerprint | undefined;

  if (signatures && signatures.length > 0) {
    try {
      await signatures[0].verified;
      verified = true;
      signedBy = signatures[0].keyID.toHex();
    } catch {
      verified = false;
    }
  }

  return {
    data: data as Uint8Array,
    verified,
    signedBy,
  };
}

/**
 * Sign data (detached signature)
 */
export async function sign(data: Uint8Array, privateKey: openpgp.PrivateKey): Promise<string> {
  const message = await openpgp.createMessage({ binary: data });
  const signature = await openpgp.sign({
    message,
    signingKeys: privateKey,
    detached: true,
    format: 'armored',
  });
  return signature as string;
}

/**
 * Verify detached signature
 */
export async function verify(
  data: Uint8Array,
  signature: string,
  publicKey: openpgp.PublicKey
): Promise<boolean> {
  const message = await openpgp.createMessage({ binary: data });
  const sig = await openpgp.readSignature({ armoredSignature: signature });

  const verificationResult = await openpgp.verify({
    message,
    signature: sig,
    verificationKeys: publicKey,
  });

  try {
    await verificationResult.signatures[0].verified;
    return true;
  } catch {
    return false;
  }
}

/**
 * Compute SHA-256 checksum
 */
export async function sha256(data: Uint8Array): Promise<string> {
  if (typeof crypto !== 'undefined' && crypto.subtle) {
    // Browser or modern Node.js with Web Crypto API
    const hashBuffer = await crypto.subtle.digest('SHA-256', data as BufferSource);
    return Array.from(new Uint8Array(hashBuffer))
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('');
  } else {
    // Fallback for older Node.js
    const cryptoNode = await import('crypto');
    const hash = cryptoNode.createHash('sha256');
    hash.update(Buffer.from(data));
    return hash.digest('hex');
  }
}

/**
 * Generate self-signed TLS certificate with embedded PGP fingerprint
 * Note: This is a simplified version - full implementation requires X.509 library
 */
export async function generateCertificate(
  privateKey: openpgp.PrivateKey,
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  _dnsNames: string[] = []
): Promise<CertificateInfo> {
  // This would require a full X.509 implementation
  // For now, return a placeholder that should be implemented with a proper crypto library
  throw new MauError(
    'Certificate generation not yet implemented in TypeScript - use Node.js crypto or Web Crypto API with X.509 library',
    'NOT_IMPLEMENTED'
  );
}
