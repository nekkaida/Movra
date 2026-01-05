# Notification Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a production-ready Notification Service with gRPC server, email/SMS providers, template system, and proper Kafka event handling.

**Architecture:** Event-driven service consuming Kafka events, processing notifications through provider interfaces, exposing gRPC for direct calls, with a template system for consistent messaging.

**Tech Stack:** Node.js/TypeScript, gRPC (@grpc/grpc-js), Kafka (kafkajs), Nodemailer (email), Templates (Handlebars), Jest (testing)

---

## Task 1: Project Structure Setup

**Files:**
- Create: `services/notification-service/src/config/index.ts`
- Create: `services/notification-service/src/types/index.ts`

**Step 1: Create config file**

```typescript
// src/config/index.ts
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
```

**Step 2: Create types file**

```typescript
// src/types/index.ts
export enum NotificationChannel {
  EMAIL = 'EMAIL',
  SMS = 'SMS',
  PUSH = 'PUSH',
}

export enum NotificationType {
  TRANSFER_CREATED = 'TRANSFER_CREATED',
  TRANSFER_AWAITING_FUNDS = 'TRANSFER_AWAITING_FUNDS',
  TRANSFER_FUNDS_RECEIVED = 'TRANSFER_FUNDS_RECEIVED',
  TRANSFER_PROCESSING = 'TRANSFER_PROCESSING',
  TRANSFER_COMPLETED = 'TRANSFER_COMPLETED',
  TRANSFER_FAILED = 'TRANSFER_FAILED',
  TRANSFER_REFUNDED = 'TRANSFER_REFUNDED',
  PICKUP_CODE_READY = 'PICKUP_CODE_READY',
  PICKUP_REMINDER = 'PICKUP_REMINDER',
  PICKUP_COLLECTED = 'PICKUP_COLLECTED',
  WELCOME = 'WELCOME',
  KYC_APPROVED = 'KYC_APPROVED',
  KYC_REJECTED = 'KYC_REJECTED',
  PASSWORD_RESET = 'PASSWORD_RESET',
  LOGIN_ALERT = 'LOGIN_ALERT',
}

export enum DeliveryStatus {
  PENDING = 'PENDING',
  SENT = 'SENT',
  DELIVERED = 'DELIVERED',
  FAILED = 'FAILED',
  BOUNCED = 'BOUNCED',
}

export interface Notification {
  id: string;
  userId: string;
  channel: NotificationChannel;
  type: NotificationType;
  status: DeliveryStatus;
  recipient: string;
  subject?: string;
  body: string;
  templateData: Record<string, string>;
  correlationId?: string;
  providerMessageId?: string;
  failureReason?: string;
  retryCount: number;
  createdAt: Date;
  sentAt?: Date;
  deliveredAt?: Date;
}

export interface SendNotificationRequest {
  userId: string;
  channel: NotificationChannel;
  type: NotificationType;
  recipient?: string;
  templateData: Record<string, string>;
  correlationId?: string;
}
```

**Step 3: Verify TypeScript compiles**

Run: `cd services/notification-service && npx tsc --noEmit`

**Step 4: Commit**

```bash
git add services/notification-service/src/config/ services/notification-service/src/types/
git commit -m "feat(notification): add configuration and type definitions"
```

---

## Task 2: Template System

**Files:**
- Create: `services/notification-service/src/templates/index.ts`

**Step 1: Create template system**

