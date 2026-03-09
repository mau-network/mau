/**
 * Types Module Tests - Error classes and constants
 */

import { describe, it, expect } from '@jest/globals';
import {
  MauError,
  PassphraseRequiredError,
  IncorrectPassphraseError,
  NoIdentityError,
  AccountAlreadyExistsError,
  InvalidFileNameError,
  FriendNotFollowedError,
  PeerNotFoundError,
  IncorrectPeerCertificateError,
  MAU_DIR_NAME,
  ACCOUNT_KEY_FILENAME,
  HTTP_TIMEOUT_MS,
  URI_PROTOCOL_NAME,
} from './index';

describe('Types Module', () => {
  describe('Error Classes', () => {
    it('should create PassphraseRequiredError', () => {
      const error = new PassphraseRequiredError();
      expect(error.code).toBe('PASSPHRASE_REQUIRED');
    });

    it('should create IncorrectPassphraseError', () => {
      const error = new IncorrectPassphraseError();
      expect(error.code).toBe('INCORRECT_PASSPHRASE');
    });

    it('should create NoIdentityError', () => {
      const error = new NoIdentityError();
      expect(error.code).toBe('NO_IDENTITY');
    });

    it('should create AccountAlreadyExistsError', () => {
      const error = new AccountAlreadyExistsError();
      expect(error.code).toBe('ACCOUNT_ALREADY_EXISTS');
    });

    it('should create InvalidFileNameError', () => {
      const error = new InvalidFileNameError('test');
      expect(error.code).toBe('INVALID_FILE_NAME');
    });

    it('should create FriendNotFollowedError', () => {
      const error = new FriendNotFollowedError();
      expect(error.code).toBe('FRIEND_NOT_FOLLOWED');
    });

    it('should create PeerNotFoundError', () => {
      const error = new PeerNotFoundError();
      expect(error.code).toBe('PEER_NOT_FOUND');
    });

    it('should create IncorrectPeerCertificateError', () => {
      const error = new IncorrectPeerCertificateError();
      expect(error.code).toBe('INCORRECT_PEER_CERTIFICATE');
    });
  });

  describe('Constants', () => {
    it('should define MAU_DIR_NAME', () => {
      expect(MAU_DIR_NAME).toBe('.mau');
    });

    it('should define ACCOUNT_KEY_FILENAME', () => {
      expect(ACCOUNT_KEY_FILENAME).toBe('account.pgp');
    });

    it('should define HTTP_TIMEOUT_MS', () => {
      expect(HTTP_TIMEOUT_MS).toBe(30000);
    });

    it('should define URI_PROTOCOL_NAME', () => {
      expect(URI_PROTOCOL_NAME).toBe('mau');
    });
  });
});
