import { test, expect } from '@playwright/test';
import { cleanupTestData, createAndUnlockAccount } from '../helpers';

const TEST_NAME = 'Test User';
const TEST_EMAIL = 'test@example.com';
const TEST_PASSWORD = 'secure-passphrase-123';

test.describe('Friends Management', () => {
  test.beforeEach(async ({ page }) => {
    await cleanupTestData();
    await createAndUnlockAccount(page, TEST_NAME, TEST_EMAIL, TEST_PASSWORD);
  });

  test.afterEach(async () => {
    await cleanupTestData();
  });

  test('should display user public key', async ({ page }) => {
    // Navigate to Friends page
    await page.getByText('Friends').click();
    await expect(page.getByText('My Public Key')).toBeVisible();

    // Check public key is displayed
    const publicKeyText = page.locator('textarea[readonly]');
    await expect(publicKeyText).toBeVisible();
    
    const keyValue = await publicKeyText.inputValue();
    expect(keyValue).toContain('-----BEGIN PGP PUBLIC KEY BLOCK-----');
    expect(keyValue).toContain('-----END PGP PUBLIC KEY BLOCK-----');
  });

  test('should copy public key to clipboard', async ({ page, context }) => {
    // Grant clipboard permissions
    await context.grantPermissions(['clipboard-read', 'clipboard-write']);

    // Navigate to Friends page
    await page.getByText('Friends').click();

    // Click copy button
    await page.getByRole('button', { name: /copy public key/i }).click();

    // Verify success message
    await expect(page.getByText(/copied/i)).toBeVisible({ timeout: 2000 });
  });

  test('should show empty friends list initially', async ({ page }) => {
    // Navigate to Friends page
    await page.getByText('Friends').click();

    // Check empty state
    await expect(page.getByText('Friends (0)')).toBeVisible();
    await expect(page.getByText(/no friends yet/i)).toBeVisible();
  });

  test('should open add friend modal', async ({ page }) => {
    // Navigate to Friends page
    await page.getByText('Friends').click();

    // Click Add Friend button
    await page.getByRole('button', { name: /add friend/i }).first().click();

    // Check modal is visible
    await expect(page.getByRole('dialog')).toBeVisible();
    await expect(page.locator('.ant-modal-title', { hasText: 'Add Friend' })).toBeVisible();
  });

  test('should add friend with pasted public key', async ({ page }) => {
    // Navigate to Friends page
    await page.getByText('Friends').click();

    // Get user's own public key (to simulate adding a friend)
    const publicKeyText = page.locator('textarea[readonly]').first();
    const publicKey = await publicKeyText.inputValue();

    // Open add friend modal
    await page.getByRole('button', { name: /add friend/i }).first().click();

    // Wait for modal and find the textarea inside it
    const modal = page.getByRole('dialog');
    await expect(modal).toBeVisible();
    
    // Find the input textarea (not readonly) within the modal
    const inputArea = modal.locator('textarea').nth(0);
    await inputArea.fill(publicKey);

    // Submit
    await page.getByRole('button', { name: /add friend/i }).last().click();

    // Wait for modal to close
    await expect(modal).not.toBeVisible({ timeout: 5000 });

    // Verify friend appears in list
    await expect(page.getByText('Friends (1)')).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('heading', { name: TEST_NAME })).toBeVisible();
    await expect(page.getByText(TEST_EMAIL)).toBeVisible();
  });

  test('should show error for invalid public key', async ({ page }) => {
    // Navigate to Friends page
    await page.getByText('Friends').click();

    // Open add friend modal
    await page.getByRole('button', { name: /add friend/i }).first().click();

    const modal = page.getByRole('dialog');
    await expect(modal).toBeVisible();

    // Enter invalid key
    const inputArea = modal.locator('textarea').nth(0);
    await inputArea.fill('invalid-key-data');

    // Submit
    await page.getByRole('button', { name: /add friend/i }).last().click();

    // Verify error message
    await expect(page.getByText(/failed/i)).toBeVisible({ timeout: 5000 });
  });

  test('should remove friend', async ({ page }) => {
    // Navigate to Friends page
    await page.getByText('Friends').click();

    // Add a friend first (using own public key)
    const publicKeyText = page.locator('textarea[readonly]').first();
    const publicKey = await publicKeyText.inputValue();
    
    await page.getByRole('button', { name: /add friend/i }).first().click();
    const modal = page.getByRole('dialog');
    await expect(modal).toBeVisible();
    
    const inputArea = modal.locator('textarea').nth(0);
    await inputArea.fill(publicKey);
    await page.getByRole('button', { name: /add friend/i }).last().click();
    await expect(modal).not.toBeVisible({ timeout: 5000 });

    // Verify friend is in list
    await expect(page.getByText('Friends (1)')).toBeVisible({ timeout: 5000 });

    // Remove friend
    await page.getByRole('button', { name: /remove/i }).first().click();

    // Wait for confirmation dialog and click Remove
    await expect(page.locator('.ant-modal-confirm-title', { hasText: 'Remove Friend' })).toBeVisible({ timeout: 2000 });
    await page.getByRole('button', { name: /remove/i }).last().click();

    // Verify friend is removed
    await expect(page.getByText(/no friends yet/i)).toBeVisible({ timeout: 5000 });
  });

  test('should navigate back to feed', async ({ page }) => {
    // Navigate to Friends page
    await page.getByText('Friends').click();
    await expect(page.getByText('My Public Key')).toBeVisible();

    // Navigate back to feed
    await page.getByText('Feed').click();
    await expect(page.getByPlaceholder("What's on your mind?")).toBeVisible();
  });
});
