import { test, expect } from '@playwright/test';

test('application loads without errors', async ({ page }) => {
  const errors: string[] = [];
  
  // Capture console errors
  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      errors.push(msg.text());
    }
  });

  // Capture page errors
  page.on('pageerror', (error) => {
    errors.push(error.message);
  });

  // Navigate to the app
  await page.goto('/');

  // Wait for the page to load
  await page.waitForLoadState('networkidle');

  // Check that the page title is correct
  await expect(page).toHaveTitle(/Mau Status/);

  // Check for the app header
  await expect(page.getByText('Mau Status')).toBeVisible();

  // Verify no console or page errors occurred
  expect(errors).toHaveLength(0);
});
