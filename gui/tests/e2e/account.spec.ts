import { test, expect } from '@playwright/test';
import { cleanupTestData } from '../helpers';

test.describe('Account Management', () => {
  test.beforeEach(async () => {
    await cleanupTestData();
  });

  test.afterEach(async () => {
    await cleanupTestData();
  });

  test('create new account successfully', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Verify on Create Account tab
    const createTab = page.getByRole('tab', { name: 'Create Account' });
    await expect(createTab).toBeVisible();
    await createTab.click();

    // Fill in account creation form
    await page.getByLabel('Name').fill('Test User');
    await page.getByLabel('Email').fill('test@example.com');
    await page.getByLabel('Passphrase').fill('secure-passphrase-123');

    // Submit form
    await page.getByRole('button', { name: 'Create Account' }).click();

    // Wait for success message
    await expect(page.getByText('Account created successfully!')).toBeVisible({ timeout: 5000 });

    // Verify redirected to main app (composer should be visible)
    await expect(page.getByPlaceholder("What's on your mind?")).toBeVisible({ timeout: 5000 });
  });

  test('show validation errors for invalid account creation', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const createTab = page.getByRole('tab', { name: 'Create Account' });
    await createTab.click();

    // Try to submit with empty fields
    await page.getByRole('button', { name: 'Create Account' }).click();

    // Verify validation messages
    await expect(page.getByText(/Name must be at least 2 characters/)).toBeVisible();

    // Fill with invalid email
    await page.getByLabel('Name').fill('Test User');
    await page.getByLabel('Email').fill('invalid-email');
    await page.getByLabel('Passphrase').fill('short');

    await page.getByRole('button', { name: 'Create Account' }).click();

    await expect(page.getByText('Invalid email')).toBeVisible();
    await expect(page.getByText('At least 12 characters')).toBeVisible();
  });
});
