import express from 'express';
import pinoHttp from 'pino-http';
import { Kafka } from 'kafkajs';
import pino from 'pino';
import { Registry, collectDefaultMetrics } from 'prom-client';

const logger = pino({
  level: process.env.NODE_ENV === 'production' ? 'info' : 'debug',
});

const config = {
  port: parseInt(process.env.PORT || '3002', 10),
  kafkaBrokers: (process.env.KAFKA_BROKERS || 'localhost:9092').split(','),
  smtpHost: process.env.SMTP_HOST || 'localhost',
  smtpPort: parseInt(process.env.SMTP_PORT || '1025', 10),
};

// Prometheus metrics
const register = new Registry();
collectDefaultMetrics({ register });

// Kafka consumer
const kafka = new Kafka({
  clientId: 'notification-service',
  brokers: config.kafkaBrokers,
});

const consumer = kafka.consumer({ groupId: 'notification-service-group' });

// Express app
const app = express();
app.use(express.json());
app.use(pinoHttp({ logger }));

// Health endpoints
app.get('/health', (_req, res) => {
  res.json({ status: 'healthy', service: 'notification-service' });
});

app.get('/ready', (_req, res) => {
  res.json({ status: 'ready', service: 'notification-service' });
});

// Metrics endpoint
app.get('/metrics', async (_req, res) => {
  try {
    res.set('Content-Type', register.contentType);
    res.end(await register.metrics());
  } catch (error) {
    res.status(500).end();
  }
});

// API endpoints
app.post('/api/notifications', async (req, res) => {
  const { userId, channel, type, recipient, templateData, correlationId } = req.body;

  // TODO: Implement actual notification sending
  const notification = {
    id: `notif_${Date.now()}`,
    userId,
    channel,
    type,
    recipient,
    status: 'SENT',
    correlationId,
    createdAt: new Date().toISOString(),
  };

  logger.info({ notification }, 'Notification sent');

  res.status(201).json(notification);
});

app.get('/api/notifications/:id', (req, res) => {
  const { id } = req.params;
  res.json({
    id,
    status: 'DELIVERED',
    deliveredAt: new Date().toISOString(),
  });
});

// Kafka consumer setup
async function startKafkaConsumer() {
  try {
    await consumer.connect();
    await consumer.subscribe({
      topics: [
        'movra.transfers.initiated',
        'movra.transfers.funds-received',
        'movra.transfers.completed',
        'movra.transfers.failed',
        'movra.payouts.completed',
        'movra.payouts.failed',
      ],
      fromBeginning: false,
    });

    await consumer.run({
      eachMessage: async ({ topic, partition, message }) => {
        const value = message.value?.toString();
        if (!value) return;

        try {
          const event = JSON.parse(value);
          logger.info({ topic, event }, 'Received event');

          // TODO: Implement notification logic based on event type
          // For now, just log
          switch (topic) {
            case 'movra.transfers.initiated':
              logger.info({ transferId: event.transferId }, 'Would send transfer initiated notification');
              break;
            case 'movra.transfers.completed':
              logger.info({ transferId: event.transferId }, 'Would send transfer completed notification');
              break;
            case 'movra.transfers.failed':
              logger.info({ transferId: event.transferId }, 'Would send transfer failed notification');
              break;
            default:
              logger.debug({ topic }, 'Unhandled topic');
          }
        } catch (error) {
          logger.error({ error, topic, message: value }, 'Failed to process message');
        }
      },
    });

    logger.info('Kafka consumer started');
  } catch (error) {
    logger.error({ error }, 'Failed to start Kafka consumer');
  }
}

// Start server
app.listen(config.port, () => {
  logger.info({ port: config.port }, 'Notification Service started');
});

// Start Kafka consumer (don't await - let it run in background)
startKafkaConsumer().catch((error) => {
  logger.error({ error }, 'Kafka consumer error');
});

// Graceful shutdown
process.on('SIGTERM', async () => {
  logger.info('SIGTERM received, shutting down');
  await consumer.disconnect();
  process.exit(0);
});
