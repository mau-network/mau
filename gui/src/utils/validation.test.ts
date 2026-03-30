import { test, expect, describe, beforeEach } from 'bun:test';
import { validateContent, validatePassphrase, validateEmail, validateName } from './validation';

describe('Validation utilities', () => {
  describe('validateContent', () => {
    test('should accept valid content', () => {
      const result = validateContent('Hello world');
      expect(result.valid).toBe(true);
      expect(result.error).toBeUndefined();
    });

    test('should reject empty content', () => {
      const result = validateContent('');
      expect(result.valid).toBe(false);
      expect(result.error).toContain('at least 1 character');
    });

    test('should reject content over 500 characters', () => {
      const result = validateContent('a'.repeat(501));
      expect(result.valid).toBe(false);
      expect(result.error).toContain('500 characters');
    });

    test('should accept content at exact limits', () => {
      expect(validateContent('a').valid).toBe(true);
      expect(validateContent('a'.repeat(500)).valid).toBe(true);
    });
  });

  describe('validatePassphrase', () => {
    test('should accept valid passphrase', () => {
      const result = validatePassphrase('SecurePass123!');
      expect(result.valid).toBe(true);
    });

    test('should reject short passphrase', () => {
      const result = validatePassphrase('short');
      expect(result.valid).toBe(false);
      expect(result.error).toContain('12 characters');
    });

    test('should accept passphrase at minimum length', () => {
      const result = validatePassphrase('12characters');
      expect(result.valid).toBe(true);
    });
  });

  describe('validateEmail', () => {
    test('should accept valid email', () => {
      const result = validateEmail('user@example.com');
      expect(result.valid).toBe(true);
    });

    test('should reject invalid email format', () => {
      const result = validateEmail('notanemail');
      expect(result.valid).toBe(false);
    });

    test('should reject empty email', () => {
      const result = validateEmail('');
      expect(result.valid).toBe(false);
    });
  });

  describe('validateName', () => {
    test('should accept valid name', () => {
      const result = validateName('John Doe');
      expect(result.valid).toBe(true);
    });

    test('should reject empty name', () => {
      const result = validateName('');
      expect(result.valid).toBe(false);
    });

    test('should reject overly long name', () => {
      const result = validateName('a'.repeat(101));
      expect(result.valid).toBe(false);
    });
  });
});