```typescript
// src/templates/index.ts
import { NotificationType } from '../types';

interface Template {
  subject: string;
  body: string;
  smsBody?: string;
}

const templates: Record<NotificationType, Template> = {
  [NotificationType.TRANSFER_CREATED]: {
    subject: 'Transfer Initiated - {{amount}} {{currency}} to {{recipientName}}',
    body: `
      <h2>Transfer Initiated</h2>
      <p>Hi {{senderName}},</p>
      <p>Your transfer of <strong>{{amount}} {{currency}}</strong> to {{recipientName}} has been created.</p>
      <p><strong>Transfer ID:</strong> {{transferId}}</p>
      <p><strong>Exchange Rate:</strong> 1 {{sourceCurrency}} = {{exchangeRate}} {{targetCurrency}}</p>
      <p><strong>Recipient Gets:</strong> {{recipientAmount}} {{targetCurrency}}</p>
      <p>Please complete the payment to proceed with your transfer.</p>
      <p>Thank you for using Movra!</p>
    `,
    smsBody: 'Movra: Transfer of {{amount}} {{currency}} to {{recipientName}} created. ID: {{transferId}}',
  },

  [NotificationType.TRANSFER_FUNDS_RECEIVED]: {
    subject: 'Funds Received - Processing Your Transfer',
    body: `
      <h2>Funds Received</h2>
      <p>Hi {{senderName}},</p>
      <p>We've received your payment of <strong>{{amount}} {{currency}}</strong>.</p>
      <p>Your transfer to {{recipientName}} is now being processed.</p>
      <p><strong>Transfer ID:</strong> {{transferId}}</p>
      <p>We'll notify you once the transfer is complete.</p>
    `,
    smsBody: 'Movra: Payment received. Transfer {{transferId}} to {{recipientName}} is processing.',
  },

  [NotificationType.TRANSFER_COMPLETED]: {
    subject: 'Transfer Complete - {{amount}} {{currency}} Sent!',
    body: `
      <h2>Transfer Complete!</h2>
      <p>Hi {{senderName}},</p>
      <p>Great news! Your transfer has been completed successfully.</p>
      <p><strong>Amount Sent:</strong> {{amount}} {{currency}}</p>
      <p><strong>Recipient:</strong> {{recipientName}}</p>
      <p><strong>Amount Received:</strong> {{recipientAmount}} {{targetCurrency}}</p>
      <p><strong>Transfer ID:</strong> {{transferId}}</p>
      <p>Thank you for choosing Movra!</p>
    `,
    smsBody: 'Movra: Transfer complete! {{recipientName}} received {{recipientAmount}} {{targetCurrency}}. ID: {{transferId}}',
  },

  [NotificationType.TRANSFER_FAILED]: {
    subject: 'Transfer Failed - Action Required',
    body: `
      <h2>Transfer Failed</h2>
      <p>Hi {{senderName}},</p>
      <p>Unfortunately, your transfer to {{recipientName}} could not be completed.</p>
      <p><strong>Transfer ID:</strong> {{transferId}}</p>
      <p><strong>Reason:</strong> {{failureReason}}</p>
      <p>Please contact support if you need assistance.</p>
    `,
    smsBody: 'Movra: Transfer {{transferId}} failed. Reason: {{failureReason}}. Contact support.',
  },

  [NotificationType.PICKUP_CODE_READY]: {
    subject: 'Cash Pickup Code Ready - {{transferId}}',
    body: `
      <h2>Your Cash Pickup Code is Ready</h2>
      <p>Hi {{recipientName}},</p>
      <p>You have a cash pickup waiting for you!</p>
      <p><strong>Amount:</strong> {{amount}} {{currency}}</p>
      <p><strong>Pickup Code:</strong> <span style="font-size: 24px; font-weight: bold;">{{pickupCode}}</span></p>
      <p><strong>Expires:</strong> {{expiresAt}}</p>
      <p>Present this code and your ID at any authorized pickup location.</p>
    `,
    smsBody: 'Movra: Cash pickup ready! Code: {{pickupCode}}. Amount: {{amount}} {{currency}}. Expires: {{expiresAt}}',
  },

  [NotificationType.WELCOME]: {
    subject: 'Welcome to Movra!',
    body: `
      <h2>Welcome to Movra!</h2>
      <p>Hi {{userName}},</p>
      <p>Thank you for joining Movra. We're excited to help you send money internationally.</p>
      <p>Get started by completing your profile verification to unlock all features.</p>
      <p>If you have any questions, our support team is here to help.</p>
    `,
    smsBody: 'Welcome to Movra, {{userName}}! Complete verification to start sending money internationally.',
  },

  [NotificationType.PASSWORD_RESET]: {
    subject: 'Password Reset Request',
    body: `
      <h2>Password Reset</h2>
      <p>Hi {{userName}},</p>
      <p>We received a request to reset your password.</p>
      <p>Click the link below to reset your password:</p>
      <p><a href="{{resetLink}}">Reset Password</a></p>
      <p>This link expires in {{expiresIn}}.</p>
      <p>If you didn't request this, please ignore this email.</p>
    `,
    smsBody: 'Movra: Password reset requested. Link expires in {{expiresIn}}.',
  },

  // Placeholder templates for other types
  [NotificationType.TRANSFER_AWAITING_FUNDS]: {
    subject: 'Awaiting Payment - {{transferId}}',
    body: '<p>Please complete payment for transfer {{transferId}}</p>',
    smsBody: 'Movra: Complete payment for transfer {{transferId}}',
  },
  [NotificationType.TRANSFER_PROCESSING]: {
    subject: 'Transfer Processing - {{transferId}}',
    body: '<p>Your transfer {{transferId}} is being processed</p>',
    smsBody: 'Movra: Transfer {{transferId}} processing',
  },
  [NotificationType.TRANSFER_REFUNDED]: {
    subject: 'Transfer Refunded - {{transferId}}',
    body: '<p>Your transfer {{transferId}} has been refunded</p>',
    smsBody: 'Movra: Transfer {{transferId}} refunded',
  },
  [NotificationType.PICKUP_REMINDER]: {
    subject: 'Reminder: Cash Pickup Expiring Soon',
    body: '<p>Your pickup code {{pickupCode}} expires on {{expiresAt}}</p>',
    smsBody: 'Movra: Pickup code {{pickupCode}} expires {{expiresAt}}',
  },
  [NotificationType.PICKUP_COLLECTED]: {
    subject: 'Cash Collected',
    body: '<p>Cash pickup {{transferId}} has been collected</p>',
    smsBody: 'Movra: Cash collected for {{transferId}}',
  },
  [NotificationType.KYC_APPROVED]: {
    subject: 'Verification Approved',
    body: '<p>Your identity verification has been approved!</p>',
    smsBody: 'Movra: Your verification is approved!',
  },
  [NotificationType.KYC_REJECTED]: {
    subject: 'Verification Needs Attention',
    body: '<p>Please update your verification documents</p>',
    smsBody: 'Movra: Verification needs attention',
  },
  [NotificationType.LOGIN_ALERT]: {
    subject: 'New Login Detected',
    body: '<p>New login to your account from {{device}} at {{location}}</p>',
    smsBody: 'Movra: New login from {{device}}',
  },
};

export function renderTemplate(
  type: NotificationType,
  data: Record<string, string>,
  channel: 'email' | 'sms' = 'email'
): { subject: string; body: string } {
  const template = templates[type];
  if (!template) {
    return { subject: 'Notification', body: 'You have a new notification from Movra.' };
  }

  const body = channel === 'sms' && template.smsBody ? template.smsBody : template.body;

  return {
    subject: interpolate(template.subject, data),
    body: interpolate(body, data),
  };
}

function interpolate(text: string, data: Record<string, string>): string {
  return text.replace(/\{\{(\w+)\}\}/g, (_, key) => data[key] || '');
}

export { templates };
```

