export const config = {
  // Server
  httpPort: parseInt(process.env.HTTP_PORT || '3002', 10),
  grpcPort: parseInt(process.env.GRPC_PORT || '50053', 10),

  // Kafka
  kafkaBrokers: (process.env.KAFKA_BROKERS || 'localhost:9092').split(','),
  kafkaGroupId: process.env.KAFKA_GROUP_ID || 'notification-service-group',

  // Email (SMTP)
  smtp: {
    host: process.env.SMTP_HOST || 'localhost',
    port: parseInt(process.env.SMTP_PORT || '1025', 10),
    secure: process.env.SMTP_SECURE === 'true',
    auth: process.env.SMTP_USER ? {
      user: process.env.SMTP_USER,
      pass: process.env.SMTP_PASS,
    } : undefined,
    from: process.env.SMTP_FROM || 'noreply@movra.com',
  },

  // SMS (Twilio-style config)
  sms: {
    provider: process.env.SMS_PROVIDER || 'simulated',
    accountSid: process.env.SMS_ACCOUNT_SID || '',
    authToken: process.env.SMS_AUTH_TOKEN || '',
    fromNumber: process.env.SMS_FROM_NUMBER || '+15551234567',
  },

  // Retry
  maxRetries: parseInt(process.env.MAX_RETRIES || '3', 10),
};
