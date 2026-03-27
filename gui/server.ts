import indexHtml from './public/index.html';

const server = Bun.serve({
  port: 3000,
  routes: {
    '/': indexHtml,
  },
  development: {
    hmr: true,
    console: true,
  },
});

console.log(`🚀 Server running at http://localhost:${server.port}`);