**Step 2: Commit**

```bash
git add services/notification-service/src/templates/
git commit -m "feat(notification): add template system for notifications"
```

---

## Task 3: Provider Interface and Implementations

**Files:**
- Create: `services/notification-service/src/providers/types.ts`
- Create: `services/notification-service/src/providers/emailProvider.ts`
- Create: `services/notification-service/src/providers/smsProvider.ts`
- Create: `services/notification-service/src/providers/index.ts`

**Step 1: Create provider types**

```typescript
// src/providers/types.ts
export interface SendResult {
  success: boolean;
  messageId?: string;
  error?: string;
}

export interface NotificationProvider {
  name: string;
  send(recipient: string, subject: string, body: string): Promise<SendResult>;
}
```

**Step 2: Create email provider**

```typescript
// src/providers/emailProvider.ts
import nodemailer from 'nodemailer';
import { config } from '../config';
import { NotificationProvider, SendResult } from './types';

export class EmailProvider implements NotificationProvider {
  name = 'email';
  private transporter: nodemailer.Transporter;

  constructor() {
    this.transporter = nodemailer.createTransport({
      host: config.smtp.host,
      port: config.smtp.port,
      secure: config.smtp.secure,
      auth: config.smtp.auth,
    });
  }

  async send(recipient: string, subject: string, body: string): Promise<SendResult> {
    try {
      const info = await this.transporter.sendMail({
        from: config.smtp.from,
        to: recipient,
        subject,
        html: body,
      });

      return {
        success: true,
        messageId: info.messageId,
      };
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }
}
```

**Step 3: Create SMS provider (simulated)**

```typescript
// src/providers/smsProvider.ts
import { config } from '../config';
import { NotificationProvider, SendResult } from './types';
import pino from 'pino';

const logger = pino({ name: 'sms-provider' });

export class SmsProvider implements NotificationProvider {
  name = 'sms';

  async send(recipient: string, _subject: string, body: string): Promise<SendResult> {
    // Simulated SMS sending - in production, integrate with Twilio/Nexmo/etc.
    if (config.sms.provider === 'simulated') {
      logger.info({ recipient, body: body.substring(0, 50) }, 'Simulated SMS sent');
      return {
        success: true,
        messageId: `SMS_${Date.now()}`,
      };
    }

    // Placeholder for real SMS provider integration
    // Example with Twilio:
    // const client = twilio(config.sms.accountSid, config.sms.authToken);
    // const message = await client.messages.create({
    //   body,
    //   from: config.sms.fromNumber,
    //   to: recipient,
    // });

    return {
      success: false,
      error: `SMS provider '${config.sms.provider}' not implemented`,
    };
  }
}
```

**Step 4: Create provider index**

```typescript
// src/providers/index.ts
export { EmailProvider } from './emailProvider';
export { SmsProvider } from './smsProvider';
export { NotificationProvider, SendResult } from './types';
```

**Step 5: Commit**

```bash
git add services/notification-service/src/providers/
git commit -m "feat(notification): add email and SMS provider implementations"
```

