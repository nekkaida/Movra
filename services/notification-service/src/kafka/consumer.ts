import { Kafka, Consumer, EachMessagePayload } from 'kafkajs';
import pino from 'pino';
import { NotificationService } from '../services/notificationService';
import { NotificationChannel, NotificationType } from '../types';
import { config } from '../config';

const logger = pino({ name: 'kafka-consumer' });

interface TransferEvent {
  transferId: string;
  userId: string;
  senderEmail?: string;
  senderPhone?: string;
  senderName?: string;
  recipientName?: string;
  amount?: string;
  currency?: string;
  sourceCurrency?: string;
  targetCurrency?: string;
  recipientAmount?: string;
  exchangeRate?: string;
  failureReason?: string;
  pickupCode?: string;
  expiresAt?: string;
}

export class KafkaNotificationConsumer {
  private consumer: Consumer;
  private notificationService: NotificationService;

  constructor(notificationService: NotificationService) {
    const kafka = new Kafka({
      clientId: 'notification-service',
      brokers: config.kafkaBrokers,
    });

    this.consumer = kafka.consumer({ groupId: config.kafkaGroupId });
    this.notificationService = notificationService;
  }

  async start(): Promise<void> {
    await this.consumer.connect();
    await this.consumer.subscribe({
      topics: [
        'movra.transfers.initiated',
        'movra.transfers.funds-received',
        'movra.transfers.completed',
        'movra.transfers.failed',
        'movra.payouts.ready-for-pickup',
        'movra.payouts.completed',
        'movra.payouts.failed',
      ],
      fromBeginning: false,
    });

    await this.consumer.run({
      eachMessage: async (payload: EachMessagePayload) => {
        await this.handleMessage(payload);
      },
    });

    logger.info('Kafka consumer started');
  }

  async stop(): Promise<void> {
    await this.consumer.disconnect();
    logger.info('Kafka consumer stopped');
  }

  private async handleMessage({ topic, message }: EachMessagePayload): Promise<void> {
    const value = message.value?.toString();
    if (!value) return;

    try {
      const event: TransferEvent = JSON.parse(value);
      logger.info({ topic, transferId: event.transferId }, 'Processing event');

      switch (topic) {
        case 'movra.transfers.initiated':
          await this.handleTransferInitiated(event);
          break;
        case 'movra.transfers.funds-received':
          await this.handleFundsReceived(event);
          break;
        case 'movra.transfers.completed':
          await this.handleTransferCompleted(event);
          break;
        case 'movra.transfers.failed':
          await this.handleTransferFailed(event);
          break;
        case 'movra.payouts.ready-for-pickup':
          await this.handlePickupReady(event);
          break;
        default:
          logger.debug({ topic }, 'Unhandled topic');
      }
    } catch (error) {
      logger.error({ error, topic, message: value }, 'Failed to process message');
    }
  }

  private async handleTransferInitiated(event: TransferEvent): Promise<void> {
    if (event.senderEmail) {
      await this.notificationService.sendNotification({
        userId: event.userId,
        channel: NotificationChannel.EMAIL,
        type: NotificationType.TRANSFER_CREATED,
        recipient: event.senderEmail,
        templateData: {
          senderName: event.senderName || 'Customer',
          recipientName: event.recipientName || 'Recipient',
          amount: event.amount || '0',
          currency: event.currency || 'SGD',
          sourceCurrency: event.sourceCurrency || 'SGD',
          targetCurrency: event.targetCurrency || 'PHP',
          recipientAmount: event.recipientAmount || '0',
          exchangeRate: event.exchangeRate || '1',
          transferId: event.transferId,
        },
        correlationId: event.transferId,
      });
    }
  }

  private async handleFundsReceived(event: TransferEvent): Promise<void> {
    if (event.senderEmail) {
      await this.notificationService.sendNotification({
        userId: event.userId,
        channel: NotificationChannel.EMAIL,
        type: NotificationType.TRANSFER_FUNDS_RECEIVED,
        recipient: event.senderEmail,
        templateData: {
          senderName: event.senderName || 'Customer',
          recipientName: event.recipientName || 'Recipient',
          amount: event.amount || '0',
          currency: event.currency || 'SGD',
          transferId: event.transferId,
        },
        correlationId: event.transferId,
      });
    }
  }

  private async handleTransferCompleted(event: TransferEvent): Promise<void> {
    if (event.senderEmail) {
      await this.notificationService.sendNotification({
        userId: event.userId,
        channel: NotificationChannel.EMAIL,
        type: NotificationType.TRANSFER_COMPLETED,
        recipient: event.senderEmail,
        templateData: {
          senderName: event.senderName || 'Customer',
          recipientName: event.recipientName || 'Recipient',
          amount: event.amount || '0',
          currency: event.currency || 'SGD',
          targetCurrency: event.targetCurrency || 'PHP',
          recipientAmount: event.recipientAmount || '0',
          transferId: event.transferId,
        },
        correlationId: event.transferId,
      });
    }
  }

  private async handleTransferFailed(event: TransferEvent): Promise<void> {
    if (event.senderEmail) {
      await this.notificationService.sendNotification({
        userId: event.userId,
        channel: NotificationChannel.EMAIL,
        type: NotificationType.TRANSFER_FAILED,
        recipient: event.senderEmail,
        templateData: {
          senderName: event.senderName || 'Customer',
          recipientName: event.recipientName || 'Recipient',
          transferId: event.transferId,
          failureReason: event.failureReason || 'Unknown error',
        },
        correlationId: event.transferId,
      });
    }
  }

  private async handlePickupReady(event: TransferEvent): Promise<void> {
    // Send SMS for pickup code (time-sensitive)
    if (event.senderPhone) {
      await this.notificationService.sendNotification({
        userId: event.userId,
        channel: NotificationChannel.SMS,
        type: NotificationType.PICKUP_CODE_READY,
        recipient: event.senderPhone,
        templateData: {
          recipientName: event.recipientName || 'Customer',
          amount: event.amount || '0',
          currency: event.currency || 'PHP',
          pickupCode: event.pickupCode || '',
          expiresAt: event.expiresAt || '',
          transferId: event.transferId,
        },
        correlationId: event.transferId,
      });
    }
  }
}
