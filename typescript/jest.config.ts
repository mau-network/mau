import type { Config } from 'jest';

const config: Config = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  setupFilesAfterEnv: ['<rootDir>/jest.setup.ts'],
  roots: ['<rootDir>/src'],
  testMatch: ['**/*.test.ts', '!**/*.browser.test.ts', '!**/dht-bootstrap-e2e.test.ts'],
  moduleNameMapper: {
    '^(\\.{1,2}/.*)\\.js$': '$1',
  },
  transformIgnorePatterns: [
    'node_modules/(?!node-datachannel)'
  ],
  testTimeout: 20000,
  collectCoverage: true,
  coverageDirectory: 'coverage',
  coverageReporters: ['text', 'lcov', 'html'],
  coveragePathIgnorePatterns: [
    '/node_modules/',
    '/dist/',
  ],
  collectCoverageFrom: [
    'src/**/*.ts',
    '!src/**/*.test.ts',
    '!src/**/*.d.ts',
  ],
  coverageThreshold: {
    global: {
      // branches are lower because the DHT and WebRTC code (dht.ts,
      // signaling.ts, webrtc.ts) is heavily async and requires a real
      // P2P network to exercise the conditional paths fully.
      branches: 35,
      functions: 50,
      lines: 50,
      statements: 50
    }
  }
};

export default config;
