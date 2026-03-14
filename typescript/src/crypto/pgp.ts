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

// ---------------------------------------------------------------------------
// Minimal ASN.1 DER helpers (used only by generateCertificate)
// ---------------------------------------------------------------------------

function derLen(len: number): Uint8Array {
  if (len < 0x80) return new Uint8Array([len]);
  const b: number[] = [];
  let n = len;
  while (n > 0) { b.unshift(n & 0xff); n >>>= 8; }
  return new Uint8Array([0x80 | b.length, ...b]);
}

function derTLV(tag: number, val: Uint8Array): Uint8Array {
  const l = derLen(val.length);
  const out = new Uint8Array(1 + l.length + val.length);
  out[0] = tag; out.set(l, 1); out.set(val, 1 + l.length);
  return out;
}

function cat(...parts: Uint8Array[]): Uint8Array {
  const total = parts.reduce((s, p) => s + p.length, 0);
  const out = new Uint8Array(total);
  let off = 0;
  for (const p of parts) { out.set(p, off); off += p.length; }
  return out;
}

function derSeq(...items: Uint8Array[]): Uint8Array { return derTLV(0x30, cat(...items)); }
function derOID(b: number[]): Uint8Array { return derTLV(0x06, new Uint8Array(b)); }
function derUTF8(s: string): Uint8Array { return derTLV(0x0c, new TextEncoder().encode(s)); }
function derOctetStr(d: Uint8Array): Uint8Array { return derTLV(0x04, d); }
function derBitStr(d: Uint8Array): Uint8Array { return derTLV(0x03, cat(new Uint8Array([0x00]), d)); }

function derInt(v: Uint8Array): Uint8Array {
  // Strip leading zeros (keep at least one byte)
  let i = 0;
  while (i < v.length - 1 && v[i] === 0) i++;
  const trimmed = v.slice(i);
  // Prepend 0x00 if the high bit would be interpreted as a sign bit
  const content = (trimmed[0] & 0x80) ? cat(new Uint8Array([0x00]), trimmed) : trimmed;
  return derTLV(0x02, content);
}

function derUTCTime(d: Date): Uint8Array {
  const p = (n: number) => n.toString().padStart(2, '0');
  const s = `${p(d.getUTCFullYear() % 100)}${p(d.getUTCMonth() + 1)}${p(d.getUTCDate())}` +
            `${p(d.getUTCHours())}${p(d.getUTCMinutes())}${p(d.getUTCSeconds())}Z`;
  return derTLV(0x17, new TextEncoder().encode(s));
}

// OID byte sequences (pre-encoded OID value octets)
const OID_ECDSA_SHA256 = [0x2a, 0x86, 0x48, 0xce, 0x3d, 0x04, 0x03, 0x02];
const OID_COMMON_NAME  = [0x55, 0x04, 0x03];
const OID_SAN          = [0x55, 0x1d, 0x11]; // id-ce-subjectAltName
const OID_BC           = [0x55, 0x1d, 0x13]; // id-ce-basicConstraints

function buildX509Name(cn: string): Uint8Array {
  return derSeq(derTLV(0x31, derSeq(derOID(OID_COMMON_NAME), derUTF8(cn))));
}

/** Convert Web Crypto P1363 (r‖s, each 32 bytes) to DER SEQUENCE { INTEGER r, INTEGER s } */
function p1363ToDer(sig: Uint8Array): Uint8Array {
  return derSeq(derInt(sig.slice(0, 32)), derInt(sig.slice(32, 64)));
}

// ---------------------------------------------------------------------------

/**
 * Generate a self-signed ECDSA P-256 TLS certificate with the PGP fingerprint
 * embedded in the Subject CN and as a URI Subject Alternative Name.
 *
 * The returned `key` is a PKCS#8 DER-encoded private key suitable for use
 * with Node.js `tls.createServer` or the Web Crypto API.
 */
export async function generateCertificate(
  privateKey: openpgp.PrivateKey,
  dnsNames: string[] = []
): Promise<CertificateInfo> {
  const pgpFpr = privateKey.getFingerprint();

  // Generate an ephemeral ECDSA P-256 key pair exclusively for TLS
  const tlsKeys = await crypto.subtle.generateKey(
    { name: 'ECDSA', namedCurve: 'P-256' },
    true,
    ['sign', 'verify']
  );

  const spkiDer  = new Uint8Array(await crypto.subtle.exportKey('spki',  tlsKeys.publicKey));
  const pkcs8Der = new Uint8Array(await crypto.subtle.exportKey('pkcs8', tlsKeys.privateKey));

  // Validity: now → +1 year
  const now = new Date();
  const exp = new Date(now);
  exp.setFullYear(exp.getFullYear() + 1);

  // Random positive serial number (16 bytes)
  const serial = crypto.getRandomValues(new Uint8Array(16));
  serial[0] &= 0x7f;

  const sigAlgId = derSeq(derOID(OID_ECDSA_SHA256));
  const name     = buildX509Name(pgpFpr);
  const validity = derSeq(derUTCTime(now), derUTCTime(exp));

  // Extensions
  const exts: Uint8Array[] = [
    // BasicConstraints: CA:false (critical in some validators, empty SEQUENCE = false)
    derSeq(derOID(OID_BC), derOctetStr(derSeq())),
  ];

  // SubjectAltName: caller-supplied DNS names + pgp: URI for the fingerprint
  const sans: Uint8Array[] = dnsNames.map(n =>
    derTLV(0x82, new TextEncoder().encode(n)) // [2] dNSName
  );
  sans.push(derTLV(0x86, new TextEncoder().encode(`pgp:${pgpFpr}`))); // [6] uniformResourceIdentifier
  exts.push(derSeq(derOID(OID_SAN), derOctetStr(derSeq(...sans))));

  // TBSCertificate
  const tbs = derSeq(
    derTLV(0xa0, derTLV(0x02, new Uint8Array([0x02]))), // [0] EXPLICIT version v3 = 2
    derInt(serial),
    sigAlgId,
    name,      // issuer
    validity,
    name,      // subject (same as issuer for self-signed)
    spkiDer,   // SubjectPublicKeyInfo (already DER from exportKey)
    derTLV(0xa3, derSeq(...exts)),                      // [3] EXPLICIT extensions
  );

  // Sign TBSCertificate; Web Crypto returns P1363 format for ECDSA
  const sigP1363 = new Uint8Array(
    await crypto.subtle.sign({ name: 'ECDSA', hash: 'SHA-256' }, tlsKeys.privateKey, tbs)
  );

  // Certificate ::= SEQUENCE { tbs, signatureAlgorithm, signatureValue BIT STRING }
  const certDer = derSeq(tbs, sigAlgId, derBitStr(p1363ToDer(sigP1363)));

  return { cert: certDer, key: pkcs8Der, fingerprint: pgpFpr };
}
