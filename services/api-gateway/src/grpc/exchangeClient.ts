import * as grpc from '@grpc/grpc-js';
import { loadProto, createClient } from './protoLoader';
import { config } from '../config';
import { logger } from '../utils/logger';

const exchangeProto = loadProto('exchange.proto');

export interface ExchangeRate {
  sourceCurrency: string;
  targetCurrency: string;
  rate: string;
  buyRate: string;
  marginPercentage: string;
  fetchedAt: { seconds: string; nanos: number };
  expiresAt: { seconds: string; nanos: number };
}

export interface LockedRate {
  lockId: string;
  rate: ExchangeRate;
  lockedAt: { seconds: string; nanos: number };
  expiresAt: { seconds: string; nanos: number };
  expired: boolean;
}

export interface Corridor {
  sourceCurrency: string;
  targetCurrency: string;
  enabled: boolean;
  feePercentage: string;
  feeMinimum: { currency: string; amount: string };
  marginPercentage: string;
  payoutMethods: string[];
}

export interface GetRateResponse {
  rate: ExchangeRate;
  error?: { code: string; message: string };
}

export interface LockRateResponse {
  lockedRate: LockedRate;
  error?: { code: string; message: string };
}

export interface GetLockedRateResponse {
  lockedRate: LockedRate;
  error?: { code: string; message: string };
}

export interface GetCorridorsResponse {
  corridors: Corridor[];
  error?: { code: string; message: string };
}

interface ExchangeServiceClient {
  getRate(
    request: { sourceCurrency: string; targetCurrency: string },
    callback: (error: grpc.ServiceError | null, response: GetRateResponse) => void
  ): void;
  lockRate(
    request: { sourceCurrency: string; targetCurrency: string; lockDurationSeconds: number },
    callback: (error: grpc.ServiceError | null, response: LockRateResponse) => void
  ): void;
  getLockedRate(
    request: { lockId: string },
    callback: (error: grpc.ServiceError | null, response: GetLockedRateResponse) => void
  ): void;
  getCorridors(
    request: { sourceCurrency?: string },
    callback: (error: grpc.ServiceError | null, response: GetCorridorsResponse) => void
  ): void;
}

let client: ExchangeServiceClient | null = null;

function getClient(): ExchangeServiceClient {
  if (!client) {
    client = createClient<ExchangeServiceClient>(
      exchangeProto,
      'movra.exchange',
      'ExchangeRateService',
      config.services.exchangeRate
    );
  }
  return client;
}

export async function getRate(sourceCurrency: string, targetCurrency: string): Promise<GetRateResponse> {
  return new Promise((resolve, reject) => {
    getClient().getRate({ sourceCurrency, targetCurrency }, (error, response) => {
      if (error) {
        logger.error({ error }, 'Exchange gRPC error: getRate');
        reject(error);
      } else {
        resolve(response);
      }
    });
  });
}

export async function lockRate(
  sourceCurrency: string,
  targetCurrency: string,
  durationSeconds: number = 30
): Promise<LockRateResponse> {
  return new Promise((resolve, reject) => {
    getClient().lockRate(
      { sourceCurrency, targetCurrency, lockDurationSeconds: durationSeconds },
      (error, response) => {
        if (error) {
          logger.error({ error }, 'Exchange gRPC error: lockRate');
          reject(error);
        } else {
          resolve(response);
        }
      }
    );
  });
}

export async function getLockedRate(lockId: string): Promise<GetLockedRateResponse> {
  return new Promise((resolve, reject) => {
    getClient().getLockedRate({ lockId }, (error, response) => {
      if (error) {
        logger.error({ error }, 'Exchange gRPC error: getLockedRate');
        reject(error);
      } else {
        resolve(response);
      }
    });
  });
}

export async function getCorridors(sourceCurrency?: string): Promise<GetCorridorsResponse> {
  return new Promise((resolve, reject) => {
    getClient().getCorridors({ sourceCurrency }, (error, response) => {
      if (error) {
        logger.error({ error }, 'Exchange gRPC error: getCorridors');
        reject(error);
      } else {
        resolve(response);
      }
    });
  });
}
