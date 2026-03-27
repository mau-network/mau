export interface ValidationResult {
  valid: boolean;
  error?: string;
}

export function validateContent(content: string): ValidationResult {
  if (content.length < 1) {
    return { valid: false, error: 'Content must be at least 1 character' };
  }
  if (content.length > 500) {
    return { valid: false, error: 'Content must not exceed 500 characters' };
  }
  return { valid: true };
}

export function validatePassphrase(passphrase: string): ValidationResult {
  if (passphrase.length < 12) {
    return { valid: false, error: 'Passphrase must be at least 12 characters' };
  }
  return { valid: true };
}

export function validateEmail(email: string): ValidationResult {
  if (!email) {
    return { valid: false, error: 'Email is required' };
  }
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!emailRegex.test(email)) {
    return { valid: false, error: 'Invalid email format' };
  }
  return { valid: true };
}

export function validateName(name: string): ValidationResult {
  if (!name || name.length === 0) {
    return { valid: false, error: 'Name is required' };
  }
  if (name.length > 100) {
    return { valid: false, error: 'Name must not exceed 100 characters' };
  }
  return { valid: true };
}