---

## Task 4: Notification Service Layer

**Files:**
- Create: `services/notification-service/src/services/notificationService.ts`

**Step 1: Create notification service**

```typescript
// src/services/notificationService.ts
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

    let result = Array.from(notifications.values())
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
```

**Step 2: Commit**

```bash
git add services/notification-service/src/services/
git commit -m "feat(notification): add NotificationService with business logic"
```

---

## Task 5: gRPC Server

**Files:**
- Create: `services/notification-service/src/grpc/server.ts`
- Create: `services/notification-service/src/grpc/types.ts`

**Step 1: Install gRPC dependencies**

Run: `cd services/notification-service && npm install @grpc/grpc-js @grpc/proto-loader`

**Step 2: Create gRPC types**

```typescript
// src/grpc/types.ts
// Placeholder types matching proto definition

export enum GrpcNotificationChannel {
  NOTIFICATION_CHANNEL_UNSPECIFIED = 0,
  NOTIFICATION_CHANNEL_EMAIL = 1,
  NOTIFICATION_CHANNEL_SMS = 2,
  NOTIFICATION_CHANNEL_PUSH = 3,
}

export enum GrpcNotificationType {
  NOTIFICATION_TYPE_UNSPECIFIED = 0,
  NOTIFICATION_TYPE_TRANSFER_CREATED = 1,
  NOTIFICATION_TYPE_TRANSFER_AWAITING_FUNDS = 2,
  NOTIFICATION_TYPE_TRANSFER_FUNDS_RECEIVED = 3,
  NOTIFICATION_TYPE_TRANSFER_PROCESSING = 4,
  NOTIFICATION_TYPE_TRANSFER_COMPLETED = 5,
  NOTIFICATION_TYPE_TRANSFER_FAILED = 6,
  NOTIFICATION_TYPE_TRANSFER_REFUNDED = 7,
  NOTIFICATION_TYPE_PICKUP_CODE_READY = 8,
  NOTIFICATION_TYPE_PICKUP_REMINDER = 9,
  NOTIFICATION_TYPE_PICKUP_COLLECTED = 10,
  NOTIFICATION_TYPE_WELCOME = 11,
  NOTIFICATION_TYPE_KYC_APPROVED = 12,
  NOTIFICATION_TYPE_KYC_REJECTED = 13,
  NOTIFICATION_TYPE_PASSWORD_RESET = 14,
  NOTIFICATION_TYPE_LOGIN_ALERT = 15,
}

export enum GrpcDeliveryStatus {
  DELIVERY_STATUS_UNSPECIFIED = 0,
  DELIVERY_STATUS_PENDING = 1,
  DELIVERY_STATUS_SENT = 2,
  DELIVERY_STATUS_DELIVERED = 3,
  DELIVERY_STATUS_FAILED = 4,
  DELIVERY_STATUS_BOUNCED = 5,
}
```

**Step 3: Create gRPC server**

