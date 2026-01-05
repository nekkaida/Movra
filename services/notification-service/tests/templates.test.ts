import { renderTemplate, templates } from '../src/templates';
import { NotificationType } from '../src/types';

describe('renderTemplate', () => {
  describe('email templates', () => {
    it('should render TRANSFER_CREATED template with all placeholders', () => {
      const data = {
        senderName: 'John Doe',
        recipientName: 'Jane Doe',
        amount: '100',
        currency: 'SGD',
        transferId: 'TRX123',
        sourceCurrency: 'SGD',
        targetCurrency: 'PHP',
        exchangeRate: '41.50',
        recipientAmount: '4150',
      };

      const result = renderTemplate(NotificationType.TRANSFER_CREATED, data, 'email');

      expect(result.subject).toBe('Transfer Initiated - 100 SGD to Jane Doe');
      expect(result.body).toContain('Hi John Doe');
      expect(result.body).toContain('100 SGD');
      expect(result.body).toContain('Jane Doe');
      expect(result.body).toContain('TRX123');
      expect(result.body).toContain('41.50');
    });

    it('should render TRANSFER_COMPLETED template', () => {
      const data = {
        senderName: 'John',
        recipientName: 'Jane',
        amount: '500',
        currency: 'SGD',
        recipientAmount: '20750',
        targetCurrency: 'PHP',
        transferId: 'TRX456',
      };

      const result = renderTemplate(NotificationType.TRANSFER_COMPLETED, data, 'email');

      expect(result.subject).toBe('Transfer Complete - 500 SGD Sent!');
      expect(result.body).toContain('Transfer Complete');
      expect(result.body).toContain('20750 PHP');
    });

    it('should render TRANSFER_FAILED template with failure reason', () => {
      const data = {
        senderName: 'John',
        recipientName: 'Jane',
        transferId: 'TRX789',
        failureReason: 'Insufficient funds',
      };

      const result = renderTemplate(NotificationType.TRANSFER_FAILED, data, 'email');

      expect(result.subject).toBe('Transfer Failed - Action Required');
      expect(result.body).toContain('Insufficient funds');
    });

    it('should render PICKUP_CODE_READY template', () => {
      const data = {
        recipientName: 'Jane',
        amount: '5000',
        currency: 'PHP',
        pickupCode: 'ABC123',
        expiresAt: '2026-01-10',
        transferId: 'TRX101',
      };

      const result = renderTemplate(NotificationType.PICKUP_CODE_READY, data, 'email');

      expect(result.subject).toBe('Cash Pickup Code Ready - TRX101');
      expect(result.body).toContain('ABC123');
      expect(result.body).toContain('5000 PHP');
      expect(result.body).toContain('2026-01-10');
    });
  });

  describe('SMS templates', () => {
    it('should use smsBody for SMS channel', () => {
      const data = {
        amount: '100',
        currency: 'SGD',
        recipientName: 'Jane',
        transferId: 'TRX123',
      };

      const result = renderTemplate(NotificationType.TRANSFER_CREATED, data, 'sms');

      expect(result.body).toBe('Movra: Transfer of 100 SGD to Jane created. ID: TRX123');
      expect(result.body).not.toContain('<h2>');
    });

    it('should render PICKUP_CODE_READY SMS template', () => {
      const data = {
        pickupCode: 'XYZ789',
        amount: '10000',
        currency: 'PHP',
        expiresAt: '2026-01-15',
      };

      const result = renderTemplate(NotificationType.PICKUP_CODE_READY, data, 'sms');

      expect(result.body).toBe('Movra: Cash pickup ready! Code: XYZ789. Amount: 10000 PHP. Expires: 2026-01-15');
    });
  });

  describe('edge cases', () => {
    it('should handle missing placeholders gracefully', () => {
      const data = { senderName: 'John' }; // Missing other fields

      const result = renderTemplate(NotificationType.TRANSFER_CREATED, data, 'email');

      expect(result.subject).toBe('Transfer Initiated -   to ');
      expect(result.body).toContain('Hi John');
    });

    it('should handle unknown notification type', () => {
      const result = renderTemplate('UNKNOWN_TYPE' as NotificationType, {}, 'email');

      expect(result.subject).toBe('Notification');
      expect(result.body).toBe('You have a new notification from Movra.');
    });

    it('should default to email channel when not specified', () => {
      const data = { userName: 'John' };

      const result = renderTemplate(NotificationType.WELCOME, data);

      expect(result.body).toContain('<h2>');
    });
  });

  describe('template coverage', () => {
    it('should have templates for all notification types', () => {
      const notificationTypes = Object.values(NotificationType);

      notificationTypes.forEach((type) => {
        expect(templates[type]).toBeDefined();
        expect(templates[type].subject).toBeDefined();
        expect(templates[type].body).toBeDefined();
      });
    });
  });
});
