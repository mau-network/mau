import { test, expect } from '@playwright/test';
import { cleanupTestData, createTestAccount } from '../helpers';

test.describe('Account Unlock', () => {
  test.beforeEach(async () => {
    await cleanupTestData();
  });

  test.afterEach(async () => {
    await cleanupTestData();
  });

  test('unlock existing account successfully', async ({ page }) => {
    // Pre-create account
    await createTestAccount(page, 'Test User', 'test@example.com', 'secure-passphrase-123');

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Should default to Unlock tab when account exists
    const unlockTab = page.getByRole('tab', { name: 'Unlock Account' });
    await expect(unlockTab).toBeVisible();
    await unlockTab.click();

    // Verify account info is displayed
    await expect(page.getByText('Account:')).toBeVisible();
    await expect(page.getByText('Test User')).toBeVisible();
    await expect(page.getByText('Email:')).toBeVisible();
    await expect(page.getByText('test@example.com')).toBeVisible();
    await expect(page.getByText('Fingerprint:')).toBeVisible();

    // Enter passphrase
    await page.getByLabel('Passphrase').fill('secure-passphrase-123');

    // Submit form
    await page.getByRole('button', { name: 'Unlock Account' }).click();

    // Wait for success message
    await expect(page.getByText('Account unlocked successfully!')).toBeVisible({ timeout: 5000 });

    // Verify redirected to main app
    await expect(page.getByPlaceholder("What's on your mind?")).toBeVisible({ timeout: 5000 });
  });

  test('show error for incorrect passphrase', async ({ page }) => {
    // Pre-create account
    await createTestAccount(page, 'Test User', 'test@example.com', 'secure-passphrase-123');

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const unlockTab = page.getByRole('tab', { name: 'Unlock Account' });
    await unlockTab.click();

    // Enter wrong passphrase
    await page.getByLabel('Passphrase').fill('wrong-passphrase');
    await page.getByRole('button', { name: 'Unlock Account' }).click();

    // Should show error message
    await expect(page.locator('.ant-message-error')).toBeVisible({ timeout: 5000 });
  });

  test('show empty state when no account exists', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const unlockTab = page.getByRole('tab', { name: 'Unlock Account' });
    await unlockTab.click();

    // Should show no account message
    await expect(page.getByText('No account found. Please create one first.')).toBeVisible();
  });
});