```typescript
// src/grpc/server.ts
import * as grpc from '@grpc/grpc-js';
import pino from 'pino';
import { NotificationService } from '../services/notificationService';
import { NotificationChannel, NotificationType, DeliveryStatus, Notification } from '../types';
import { GrpcNotificationChannel, GrpcNotificationType, GrpcDeliveryStatus } from './types';

const logger = pino({ name: 'grpc-server' });

// Type converters
function grpcChannelToModel(channel: GrpcNotificationChannel): NotificationChannel {
  switch (channel) {
    case GrpcNotificationChannel.NOTIFICATION_CHANNEL_EMAIL:
      return NotificationChannel.EMAIL;
    case GrpcNotificationChannel.NOTIFICATION_CHANNEL_SMS:
      return NotificationChannel.SMS;
    default:
      return NotificationChannel.EMAIL;
  }
}

function grpcTypeToModel(type: GrpcNotificationType): NotificationType {
  const typeMap: Record<number, NotificationType> = {
    [GrpcNotificationType.NOTIFICATION_TYPE_TRANSFER_CREATED]: NotificationType.TRANSFER_CREATED,
    [GrpcNotificationType.NOTIFICATION_TYPE_TRANSFER_FUNDS_RECEIVED]: NotificationType.TRANSFER_FUNDS_RECEIVED,
    [GrpcNotificationType.NOTIFICATION_TYPE_TRANSFER_COMPLETED]: NotificationType.TRANSFER_COMPLETED,
    [GrpcNotificationType.NOTIFICATION_TYPE_TRANSFER_FAILED]: NotificationType.TRANSFER_FAILED,
    [GrpcNotificationType.NOTIFICATION_TYPE_PICKUP_CODE_READY]: NotificationType.PICKUP_CODE_READY,
    [GrpcNotificationType.NOTIFICATION_TYPE_WELCOME]: NotificationType.WELCOME,
    [GrpcNotificationType.NOTIFICATION_TYPE_PASSWORD_RESET]: NotificationType.PASSWORD_RESET,
  };
  return typeMap[type] || NotificationType.TRANSFER_CREATED;
}

function modelChannelToGrpc(channel: NotificationChannel): GrpcNotificationChannel {
  switch (channel) {
    case NotificationChannel.EMAIL:
      return GrpcNotificationChannel.NOTIFICATION_CHANNEL_EMAIL;
    case NotificationChannel.SMS:
      return GrpcNotificationChannel.NOTIFICATION_CHANNEL_SMS;
    default:
      return GrpcNotificationChannel.NOTIFICATION_CHANNEL_UNSPECIFIED;
  }
}

function modelStatusToGrpc(status: DeliveryStatus): GrpcDeliveryStatus {
  switch (status) {
    case DeliveryStatus.PENDING:
      return GrpcDeliveryStatus.DELIVERY_STATUS_PENDING;
    case DeliveryStatus.SENT:
      return GrpcDeliveryStatus.DELIVERY_STATUS_SENT;
    case DeliveryStatus.DELIVERED:
      return GrpcDeliveryStatus.DELIVERY_STATUS_DELIVERED;
    case DeliveryStatus.FAILED:
      return GrpcDeliveryStatus.DELIVERY_STATUS_FAILED;
    default:
      return GrpcDeliveryStatus.DELIVERY_STATUS_UNSPECIFIED;
  }
}

function notificationToProto(n: Notification): Record<string, unknown> {
  return {
    id: n.id,
    userId: n.userId,
    channel: modelChannelToGrpc(n.channel),
    type: n.type,
    status: modelStatusToGrpc(n.status),
    recipient: n.recipient,
    subject: n.subject || '',
    body: n.body,
    templateData: n.templateData,
    correlationId: n.correlationId || '',
    providerMessageId: n.providerMessageId || '',
    failureReason: n.failureReason || '',
    retryCount: n.retryCount,
    createdAt: { seconds: Math.floor(n.createdAt.getTime() / 1000), nanos: 0 },
    sentAt: n.sentAt ? { seconds: Math.floor(n.sentAt.getTime() / 1000), nanos: 0 } : null,
    deliveredAt: n.deliveredAt ? { seconds: Math.floor(n.deliveredAt.getTime() / 1000), nanos: 0 } : null,
  };
}

export function createGrpcServer(notificationService: NotificationService): grpc.Server {
  const server = new grpc.Server();

  // Define service handlers
  const handlers = {
    SendNotification: async (
      call: grpc.ServerUnaryCall<any, any>,
      callback: grpc.sendUnaryData<any>
    ) => {
      try {
        const req = call.request;
        const notification = await notificationService.sendNotification({
          userId: req.userId,
          channel: grpcChannelToModel(req.channel),
          type: grpcTypeToModel(req.type),
          recipient: req.recipient,
          templateData: req.templateData || {},
          correlationId: req.correlationId,
        });

        callback(null, { notification: notificationToProto(notification) });
      } catch (error) {
        logger.error({ error }, 'SendNotification failed');
        callback(null, {
          error: { code: 'SEND_FAILED', message: error instanceof Error ? error.message : 'Unknown error' },
        });
      }
    },

    GetNotification: async (
      call: grpc.ServerUnaryCall<any, any>,
      callback: grpc.sendUnaryData<any>
    ) => {
      try {
        const notification = await notificationService.getNotification(call.request.notificationId);
        if (!notification) {
          callback(null, { error: { code: 'NOT_FOUND', message: 'Notification not found' } });
          return;
        }
        callback(null, { notification: notificationToProto(notification) });
      } catch (error) {
        callback(null, {
          error: { code: 'GET_FAILED', message: error instanceof Error ? error.message : 'Unknown error' },
        });
      }
    },

    ListNotifications: async (
      call: grpc.ServerUnaryCall<any, any>,
      callback: grpc.sendUnaryData<any>
    ) => {
      try {
        const req = call.request;
        const notifications = await notificationService.listNotifications(req.userId, {
          channel: req.channelFilter ? grpcChannelToModel(req.channelFilter) : undefined,
          limit: req.pagination?.limit || 20,
          offset: req.pagination?.offset || 0,
        });

        callback(null, {
          notifications: notifications.map(notificationToProto),
          pagination: {
            total: notifications.length,
            limit: req.pagination?.limit || 20,
            offset: req.pagination?.offset || 0,
          },
        });
      } catch (error) {
        callback(null, {
          error: { code: 'LIST_FAILED', message: error instanceof Error ? error.message : 'Unknown error' },
        });
      }
    },

    ResendNotification: async (
      call: grpc.ServerUnaryCall<any, any>,
      callback: grpc.sendUnaryData<any>
    ) => {
      try {
        const notification = await notificationService.resendNotification(call.request.notificationId);
        if (!notification) {
          callback(null, { error: { code: 'NOT_FOUND', message: 'Notification not found' } });
          return;
        }
        callback(null, { notification: notificationToProto(notification) });
      } catch (error) {
        callback(null, {
          error: { code: 'RESEND_FAILED', message: error instanceof Error ? error.message : 'Unknown error' },
        });
      }
    },
  };

  // Add service (using dynamic service definition)
  const serviceDefinition: grpc.ServiceDefinition = {
    SendNotification: {
      path: '/movra.notification.NotificationService/SendNotification',
      requestStream: false,
      responseStream: false,
      requestSerialize: (value: any) => Buffer.from(JSON.stringify(value)),
      requestDeserialize: (value: Buffer) => JSON.parse(value.toString()),
      responseSerialize: (value: any) => Buffer.from(JSON.stringify(value)),
      responseDeserialize: (value: Buffer) => JSON.parse(value.toString()),
    },
    GetNotification: {
      path: '/movra.notification.NotificationService/GetNotification',
      requestStream: false,
      responseStream: false,
      requestSerialize: (value: any) => Buffer.from(JSON.stringify(value)),
      requestDeserialize: (value: Buffer) => JSON.parse(value.toString()),
      responseSerialize: (value: any) => Buffer.from(JSON.stringify(value)),
      responseDeserialize: (value: Buffer) => JSON.parse(value.toString()),
    },
    ListNotifications: {
      path: '/movra.notification.NotificationService/ListNotifications',
      requestStream: false,
      responseStream: false,
      requestSerialize: (value: any) => Buffer.from(JSON.stringify(value)),
      requestDeserialize: (value: Buffer) => JSON.parse(value.toString()),
      responseSerialize: (value: any) => Buffer.from(JSON.stringify(value)),
      responseDeserialize: (value: Buffer) => JSON.parse(value.toString()),
    },
    ResendNotification: {
      path: '/movra.notification.NotificationService/ResendNotification',
      requestStream: false,
      responseStream: false,
      requestSerialize: (value: any) => Buffer.from(JSON.stringify(value)),
      requestDeserialize: (value: Buffer) => JSON.parse(value.toString()),
      responseSerialize: (value: any) => Buffer.from(JSON.stringify(value)),
      responseDeserialize: (value: Buffer) => JSON.parse(value.toString()),
    },
  };

  server.addService(serviceDefinition, handlers);

  return server;
}
```

