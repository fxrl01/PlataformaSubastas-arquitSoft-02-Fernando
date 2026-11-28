// @ts-check
const path = require('path');
const { defineConfig } = require(path.join(__dirname, '../e2e/node_modules/@playwright/test'));

module.exports = defineConfig({
  testDir: __dirname,
  testMatch: '*.spec.js',
  timeout: 45_000,
  expect: {
    timeout: 10_000,
  },
  retries: process.env.CI ? 1 : 0,
  reporter: [['list']],
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:3000',
    headless: true,
    trace: 'on-first-retry',
  },
});
