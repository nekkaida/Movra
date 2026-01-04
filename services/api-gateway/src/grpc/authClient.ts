import * as grpc from '@grpc/grpc-js';
import { loadProto, createClient } from './protoLoader';
import { config } from '../config';
import { logger } from '../utils/logger';

// Load proto
const authProto = loadProto('auth.proto');

// Define types based on proto
export interface VerifyTokenRequest {
  token: string;
}

export interface VerifyTokenResponse {
  valid: boolean;
  userId: string;
  kycLevel: string;
  expiresAt: { seconds: string; nanos: number };
  error?: { code: string; message: string };
}

export interface User {
  id: string;
  email: string;
  phone: string;
  firstName: string;
  lastName: string;
  kycLevel: string;
  createdAt: { seconds: string; nanos: number };
  updatedAt: { seconds: string; nanos: number };
}

export interface GetUserResponse {
  user: User;
  error?: { code: string; message: string };
}

interface AuthServiceClient {
  verifyToken(
    request: VerifyTokenRequest,
    callback: (error: grpc.ServiceError | null, response: VerifyTokenResponse) => void
  ): void;
  getUser(
    request: { userId: string },
    callback: (error: grpc.ServiceError | null, response: GetUserResponse) => void
  ): void;
}

let client: AuthServiceClient | null = null;

function getClient(): AuthServiceClient {
  if (!client) {
    client = createClient<AuthServiceClient>(
      authProto,
      'movra.auth',
      'AuthService',
      config.services.auth
    );
  }
  return client;
}

export async function verifyToken(token: string): Promise<VerifyTokenResponse> {
  return new Promise((resolve, reject) => {
    getClient().verifyToken({ token }, (error, response) => {
      if (error) {
        logger.error({ error }, 'Auth gRPC error: verifyToken');
        reject(error);
      } else {
        resolve(response);
      }
    });
  });
}

export async function getUser(userId: string): Promise<GetUserResponse> {
  return new Promise((resolve, reject) => {
    getClient().getUser({ userId }, (error, response) => {
      if (error) {
        logger.error({ error }, 'Auth gRPC error: getUser');
        reject(error);
      } else {
        resolve(response);
      }
    });
  });
}