**Step 4: Commit**

```bash
git add services/notification-service/src/grpc/ services/notification-service/package.json services/notification-service/package-lock.json
git commit -m "feat(notification): add gRPC server with 4 RPCs"
```

---

## Task 6: Enhanced Kafka Consumer

**Files:**
- Create: `services/notification-service/src/kafka/consumer.ts`

**Step 1: Create enhanced Kafka consumer**

```typescript
// src/kafka/consumer.ts
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
```

**Step 2: Commit**

```bash
git add services/notification-service/src/kafka/
git commit -m "feat(notification): add enhanced Kafka consumer with event handling"
```

---

## Task 7: Wire Everything in index.ts

**Files:**
- Modify: `services/notification-service/src/index.ts`

**Step 1: Update index.ts**

Replace entire file:

```typescript
// src/index.ts
import express from 'express';
import pinoHttp from 'pino-http';
import pino from 'pino';
import * as grpc from '@grpc/grpc-js';
import { Registry, collectDefaultMetrics } from 'prom-client';
import { config } from './config';
import { NotificationService } from './services/notificationService';
import { createGrpcServer } from './grpc/server';
import { KafkaNotificationConsumer } from './kafka/consumer';
import { NotificationChannel, NotificationType } from './types';

const logger = pino({
  level: process.env.NODE_ENV === 'production' ? 'info' : 'debug',
});

// Prometheus metrics
const register = new Registry();
collectDefaultMetrics({ register });

// Create services
const notificationService = new NotificationService();

// Express app for HTTP endpoints
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

// REST API endpoints (for direct HTTP access)
app.post('/api/notifications', async (req, res) => {
  try {
    const { userId, channel, type, recipient, templateData, correlationId } = req.body;

    const notification = await notificationService.sendNotification({
      userId,
      channel: channel as NotificationChannel,
      type: type as NotificationType,
      recipient,
      templateData: templateData || {},
      correlationId,
    });

    res.status(201).json(notification);
  } catch (error) {
    logger.error({ error }, 'Failed to send notification');
    res.status(500).json({ error: error instanceof Error ? error.message : 'Unknown error' });
  }
});

app.get('/api/notifications/:id', async (req, res) => {
  try {
    const notification = await notificationService.getNotification(req.params.id);
    if (!notification) {
      res.status(404).json({ error: 'Notification not found' });
      return;
    }
    res.json(notification);
  } catch (error) {
    res.status(500).json({ error: error instanceof Error ? error.message : 'Unknown error' });
  }
});

app.get('/api/users/:userId/notifications', async (req, res) => {
  try {
    const { channel, type, limit, offset } = req.query;
    const notifications = await notificationService.listNotifications(req.params.userId, {
      channel: channel as NotificationChannel | undefined,
      type: type as NotificationType | undefined,
      limit: limit ? parseInt(limit as string, 10) : undefined,
      offset: offset ? parseInt(offset as string, 10) : undefined,
    });
    res.json({ notifications });
  } catch (error) {
    res.status(500).json({ error: error instanceof Error ? error.message : 'Unknown error' });
  }
});

app.post('/api/notifications/:id/resend', async (req, res) => {
  try {
    const notification = await notificationService.resendNotification(req.params.id);
    if (!notification) {
      res.status(404).json({ error: 'Notification not found' });
      return;
    }
    res.json(notification);
  } catch (error) {
    res.status(500).json({ error: error instanceof Error ? error.message : 'Unknown error' });
  }
});

// Create gRPC server
const grpcServer = createGrpcServer(notificationService);

// Create Kafka consumer
const kafkaConsumer = new KafkaNotificationConsumer(notificationService);

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
  } catch (error) {
    logger.error({ error }, 'Failed to start Kafka consumer');
  }

  logger.info(
    { httpPort: config.httpPort, grpcPort: config.grpcPort },
    'Notification Service started'
  );
}

// Graceful shutdown
process.on('SIGTERM', async () => {
  logger.info('SIGTERM received, shutting down');
  await kafkaConsumer.stop();
  grpcServer.forceShutdown();
  process.exit(0);
});

start().catch((error) => {
  logger.error({ error }, 'Failed to start service');
  process.exit(1);
});
```

