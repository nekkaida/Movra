import express from 'express';
import pinoHttp from 'pino-http';
import pino from 'pino';
import * as grpc from '@grpc/grpc-js';
import { Registry, collectDefaultMetrics, Counter, Histogram } from 'prom-client';

import { config } from './config';
import { NotificationService } from './services/notificationService';
import { createGrpcServer } from './grpc/server';
import { KafkaNotificationConsumer } from './kafka/consumer';
import { NotificationChannel } from './types';

const logger = pino({
  level: process.env.NODE_ENV === 'production' ? 'info' : 'debug',
  name: 'notification-service',
});

// Prometheus metrics
const register = new Registry();
collectDefaultMetrics({ register });

const notificationsSentTotal = new Counter({
  name: 'notifications_sent_total',
  help: 'Total notifications sent',
  labelNames: ['channel', 'type', 'status'],
  registers: [register],
});

const notificationDuration = new Histogram({
  name: 'notification_send_duration_seconds',
  help: 'Time to send notification',
  labelNames: ['channel'],
  buckets: [0.1, 0.5, 1, 2, 5],
  registers: [register],
});

// Initialize notification service (providers are created internally)
const notificationService = new NotificationService();

// Initialize Kafka consumer
const kafkaConsumer = new KafkaNotificationConsumer(notificationService);

// Initialize gRPC server
const grpcServer = createGrpcServer(notificationService);

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

  const start = Date.now();
  try {
    const notification = await notificationService.sendNotification({
      userId,
      channel: channel as NotificationChannel,
      type,
      recipient,
      templateData: templateData || {},
      correlationId,
    });

    notificationsSentTotal.inc({ channel, type, status: notification.status });
    notificationDuration.observe({ channel }, (Date.now() - start) / 1000);

    logger.info({ notificationId: notification.id }, 'Notification sent via HTTP');
    res.status(201).json(notification);
  } catch (error) {
    logger.error({ error }, 'Failed to send notification');
    res.status(500).json({ error: 'Failed to send notification' });
  }
});

app.get('/api/notifications/:id', async (req, res) => {
  const { id } = req.params;
  const notification = await notificationService.getNotification(id);

  if (!notification) {
    res.status(404).json({ error: 'Notification not found' });
    return;
  }

  res.json(notification);
});

app.get('/api/notifications', async (req, res) => {
  const { userId, channel, limit, offset } = req.query;

  if (!userId) {
    res.status(400).json({ error: 'userId is required' });
    return;
  }

  const notifications = await notificationService.listNotifications(userId as string, {
    channel: channel as NotificationChannel | undefined,
    limit: limit ? parseInt(limit as string, 10) : 20,
    offset: offset ? parseInt(offset as string, 10) : 0,
  });

  res.json({ notifications, total: notifications.length });
});

app.post('/api/notifications/:id/resend', async (req, res) => {
  const { id } = req.params;

  try {
    const notification = await notificationService.resendNotification(id);
    if (!notification) {
      res.status(404).json({ error: 'Notification not found' });
      return;
    }
    res.json(notification);
  } catch (error) {
    logger.error({ error, notificationId: id }, 'Failed to resend notification');
    res.status(500).json({ error: 'Failed to resend notification' });
  }
});

// Start servers
async function start() {
  // Start HTTP server
  app.listen(config.httpPort, () => {
    logger.info({ port: config.httpPort }, 'HTTP server started');
  });

  // Start gRPC server
  grpcServer.bindAsync(
    `0.0.0.0:${config.grpcPort}`,
    grpc.ServerCredentials.createInsecure(),
    (error, port) => {
      if (error) {
        logger.error({ error }, 'Failed to start gRPC server');
        return;
      }
      logger.info({ port }, 'gRPC server started');
    }
  );

  // Start Kafka consumer
  try {
    await kafkaConsumer.start();
    logger.info('Kafka consumer started');
  } catch (error) {
    logger.error({ error }, 'Failed to start Kafka consumer');
  }
}

// Graceful shutdown
async function shutdown() {
  logger.info('Shutting down...');

  try {
    await kafkaConsumer.stop();
    grpcServer.forceShutdown();
    logger.info('Shutdown complete');
    process.exit(0);
  } catch (error) {
    logger.error({ error }, 'Error during shutdown');
    process.exit(1);
  }
}

process.on('SIGTERM', shutdown);
process.on('SIGINT', shutdown);

// Start the service
start().catch((error) => {
  logger.error({ error }, 'Failed to start service');
  process.exit(1);
});
