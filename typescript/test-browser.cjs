const puppeteer = require('puppeteer');

(async () => {
  const browser = await puppeteer.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });
  
  const page = await browser.newPage();
  
  // Listen to console logs
  page.on('console', msg => {
    const type = msg.type();
    const text = msg.text();
    console.log(`[${type}] ${text}`);
  });
  
  // Listen to errors
  page.on('pageerror', error => {
    console.error('Page error:', error.message);
  });
  
  // Navigate to test page
  await page.goto('http://localhost:8888/test-standalone.html', {
    waitUntil: 'networkidle0',
    timeout: 30000,
  });
  
  // Wait for tests to complete
  await page.waitForFunction(
    () => {
      const status = document.getElementById('status');
      return status && (status.textContent.includes('passed') || status.textContent.includes('failed'));
    },
    { timeout: 30000 }
  );
  
  // Get final status
  const status = await page.evaluate(() => {
    return document.getElementById('status').textContent;
  });
  
  const output = await page.evaluate(() => {
    return document.getElementById('output').textContent;
  });
  
  console.log('\n=== Final Status ===');
  console.log(status);
  console.log('\n=== Test Output ===');
  console.log(output);
  
  await browser.close();
  
  // Exit with proper code
  process.exit(status.includes('passed') ? 0 : 1);
})();