**Step 2: Update package.json with new dependencies**

Run: `cd services/notification-service && npm install @grpc/grpc-js`

**Step 3: Verify TypeScript compiles**

Run: `cd services/notification-service && npx tsc --noEmit`

**Step 4: Commit**

```bash
git add services/notification-service/src/index.ts services/notification-service/package.json services/notification-service/package-lock.json
git commit -m "feat(notification): wire all components in index.ts"
```

---

## Task 8: Unit Tests

**Files:**
- Create: `services/notification-service/src/__tests__/templates.test.ts`
- Create: `services/notification-service/src/__tests__/notificationService.test.ts`
- Create: `services/notification-service/jest.config.js`

**Step 1: Install test dependencies**

Run: `cd services/notification-service && npm install -D jest @types/jest ts-jest`

**Step 2: Create Jest config**

```javascript
// jest.config.js
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  testMatch: ['**/__tests__/**/*.test.ts'],
  moduleFileExtensions: ['ts', 'js'],
  collectCoverageFrom: ['src/**/*.ts', '!src/**/*.d.ts'],
};
```

**Step 3: Create template tests**

```typescript
// src/__tests__/templates.test.ts
import { renderTemplate } from '../templates';
import { NotificationType } from '../types';

describe('Template System', () => {
  describe('renderTemplate', () => {
    it('should render TRANSFER_COMPLETED template with data', () => {
      const result = renderTemplate(NotificationType.TRANSFER_COMPLETED, {
        senderName: 'John Doe',
        recipientName: 'Jane Smith',
        amount: '100.00',
        currency: 'SGD',
        targetCurrency: 'PHP',
        recipientAmount: '3,975.00',
        transferId: 'TRF123',
      });

      expect(result.subject).toContain('100.00');
      expect(result.subject).toContain('SGD');
      expect(result.body).toContain('John Doe');
      expect(result.body).toContain('Jane Smith');
      expect(result.body).toContain('TRF123');
    });

    it('should render SMS template when channel is sms', () => {
      const result = renderTemplate(
        NotificationType.TRANSFER_COMPLETED,
        {
          recipientName: 'Jane',
          recipientAmount: '3,975.00',
          targetCurrency: 'PHP',
          transferId: 'TRF123',
        },
        'sms'
      );

      expect(result.body).not.toContain('<h2>');
      expect(result.body).toContain('Movra:');
      expect(result.body.length).toBeLessThan(200);
    });

    it('should handle missing template data gracefully', () => {
      const result = renderTemplate(NotificationType.WELCOME, {});

      expect(result.subject).toBe('Welcome to Movra!');
      expect(result.body).toContain('Welcome');
    });

    it('should render PICKUP_CODE_READY with pickup code', () => {
      const result = renderTemplate(NotificationType.PICKUP_CODE_READY, {
        recipientName: 'Maria Garcia',
        amount: '5,000.00',
        currency: 'PHP',
        pickupCode: '12345678',
        expiresAt: '2024-01-08 18:00',
      });

      expect(result.body).toContain('12345678');
      expect(result.body).toContain('Maria Garcia');
      expect(result.body).toContain('5,000.00');
    });
  });
});
```

