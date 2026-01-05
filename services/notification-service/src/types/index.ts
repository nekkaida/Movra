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
