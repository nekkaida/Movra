import * as grpc from '@grpc/grpc-js';
import { loadProto, createClient } from './protoLoader';
import { config } from '../config';
import { logger } from '../utils/logger';

const paymentProto = loadProto('payment.proto');

export interface Money {
  currency: string;
  amount: string;
}

export interface RequestMetadata {
  correlationId: string;
  userId: string;
  kycLevel: string;
}

export interface Transfer {
  id: string;
  userId: string;
  idempotencyKey: string;
  status: string;
  sourceAmount: Money;
  targetAmount: Money;
  fee: Money;
  exchangeRate: string;
  rateLockId: string;
  fundingMethod: string;
  payoutMethod: string;
  recipientId: string;
  fundingDetails?: {
    bankName: string;
    accountNumber: string;
    accountName: string;
    reference: string;
  };
  createdAt: { seconds: string; nanos: number };
  updatedAt: { seconds: string; nanos: number };
}

export interface CreateTransferResponse {
  transfer: Transfer;
  error?: { code: string; message: string };
}

export interface GetTransferResponse {
  transfer: Transfer;
  error?: { code: string; message: string };
}

export interface ListTransfersResponse {
  transfers: Transfer[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalItems: number;
  };
  error?: { code: string; message: string };
}

interface PaymentServiceClient {
  createTransfer(
    request: {
      metadata: RequestMetadata;
      idempotencyKey: string;
      sourceAmount: Money;
      targetCurrency: string;
      fundingMethod: string;
      recipientId: string;
      rateLockId?: string;
    },
    callback: (error: grpc.ServiceError | null, response: CreateTransferResponse) => void
  ): void;
  getTransfer(
    request: { metadata: RequestMetadata; transferId: string },
    callback: (error: grpc.ServiceError | null, response: GetTransferResponse) => void
  ): void;
  confirmTransfer(
    request: { metadata: RequestMetadata; transferId: string },
    callback: (error: grpc.ServiceError | null, response: GetTransferResponse) => void
  ): void;
  cancelTransfer(
    request: { metadata: RequestMetadata; transferId: string; reason: string },
    callback: (error: grpc.ServiceError | null, response: GetTransferResponse) => void
  ): void;
  listTransfers(
    request: {
      metadata: RequestMetadata;
      pagination: { page: number; pageSize: number };
      statusFilter?: string;
    },
    callback: (error: grpc.ServiceError | null, response: ListTransfersResponse) => void
  ): void;
}

let client: PaymentServiceClient | null = null;

function getClient(): PaymentServiceClient {
  if (!client) {
    client = createClient<PaymentServiceClient>(
      paymentProto,
      'movra.payment',
      'PaymentService',
      config.services.payment
    );
  }
  return client;
}

export function buildMetadata(userId: string, kycLevel: string, correlationId: string): RequestMetadata {
  return { correlationId, userId, kycLevel };
}

export async function createTransfer(
  metadata: RequestMetadata,
  idempotencyKey: string,
  sourceAmount: Money,
  targetCurrency: string,
  fundingMethod: string,
  recipientId: string,
  rateLockId?: string
): Promise<CreateTransferResponse> {
  return new Promise((resolve, reject) => {
    getClient().createTransfer(
      { metadata, idempotencyKey, sourceAmount, targetCurrency, fundingMethod, recipientId, rateLockId },
      (error, response) => {
        if (error) {
          logger.error({ error }, 'Payment gRPC error: createTransfer');
          reject(error);
        } else {
          resolve(response);
        }
      }
    );
  });
}

export async function getTransfer(
  metadata: RequestMetadata,
  transferId: string
): Promise<GetTransferResponse> {
  return new Promise((resolve, reject) => {
    getClient().getTransfer({ metadata, transferId }, (error, response) => {
      if (error) {
        logger.error({ error }, 'Payment gRPC error: getTransfer');
        reject(error);
      } else {
        resolve(response);
      }
    });
  });
}

export async function confirmTransfer(
  metadata: RequestMetadata,
  transferId: string
): Promise<GetTransferResponse> {
  return new Promise((resolve, reject) => {
    getClient().confirmTransfer({ metadata, transferId }, (error, response) => {
      if (error) {
        logger.error({ error }, 'Payment gRPC error: confirmTransfer');
        reject(error);
      } else {
        resolve(response);
      }
    });
  });
}

export async function cancelTransfer(
  metadata: RequestMetadata,
  transferId: string,
  reason: string
): Promise<GetTransferResponse> {
  return new Promise((resolve, reject) => {
    getClient().cancelTransfer({ metadata, transferId, reason }, (error, response) => {
      if (error) {
        logger.error({ error }, 'Payment gRPC error: cancelTransfer');
        reject(error);
      } else {
        resolve(response);
      }
    });
  });
}

export async function listTransfers(
  metadata: RequestMetadata,
  page: number,
  pageSize: number,
  statusFilter?: string
): Promise<ListTransfersResponse> {
  return new Promise((resolve, reject) => {
    getClient().listTransfers(
      { metadata, pagination: { page, pageSize }, statusFilter },
      (error, response) => {
        if (error) {
          logger.error({ error }, 'Payment gRPC error: listTransfers');
          reject(error);
        } else {
          resolve(response);
        }
      }
    );
  });
}
