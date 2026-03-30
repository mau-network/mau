import { Page } from '@playwright/test';
import { rm } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';

/**
 * Clean up test data directories and browser storage
 */
export async function cleanupTestData(): Promise<void> {
  // Clear browser storage
  const testDataPath = join(tmpdir(), 'mau-test-data');
  
  try {
    await rm(testDataPath, { recursive: true, force: true });
  } catch (error) {
    // Ignore errors if directory doesn't exist
    console.log('Cleanup warning:', error);
  }
}

/**
 * Create a test account using browser page context
 */
export async function createTestAccount(
  page: Page,
  name: string,
  email: string,
  passphrase: string
): Promise<void> {
  await page.goto('/');
  await page.waitForLoadState('networkidle');

  const createTab = page.getByRole('tab', { name: 'Create Account' });
  await createTab.click();

  await page.getByLabel('Name').fill(name);
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Passphrase').fill(passphrase);

  await page.getByRole('button', { name: 'Create Account' }).click();
  
  // Wait for success and then navigate away
  await page.getByText('Account created successfully!').waitFor({ timeout: 5000 });
  
  // Clear storage to simulate app reload
  await page.context().clearCookies();
  await page.evaluate(() => localStorage.clear());
}

/**
 * Create and unlock an account within a page context
 */
export async function createAndUnlockAccount(
  page: Page,
  name: string,
  email: string,
  passphrase: string
): Promise<void> {
  await page.goto('/');
  await page.waitForLoadState('networkidle');

  // Create account
  const createTab = page.getByRole('tab', { name: 'Create Account' });
  await createTab.click();

  await page.getByLabel('Name').fill(name);
  await page.getByLabel('Email').fill(email);
  await page.getByLabel('Passphrase').fill(passphrase);

  await page.getByRole('button', { name: 'Create Account' }).click();

  // Wait for authentication to complete
  await page.getByPlaceholder("What's on your mind?").waitFor({ timeout: 5000 });
}

/**
 * Navigate to app and unlock existing account
 */
export async function unlockAccount(page: Page, passphrase: string): Promise<void> {
  await page.goto('/');
  await page.waitForLoadState('networkidle');

  const unlockTab = page.getByRole('tab', { name: 'Unlock Account' });
  await unlockTab.click();

  await page.getByLabel('Passphrase').fill(passphrase);
  await page.getByRole('button', { name: 'Unlock Account' }).click();

  // Wait for authentication to complete
  await page.getByPlaceholder("What's on your mind?").waitFor({ timeout: 5000 });
}

/**
 * Post a status update
 */
export async function postStatus(page: Page, content: string): Promise<void> {
  const composer = page.getByPlaceholder("What's on your mind?");
  await composer.fill(content);
  await page.getByRole('button', { name: 'Post' }).click();
  
  // Wait for post to appear in timeline
  await page.getByText(content).waitFor({ timeout: 5000 });
}
