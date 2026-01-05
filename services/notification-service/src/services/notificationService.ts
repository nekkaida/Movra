import { v4 as uuidv4 } from 'uuid';
import pino from 'pino';
import {
  Notification,
  NotificationChannel,
  NotificationType,
  DeliveryStatus,
  SendNotificationRequest,
} from '../types';
import { EmailProvider, SmsProvider, NotificationProvider } from '../providers';
import { renderTemplate } from '../templates';
import { config } from '../config';

const logger = pino({ name: 'notification-service' });

// In-memory store (replace with Redis/DB in production)
const notifications = new Map<string, Notification>();

export class NotificationService {
  private emailProvider: NotificationProvider;
  private smsProvider: NotificationProvider;

  constructor() {
    this.emailProvider = new EmailProvider();
    this.smsProvider = new SmsProvider();
  }

  async sendNotification(request: SendNotificationRequest): Promise<Notification> {
    const { userId, channel, type, recipient, templateData, correlationId } = request;

    // Render template
    const channelType = channel === NotificationChannel.SMS ? 'sms' : 'email';
    const { subject, body } = renderTemplate(type, templateData, channelType);

    // Create notification record
    const notification: Notification = {
      id: `notif_${uuidv4()}`,
      userId,
      channel,
      type,
      status: DeliveryStatus.PENDING,
      recipient: recipient || '',
      subject: channel === NotificationChannel.EMAIL ? subject : undefined,
      body,
      templateData,
      correlationId,
      retryCount: 0,
      createdAt: new Date(),
    };

    // Save notification
    notifications.set(notification.id, notification);

    // Send via appropriate provider
    try {
      const provider = this.getProvider(channel);
      const result = await provider.send(notification.recipient, subject, body);

      if (result.success) {
        notification.status = DeliveryStatus.SENT;
        notification.providerMessageId = result.messageId;
        notification.sentAt = new Date();
        logger.info({ notificationId: notification.id, channel }, 'Notification sent');
      } else {
        notification.status = DeliveryStatus.FAILED;
        notification.failureReason = result.error;
        logger.error({ notificationId: notification.id, error: result.error }, 'Notification failed');
      }
    } catch (error) {
      notification.status = DeliveryStatus.FAILED;
      notification.failureReason = error instanceof Error ? error.message : 'Unknown error';
      logger.error({ notificationId: notification.id, error }, 'Notification error');
    }

    // Update stored notification
    notifications.set(notification.id, notification);

    return notification;
  }

  async getNotification(id: string): Promise<Notification | null> {
    return notifications.get(id) || null;
  }

  async listNotifications(
    userId: string,
    options: {
      channel?: NotificationChannel;
      type?: NotificationType;
      limit?: number;
      offset?: number;
    } = {}
  ): Promise<Notification[]> {
    const { channel, type, limit = 20, offset = 0 } = options;

    const result = Array.from(notifications.values())
      .filter((n) => n.userId === userId)
      .filter((n) => !channel || n.channel === channel)
      .filter((n) => !type || n.type === type)
      .sort((a, b) => b.createdAt.getTime() - a.createdAt.getTime());

    return result.slice(offset, offset + limit);
  }

  async resendNotification(id: string): Promise<Notification | null> {
    const notification = notifications.get(id);
    if (!notification) {
      return null;
    }

    if (notification.status !== DeliveryStatus.FAILED) {
      throw new Error('Can only resend failed notifications');
    }

    if (notification.retryCount >= config.maxRetries) {
      throw new Error(`Max retries (${config.maxRetries}) exceeded`);
    }

    notification.retryCount++;
    notification.status = DeliveryStatus.PENDING;
    notification.failureReason = undefined;

    try {
      const provider = this.getProvider(notification.channel);
      const result = await provider.send(
        notification.recipient,
        notification.subject || '',
        notification.body
      );

      if (result.success) {
        notification.status = DeliveryStatus.SENT;
        notification.providerMessageId = result.messageId;
        notification.sentAt = new Date();
      } else {
        notification.status = DeliveryStatus.FAILED;
        notification.failureReason = result.error;
      }
    } catch (error) {
      notification.status = DeliveryStatus.FAILED;
      notification.failureReason = error instanceof Error ? error.message : 'Unknown error';
    }

    notifications.set(notification.id, notification);
    return notification;
  }

  private getProvider(channel: NotificationChannel): NotificationProvider {
    switch (channel) {
      case NotificationChannel.EMAIL:
        return this.emailProvider;
      case NotificationChannel.SMS:
        return this.smsProvider;
      default:
        throw new Error(`Unsupported channel: ${channel}`);
    }
  }
}
