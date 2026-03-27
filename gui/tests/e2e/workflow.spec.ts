import { test, expect } from '@playwright/test';
import { cleanupTestData } from '../helpers';

test.describe('Complete User Workflows', () => {
  test.beforeEach(async () => {
    await cleanupTestData();
  });

  test.afterEach(async () => {
    await cleanupTestData();
  });

  test('full user flow: create account → post status → view timeline', async ({ page }) => {
    // Navigate to app
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Step 1: Create account
    const createTab = page.getByRole('tab', { name: 'Create Account' });
    await createTab.click();

    await page.getByLabel('Name').fill('Alice Smith');
    await page.getByLabel('Email').fill('alice@example.com');
    await page.getByLabel('Passphrase').fill('my-secure-passphrase-2024');

    await page.getByRole('button', { name: 'Create Account' }).click();

    // Wait for success and redirect to timeline
    await expect(page.getByText('Account created successfully!')).toBeVisible({ timeout: 5000 });
    await expect(page.getByPlaceholder("What's on your mind?")).toBeVisible({ timeout: 5000 });

    // Step 2: Post first status
    const composer = page.getByPlaceholder("What's on your mind?");
    await composer.fill('Hello, Mau! This is my first post.');
    await page.getByRole('button', { name: 'Post' }).click();

    // Verify post appears in timeline
    await expect(page.getByText('Hello, Mau! This is my first post.')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Posted!')).toBeVisible();

    // Step 3: Post second status
    await composer.fill('Just posted my second update!');
    await page.getByRole('button', { name: 'Post' }).click();

    // Verify both posts visible
    await expect(page.getByText('Just posted my second update!')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Hello, Mau! This is my first post.')).toBeVisible();

    // Step 4: Verify timeline order (newest first)
    const timeline = page.locator('.ant-list');
    const posts = timeline.locator('.ant-card');
    
    await expect(posts).toHaveCount(2);
    
    // First card should contain second post (newest)
    await expect(posts.nth(0)).toContainText('Just posted my second update!');
    // Second card should contain first post (oldest)
    await expect(posts.nth(1)).toContainText('Hello, Mau! This is my first post.');
  });

  test('account unlock flow with existing account', async ({ page }) => {
    // Step 1: Create account first
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const createTab = page.getByRole('tab', { name: 'Create Account' });
    await createTab.click();

    await page.getByLabel('Name').fill('Bob Johnson');
    await page.getByLabel('Email').fill('bob@example.com');
    await page.getByLabel('Passphrase').fill('bobs-secure-passphrase');

    await page.getByRole('button', { name: 'Create Account' }).click();
    await expect(page.getByPlaceholder("What's on your mind?")).toBeVisible({ timeout: 5000 });

    // Post a status to verify persistence
    await page.getByPlaceholder("What's on your mind?").fill('My persistent post');
    await page.getByRole('button', { name: 'Post' }).click();
    await expect(page.getByText('My persistent post')).toBeVisible({ timeout: 5000 });

    // Step 2: Reload page to simulate app restart
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Step 3: Verify Unlock tab is default when account exists
    const unlockTab = page.getByRole('tab', { name: 'Unlock Account' });
    await expect(unlockTab).toHaveAttribute('aria-selected', 'true');

    // Verify account info displayed
    await expect(page.getByText('Bob Johnson')).toBeVisible();
    await expect(page.getByText('bob@example.com')).toBeVisible();
    await expect(page.getByText('Fingerprint:')).toBeVisible();

    // Step 4: Unlock with correct passphrase
    await page.getByLabel('Passphrase').fill('bobs-secure-passphrase');
    await page.getByRole('button', { name: 'Unlock Account' }).click();

    // Verify successful unlock and data persistence
    await expect(page.getByText('Account unlocked successfully!')).toBeVisible({ timeout: 5000 });
    await expect(page.getByPlaceholder("What's on your mind?")).toBeVisible({ timeout: 5000 });
    
    // Verify previous post still exists
    await expect(page.getByText('My persistent post')).toBeVisible({ timeout: 5000 });
  });

  test('error scenarios: wrong passphrase', async ({ page }) => {
    // Create account
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const createTab = page.getByRole('tab', { name: 'Create Account' });
    await createTab.click();

    await page.getByLabel('Name').fill('Carol Williams');
    await page.getByLabel('Email').fill('carol@example.com');
    await page.getByLabel('Passphrase').fill('correct-passphrase-123');
    await page.getByRole('button', { name: 'Create Account' }).click();
    await expect(page.getByPlaceholder("What's on your mind?")).toBeVisible({ timeout: 5000 });

    // Reload to lock account
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Try wrong passphrase
    const unlockTab = page.getByRole('tab', { name: 'Unlock Account' });
    await unlockTab.click();

    await page.getByLabel('Passphrase').fill('wrong-passphrase');
    await page.getByRole('button', { name: 'Unlock Account' }).click();

    // Verify error message
    await expect(page.locator('.ant-message-error')).toBeVisible({ timeout: 5000 });
    
    // Verify still on unlock screen (not authenticated)
    await expect(page.getByRole('button', { name: 'Unlock Account' })).toBeVisible();
  });

  test('multi-post timeline rendering', async ({ page }) => {
    // Create account
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const createTab = page.getByRole('tab', { name: 'Create Account' });
    await createTab.click();

    await page.getByLabel('Name').fill('Dave Martinez');
    await page.getByLabel('Email').fill('dave@example.com');
    await page.getByLabel('Passphrase').fill('daves-passphrase');
    await page.getByRole('button', { name: 'Create Account' }).click();
    await expect(page.getByPlaceholder("What's on your mind?")).toBeVisible({ timeout: 5000 });

    // Post 5 status updates
    const composer = page.getByPlaceholder("What's on your mind?");
    const postButton = page.getByRole('button', { name: 'Post' });
    const timeline = page.locator('.ant-list');

    const posts = [
      'First post - getting started',
      'Second post - testing the timeline',
      'Third post - loving this app',
      'Fourth post - more content here',
      'Fifth post - final test post',
    ];

    for (const [index, postContent] of posts.entries()) {
      await composer.fill(postContent);
      await postButton.click();
      
      // Wait for composer to clear
      await expect(composer).toHaveValue('', { timeout: 5000 });
      
      // Verify post count increases
      const cards = timeline.locator('.ant-card');
      await expect(cards).toHaveCount(index + 1, { timeout: 5000 });
    }

    // Verify all posts are visible
    for (const postContent of posts) {
      await expect(timeline.getByText(postContent)).toBeVisible();
    }

    // Verify reverse chronological order (last post first)
    const allCards = timeline.locator('.ant-card');
    await expect(allCards).toHaveCount(5);
    
    // Check order: newest to oldest
    await expect(allCards.nth(0)).toContainText('Fifth post - final test post');
    await expect(allCards.nth(1)).toContainText('Fourth post - more content here');
    await expect(allCards.nth(2)).toContainText('Third post - loving this app');
    await expect(allCards.nth(3)).toContainText('Second post - testing the timeline');
    await expect(allCards.nth(4)).toContainText('First post - getting started');

    // Verify timestamps are visible
    for (let i = 0; i < 5; i++) {
      await expect(allCards.nth(i).getByText(/just now|ago/)).toBeVisible();
    }
  });
});
