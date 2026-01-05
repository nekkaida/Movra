import { NotificationService } from '../src/services/notificationService';
import { NotificationChannel, NotificationType, DeliveryStatus } from '../src/types';

// Mock the providers
jest.mock('../src/providers/emailProvider', () => ({
  EmailProvider: jest.fn().mockImplementation(() => ({
    name: 'email',
    send: jest.fn().mockResolvedValue({ success: true, messageId: 'EMAIL_123' }),
  })),
}));

jest.mock('../src/providers/smsProvider', () => ({
  SmsProvider: jest.fn().mockImplementation(() => ({
    name: 'sms',
    send: jest.fn().mockResolvedValue({ success: true, messageId: 'SMS_123' }),
  })),
}));

describe('NotificationService', () => {
  let service: NotificationService;

  beforeEach(() => {
    jest.clearAllMocks();
    service = new NotificationService();
  });

  describe('sendNotification', () => {
    it('should send email notification successfully', async () => {
      const notification = await service.sendNotification({
        userId: 'user_123',
        channel: NotificationChannel.EMAIL,
        type: NotificationType.TRANSFER_CREATED,
        recipient: 'test@example.com',
        templateData: {
          senderName: 'John',
          recipientName: 'Jane',
          amount: '100',
          currency: 'SGD',
          transferId: 'TRX123',
          sourceCurrency: 'SGD',
          targetCurrency: 'PHP',
          exchangeRate: '41.50',
          recipientAmount: '4150',
        },
        correlationId: 'TRX123',
      });

      expect(notification.id).toMatch(/^notif_/);
      expect(notification.userId).toBe('user_123');
      expect(notification.channel).toBe(NotificationChannel.EMAIL);
      expect(notification.type).toBe(NotificationType.TRANSFER_CREATED);
      expect(notification.status).toBe(DeliveryStatus.SENT);
      expect(notification.providerMessageId).toBe('EMAIL_123');
      expect(notification.sentAt).toBeDefined();
    });

    it('should send SMS notification successfully', async () => {
      const notification = await service.sendNotification({
        userId: 'user_456',
        channel: NotificationChannel.SMS,
        type: NotificationType.PICKUP_CODE_READY,
        recipient: '+6591234567',
        templateData: {
          recipientName: 'Jane',
          amount: '5000',
          currency: 'PHP',
          pickupCode: 'ABC123',
          expiresAt: '2026-01-10',
          transferId: 'TRX456',
        },
        correlationId: 'TRX456',
      });

      expect(notification.channel).toBe(NotificationChannel.SMS);
      expect(notification.status).toBe(DeliveryStatus.SENT);
      expect(notification.providerMessageId).toBe('SMS_123');
    });

    it('should store notification with correct template data', async () => {
      const templateData = {
        senderName: 'John',
        recipientName: 'Jane',
        amount: '100',
        currency: 'SGD',
        transferId: 'TRX123',
        sourceCurrency: 'SGD',
        targetCurrency: 'PHP',
        exchangeRate: '41.50',
        recipientAmount: '4150',
      };

      const notification = await service.sendNotification({
        userId: 'user_123',
        channel: NotificationChannel.EMAIL,
        type: NotificationType.TRANSFER_CREATED,
        recipient: 'test@example.com',
        templateData,
        correlationId: 'TRX123',
      });

      expect(notification.templateData).toEqual(templateData);
      expect(notification.correlationId).toBe('TRX123');
    });
  });

  describe('getNotification', () => {
    it('should retrieve existing notification', async () => {
      const sent = await service.sendNotification({
        userId: 'user_123',
        channel: NotificationChannel.EMAIL,
        type: NotificationType.WELCOME,
        recipient: 'test@example.com',
        templateData: { userName: 'John' },
      });

      const retrieved = await service.getNotification(sent.id);

      expect(retrieved).not.toBeNull();
      expect(retrieved?.id).toBe(sent.id);
      expect(retrieved?.userId).toBe('user_123');
    });

    it('should return null for non-existent notification', async () => {
      const result = await service.getNotification('notif_does_not_exist');

      expect(result).toBeNull();
    });
  });

  describe('listNotifications', () => {
    beforeEach(async () => {
      // Create multiple notifications for the same user
      await service.sendNotification({
        userId: 'user_list_test',
        channel: NotificationChannel.EMAIL,
        type: NotificationType.TRANSFER_CREATED,
        recipient: 'test@example.com',
        templateData: {},
      });
      await service.sendNotification({
        userId: 'user_list_test',
        channel: NotificationChannel.SMS,
        type: NotificationType.PICKUP_CODE_READY,
        recipient: '+6591234567',
        templateData: {},
      });
      await service.sendNotification({
        userId: 'user_list_test',
        channel: NotificationChannel.EMAIL,
        type: NotificationType.TRANSFER_COMPLETED,
        recipient: 'test@example.com',
        templateData: {},
      });
    });

    it('should list all notifications for a user', async () => {
      const notifications = await service.listNotifications('user_list_test');

      expect(notifications.length).toBeGreaterThanOrEqual(3);
      expect(notifications.every((n) => n.userId === 'user_list_test')).toBe(true);
    });

    it('should filter by channel', async () => {
      const notifications = await service.listNotifications('user_list_test', {
        channel: NotificationChannel.EMAIL,
      });

      expect(notifications.every((n) => n.channel === NotificationChannel.EMAIL)).toBe(true);
    });

    it('should apply limit and offset', async () => {
      const notifications = await service.listNotifications('user_list_test', {
        limit: 2,
        offset: 0,
      });

      expect(notifications.length).toBeLessThanOrEqual(2);
    });

    it('should return empty array for user with no notifications', async () => {
      const notifications = await service.listNotifications('non_existent_user');

      expect(notifications).toEqual([]);
    });
  });
});
