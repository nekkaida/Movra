export const config = {
  port: parseInt(process.env.PORT || '3000', 10),
  nodeEnv: process.env.NODE_ENV || 'development',

  // gRPC service URLs
  services: {
    auth: process.env.AUTH_SERVICE_URL || 'localhost:5002',
    payment: process.env.PAYMENT_SERVICE_URL || 'localhost:9091',
    exchangeRate: process.env.EXCHANGE_RATE_SERVICE_URL || 'localhost:9092',
    settlement: process.env.SETTLEMENT_SERVICE_URL || 'localhost:9093',
  },

  // JWT
  jwt: {
    secret: process.env.JWT_SECRET || 'dev-secret-change-in-production',
  },

  // Rate limiting
  rateLimit: {
    windowMs: parseInt(process.env.RATE_LIMIT_WINDOW_MS || '60000', 10), // 1 minute
    max: parseInt(process.env.RATE_LIMIT_MAX || '100', 10), // 100 requests per window
  },

  // Cors
  cors: {
    origin: process.env.CORS_ORIGIN || '*',
  },
};
