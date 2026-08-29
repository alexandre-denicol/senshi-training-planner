const target = process.env.E2E_API_PROXY_TARGET || process.env.API_PROXY_TARGET || 'http://localhost:18080';

module.exports = {
  '/api': {
    target,
    secure: false,
    changeOrigin: true,
    pathRewrite: {
      '^/api': '',
    },
  },
};