**Step 4: Create service tests**

```typescript
// src/__tests__/notificationService.test.ts
import { NotificationService } from '../services/notificationService';
import { NotificationChannel, NotificationType, DeliveryStatus } from '../types';

describe('NotificationService', () => {
  let service: NotificationService;

  beforeEach(() => {
    service = new NotificationService();
  });

  describe('sendNotification', () => {
    it('should create and send email notification', async () => {
      const notification = await service.sendNotification({
        userId: 'user_123',
        channel: NotificationChannel.EMAIL,
        type: NotificationType.WELCOME,
        recipient: 'test@example.com',
        templateData: { userName: 'Test User' },
      });

      expect(notification.id).toMatch(/^notif_/);
      expect(notification.userId).toBe('user_123');
      expect(notification.channel).toBe(NotificationChannel.EMAIL);
      expect(notification.type).toBe(NotificationType.WELCOME);
      expect(notification.recipient).toBe('test@example.com');
      expect(notification.body).toContain('Test User');
    });

    it('should create SMS notification', async () => {
      const notification = await service.sendNotification({
        userId: 'user_456',
        channel: NotificationChannel.SMS,
        type: NotificationType.PICKUP_CODE_READY,
        recipient: '+15551234567',
        templateData: {
          pickupCode: '87654321',
          amount: '1000',
          currency: 'PHP',
        },
      });

      expect(notification.channel).toBe(NotificationChannel.SMS);
      expect(notification.body).toContain('87654321');
    });

    it('should include correlation ID when provided', async () => {
      const notification = await service.sendNotification({
        userId: 'user_789',
        channel: NotificationChannel.EMAIL,
        type: NotificationType.TRANSFER_COMPLETED,
        recipient: 'test@example.com',
        templateData: {},
        correlationId: 'transfer_abc123',
      });

      expect(notification.correlationId).toBe('transfer_abc123');
    });
  });

  describe('getNotification', () => {
    it('should retrieve sent notification', async () => {
      const sent = await service.sendNotification({
        userId: 'user_get',
        channel: NotificationChannel.EMAIL,
        type: NotificationType.WELCOME,
        recipient: 'get@example.com',
        templateData: {},
      });

      const retrieved = await service.getNotification(sent.id);

      expect(retrieved).toBeDefined();
      expect(retrieved?.id).toBe(sent.id);
    });

    it('should return null for non-existent notification', async () => {
      const result = await service.getNotification('notif_nonexistent');
      expect(result).toBeNull();
    });
  });

  describe('listNotifications', () => {
    it('should list notifications for user', async () => {
      const userId = 'user_list_test';

      await service.sendNotification({
        userId,
        channel: NotificationChannel.EMAIL,
        type: NotificationType.WELCOME,
        recipient: 'list@example.com',
        templateData: {},
      });

      await service.sendNotification({
        userId,
        channel: NotificationChannel.SMS,
        type: NotificationType.PICKUP_CODE_READY,
        recipient: '+15551234567',
        templateData: {},
      });

      const notifications = await service.listNotifications(userId);

      expect(notifications.length).toBe(2);
      expect(notifications.every((n) => n.userId === userId)).toBe(true);
    });

    it('should filter by channel', async () => {
      const userId = 'user_filter_test';

      await service.sendNotification({
        userId,
        channel: NotificationChannel.EMAIL,
        type: NotificationType.WELCOME,
        recipient: 'filter@example.com',
        templateData: {},
      });

      await service.sendNotification({
        userId,
        channel: NotificationChannel.SMS,
        type: NotificationType.WELCOME,
        recipient: '+15551234567',
        templateData: {},
      });

      const emailOnly = await service.listNotifications(userId, {
        channel: NotificationChannel.EMAIL,
      });

      expect(emailOnly.length).toBe(1);
      expect(emailOnly[0].channel).toBe(NotificationChannel.EMAIL);
    });
  });
});
```

**Step 5: Run tests**

Run: `cd services/notification-service && npm test`

**Step 6: Commit**

```bash
git add services/notification-service/jest.config.js services/notification-service/src/__tests__/
git commit -m "test(notification): add unit tests for templates and service"
```

---

## Success Criteria

- [ ] TypeScript compiles without errors: `npx tsc --noEmit`
- [ ] All tests pass: `npm test`
- [ ] Service starts: `npm run dev`
- [ ] HTTP health endpoint works: `curl localhost:3002/health`
- [ ] gRPC server listening on port 50053
- [ ] 8 atomic commits created
