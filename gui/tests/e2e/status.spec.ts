import { test, expect } from '@playwright/test';
import { cleanupTestData, createAndUnlockAccount } from '../helpers';

test.describe('Status Posting', () => {
  test.beforeEach(async ({ page }) => {
    await cleanupTestData();
    await createAndUnlockAccount(page, 'Test User', 'test@example.com', 'secure-passphrase-123');
  });

  test.afterEach(async () => {
    await cleanupTestData();
  });

  test('post new status successfully', async ({ page }) => {
    // Verify composer is visible
    const composer = page.getByPlaceholder("What's on your mind?");
    await expect(composer).toBeVisible();

    // Enter status content
    const statusContent = 'This is my first status post!';
    await composer.fill(statusContent);

    // Verify character counter
    await expect(page.getByText(`${statusContent.length} / 500`)).toBeVisible();

    // Submit post
    await page.getByRole('button', { name: 'Post' }).click();

    // Verify post appears in timeline
    await expect(page.getByText(statusContent)).toBeVisible({ timeout: 5000 });

    // Verify composer is cleared
    await expect(composer).toHaveValue('');
    await expect(page.getByText('0 / 500')).toBeVisible();
  });

  test('show validation error for empty status', async ({ page }) => {
    const composer = page.getByPlaceholder("What's on your mind?");
    await expect(composer).toBeVisible();

    // Try to submit empty post
    await page.getByRole('button', { name: 'Post' }).click();

    // Should show validation error
    await expect(page.getByText('Status cannot be empty')).toBeVisible();
  });

  test('enforce character limit', async ({ page }) => {
    const composer = page.getByPlaceholder("What's on your mind?");
    await expect(composer).toBeVisible();

    // Create content exceeding 500 characters
    const longContent = 'a'.repeat(600);
    await composer.fill(longContent);

    // Input should be limited to 500 characters
    const value = await composer.inputValue();
    expect(value.length).toBeLessThanOrEqual(500);

    // Character counter should show 500
    await expect(page.getByText(/500 \/ 500/)).toBeVisible();
  });

  test('post multiple statuses', async ({ page }) => {
    const composer = page.getByPlaceholder("What's on your mind?");

    // Post first status
    await composer.fill('First status here');
    await page.getByRole('button', { name: 'Post' }).click();
    
    // Wait for composer to clear
    await expect(composer).toHaveValue('', { timeout: 5000 });
    
    // Verify first post in timeline (not in composer)
    const timeline = page.locator('.ant-list');
    await expect(timeline.getByText('First status here')).toBeVisible({ timeout: 5000 });

    // Post second status
    await composer.fill('Second status here');
    await page.getByRole('button', { name: 'Post' }).click();
    
    // Wait for composer to clear
    await expect(composer).toHaveValue('', { timeout: 5000 });
    
    // Verify second post in timeline
    await expect(timeline.getByText('Second status here')).toBeVisible({ timeout: 5000 });

    // Verify both posts are visible in timeline
    await expect(timeline.getByText('First status here')).toBeVisible();
    await expect(timeline.getByText('Second status here')).toBeVisible();
  });

  test('post with multiline content', async ({ page }) => {
    const composer = page.getByPlaceholder("What's on your mind?");

    const multilineContent = 'Line 1\nLine 2\nLine 3';
    await composer.fill(multilineContent);

    await page.getByRole('button', { name: 'Post' }).click();

    // Verify multiline post appears correctly
    await expect(page.getByText(/Line 1.*Line 2.*Line 3/s)).toBeVisible({ timeout: 5000 });
  });
});
