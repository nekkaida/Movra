import * as grpc from '@grpc/grpc-js';

export interface ApiError {
  status: number;
  code: string;
  message: string;
}

export function mapGrpcError(error: grpc.ServiceError): ApiError {
  switch (error.code) {
    case grpc.status.NOT_FOUND:
      return { status: 404, code: 'NOT_FOUND', message: error.details || 'Resource not found' };
    case grpc.status.INVALID_ARGUMENT:
      return { status: 400, code: 'INVALID_ARGUMENT', message: error.details || 'Invalid request' };
    case grpc.status.PERMISSION_DENIED:
      return { status: 403, code: 'PERMISSION_DENIED', message: error.details || 'Permission denied' };
    case grpc.status.UNAUTHENTICATED:
      return { status: 401, code: 'UNAUTHENTICATED', message: error.details || 'Authentication required' };
    case grpc.status.UNAVAILABLE:
      return { status: 503, code: 'SERVICE_UNAVAILABLE', message: 'Service temporarily unavailable' };
    case grpc.status.DEADLINE_EXCEEDED:
      return { status: 504, code: 'TIMEOUT', message: 'Request timed out' };
    case grpc.status.RESOURCE_EXHAUSTED:
      return { status: 429, code: 'RATE_LIMITED', message: 'Too many requests' };
    default:
      return { status: 500, code: 'INTERNAL_ERROR', message: 'An unexpected error occurred' };
  }
}

export function isGrpcError(error: unknown): error is grpc.ServiceError {
  return error !== null && typeof error === 'object' && 'code' in error && 'details' in error;
}
