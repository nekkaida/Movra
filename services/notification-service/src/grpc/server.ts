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
    [GrpcNotificationType.NOTIFICATION_TYPE_KYC_APPROVED]: NotificationType.KYC_APPROVED,
    [GrpcNotificationType.NOTIFICATION_TYPE_KYC_REJECTED]: NotificationType.KYC_REJECTED,
    [GrpcNotificationType.NOTIFICATION_TYPE_LOGIN_ALERT]: NotificationType.LOGIN_ALERT,
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
  const handlers: grpc.UntypedServiceImplementation = {
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
