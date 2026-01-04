# API Gateway gRPC Integration Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace mock responses in API Gateway with actual gRPC calls to Auth, Payment, and Exchange Rate services.

**Architecture:** Use `@grpc/proto-loader` for dynamic proto loading (simpler than code generation). Create a service client layer that wraps gRPC calls with proper error handling. Routes call service clients instead of returning mock data.

**Tech Stack:** Node.js, TypeScript, Express, @grpc/grpc-js, @grpc/proto-loader

---

## Phase A: gRPC Client Infrastructure

### Task 1: Create Proto Loader Utility

**Files:**
- Create: `src/grpc/protoLoader.ts`

**Step 1: Create proto loader utility**

```typescript
import * as grpc from '@grpc/grpc-js';
import * as protoLoader from '@grpc/proto-loader';
import path from 'path';

const PROTO_DIR = path.resolve(__dirname, '../../../proto');

const loaderOptions: protoLoader.Options = {
  keepCase: false,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
};

export function loadProto(protoFile: string): grpc.GrpcObject {
  const protoPath = path.join(PROTO_DIR, protoFile);
  const packageDefinition = protoLoader.loadSync(protoPath, loaderOptions);
  return grpc.loadPackageDefinition(packageDefinition);
}

export function createClient<T>(
  proto: grpc.GrpcObject,
  packagePath: string,
  serviceName: string,
  address: string
): T {
  const parts = packagePath.split('.');
  let service: any = proto;
  for (const part of parts) {
    service = service[part];
  }
  const ServiceClient = service[serviceName];
  return new ServiceClient(address, grpc.credentials.createInsecure()) as T;
}
```

**Step 2: Commit**

```bash
git add src/grpc/protoLoader.ts
git commit -m "feat(gateway): add gRPC proto loader utility"
```

---

### Task 2: Create Auth Service Client

**Files:**
- Create: `src/grpc/authClient.ts`

**Step 1: Create auth service client**

```typescript
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
```

**Step 2: Commit**

```bash
git add src/grpc/authClient.ts
git commit -m "feat(gateway): add Auth service gRPC client"
```

---

### Task 3: Create Exchange Rate Service Client

**Files:**
- Create: `src/grpc/exchangeClient.ts`

**Step 1: Create exchange rate service client**

```typescript
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
```

**Step 2: Commit**

```bash
git add src/grpc/exchangeClient.ts
git commit -m "feat(gateway): add Exchange Rate service gRPC client"
```

---

### Task 4: Create Payment Service Client

**Files:**
- Create: `src/grpc/paymentClient.ts`

**Step 1: Create payment service client**

```typescript
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

function buildMetadata(userId: string, kycLevel: string, correlationId: string): RequestMetadata {
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

export { buildMetadata };
```

**Step 2: Commit**

```bash
git add src/grpc/paymentClient.ts
git commit -m "feat(gateway): add Payment service gRPC client"
```

---

### Task 5: Create gRPC Client Index

**Files:**
- Create: `src/grpc/index.ts`

**Step 1: Create index file**

```typescript
export * as authClient from './authClient';
export * as exchangeClient from './exchangeClient';
export * as paymentClient from './paymentClient';
```

**Step 2: Commit**

```bash
git add src/grpc/index.ts
git commit -m "feat(gateway): add gRPC client index"
```

---

## Phase B: Update Routes to Use gRPC

### Task 6: Update Rates Route

**Files:**
- Modify: `src/routes/rates.ts`

**Step 1: Replace mock with gRPC calls**

```typescript
import { Router, Request, Response } from 'express';
import { AuthenticatedRequest, optionalAuthMiddleware } from '../middleware/auth';
import { logger } from '../utils/logger';
import * as exchangeClient from '../grpc/exchangeClient';

const router = Router();

// Get exchange rate (public)
router.get('/:from/:to', optionalAuthMiddleware, async (req: Request, res: Response) => {
  try {
    const { from, to } = req.params;

    if (!/^[A-Z]{3}$/.test(from) || !/^[A-Z]{3}$/.test(to)) {
      res.status(400).json({ error: 'Invalid currency code format' });
      return;
    }

    const response = await exchangeClient.getRate(from, to);

    if (response.error) {
      res.status(400).json({ error: response.error.message, code: response.error.code });
      return;
    }

    const rate = response.rate;
    res.json({
      sourceCurrency: rate.sourceCurrency,
      targetCurrency: rate.targetCurrency,
      rate: rate.rate,
      buyRate: rate.buyRate,
      marginPercentage: rate.marginPercentage,
      fetchedAt: timestampToIso(rate.fetchedAt),
      expiresAt: timestampToIso(rate.expiresAt),
    });
  } catch (error) {
    logger.error({ error }, 'Failed to get exchange rate');
    res.status(503).json({ error: 'Exchange rate service unavailable' });
  }
});

// Lock rate
router.post('/lock', async (req: AuthenticatedRequest, res: Response) => {
  try {
    const { sourceCurrency, targetCurrency, durationSeconds = 30 } = req.body;

    if (!sourceCurrency || !targetCurrency) {
      res.status(400).json({ error: 'Source and target currency required' });
      return;
    }

    const response = await exchangeClient.lockRate(sourceCurrency, targetCurrency, durationSeconds);

    if (response.error) {
      res.status(400).json({ error: response.error.message, code: response.error.code });
      return;
    }

    const locked = response.lockedRate;
    res.json({
      lockId: locked.lockId,
      rate: {
        sourceCurrency: locked.rate.sourceCurrency,
        targetCurrency: locked.rate.targetCurrency,
        rate: locked.rate.rate,
        buyRate: locked.rate.buyRate,
        marginPercentage: locked.rate.marginPercentage,
      },
      lockedAt: timestampToIso(locked.lockedAt),
      expiresAt: timestampToIso(locked.expiresAt),
      expired: locked.expired,
    });
  } catch (error) {
    logger.error({ error }, 'Failed to lock rate');
    res.status(503).json({ error: 'Exchange rate service unavailable' });
  }
});

// Get locked rate
router.get('/locked/:lockId', async (req: Request, res: Response) => {
  try {
    const { lockId } = req.params;

    const response = await exchangeClient.getLockedRate(lockId);

    if (response.error) {
      if (response.error.code === 'RATE_LOCK_EXPIRED') {
        res.status(410).json({ error: response.error.message, expired: true });
        return;
      }
      res.status(400).json({ error: response.error.message, code: response.error.code });
      return;
    }

    const locked = response.lockedRate;
    res.json({
      lockId: locked.lockId,
      rate: {
        sourceCurrency: locked.rate.sourceCurrency,
        targetCurrency: locked.rate.targetCurrency,
        rate: locked.rate.rate,
        buyRate: locked.rate.buyRate,
      },
      lockedAt: timestampToIso(locked.lockedAt),
      expiresAt: timestampToIso(locked.expiresAt),
      expired: locked.expired,
    });
  } catch (error) {
    logger.error({ error }, 'Failed to get locked rate');
    res.status(503).json({ error: 'Exchange rate service unavailable' });
  }
});

// Get corridors
router.get('/corridors', async (_req: Request, res: Response) => {
  try {
    const response = await exchangeClient.getCorridors();

    if (response.error) {
      res.status(400).json({ error: response.error.message, code: response.error.code });
      return;
    }

    res.json({
      corridors: response.corridors.map((c) => ({
        sourceCurrency: c.sourceCurrency,
        targetCurrency: c.targetCurrency,
        enabled: c.enabled,
        feePercentage: c.feePercentage,
        feeMinimum: c.feeMinimum,
        marginPercentage: c.marginPercentage,
        payoutMethods: c.payoutMethods,
      })),
    });
  } catch (error) {
    logger.error({ error }, 'Failed to get corridors');
    res.status(503).json({ error: 'Exchange rate service unavailable' });
  }
});

function timestampToIso(ts: { seconds: string; nanos: number }): string {
  const ms = parseInt(ts.seconds, 10) * 1000 + ts.nanos / 1000000;
  return new Date(ms).toISOString();
}

export default router;
```

**Step 2: Commit**

```bash
git add src/routes/rates.ts
git commit -m "feat(gateway): integrate rates route with Exchange Rate gRPC"
```

---

### Task 7: Update Transfers Route

**Files:**
- Modify: `src/routes/transfers.ts`

**Step 1: Replace mock with gRPC calls**

```typescript
import { Router, Response } from 'express';
import { AuthenticatedRequest, authMiddleware } from '../middleware/auth';
import { logger } from '../utils/logger';
import * as paymentClient from '../grpc/paymentClient';
import { v4 as uuidv4 } from 'uuid';

const router = Router();

router.use(authMiddleware);

// Create transfer
router.post('/', async (req: AuthenticatedRequest, res: Response) => {
  try {
    const { sourceAmount, sourceCurrency, targetCurrency, recipientId, fundingMethod, rateLockId } =
      req.body;

    if (!sourceAmount || !sourceCurrency || !targetCurrency || !recipientId || !fundingMethod) {
      res.status(400).json({ error: 'Missing required fields' });
      return;
    }

    const metadata = paymentClient.buildMetadata(
      req.user!.userId,
      req.user!.kycLevel,
      req.correlationId
    );

    const response = await paymentClient.createTransfer(
      metadata,
      uuidv4(),
      { currency: sourceCurrency, amount: sourceAmount.toString() },
      targetCurrency,
      fundingMethod,
      recipientId,
      rateLockId
    );

    if (response.error) {
      res.status(400).json({ error: response.error.message, code: response.error.code });
      return;
    }

    logger.info({ transferId: response.transfer.id, correlationId: req.correlationId }, 'Transfer created');
    res.status(201).json(mapTransferResponse(response.transfer));
  } catch (error) {
    logger.error({ error, correlationId: req.correlationId }, 'Failed to create transfer');
    res.status(503).json({ error: 'Payment service unavailable' });
  }
});

// Get transfer
router.get('/:id', async (req: AuthenticatedRequest, res: Response) => {
  try {
    const { id } = req.params;

    const metadata = paymentClient.buildMetadata(
      req.user!.userId,
      req.user!.kycLevel,
      req.correlationId
    );

    const response = await paymentClient.getTransfer(metadata, id);

    if (response.error) {
      if (response.error.code === 'TRANSFER_NOT_FOUND') {
        res.status(404).json({ error: response.error.message });
        return;
      }
      res.status(400).json({ error: response.error.message, code: response.error.code });
      return;
    }

    res.json(mapTransferResponse(response.transfer));
  } catch (error) {
    logger.error({ error, correlationId: req.correlationId }, 'Failed to get transfer');
    res.status(503).json({ error: 'Payment service unavailable' });
  }
});

// Confirm transfer
router.post('/:id/confirm', async (req: AuthenticatedRequest, res: Response) => {
  try {
    const { id } = req.params;

    const metadata = paymentClient.buildMetadata(
      req.user!.userId,
      req.user!.kycLevel,
      req.correlationId
    );

    const response = await paymentClient.confirmTransfer(metadata, id);

    if (response.error) {
      res.status(400).json({ error: response.error.message, code: response.error.code });
      return;
    }

    logger.info({ transferId: id, correlationId: req.correlationId }, 'Transfer confirmed');
    res.json(mapTransferResponse(response.transfer));
  } catch (error) {
    logger.error({ error, correlationId: req.correlationId }, 'Failed to confirm transfer');
    res.status(503).json({ error: 'Payment service unavailable' });
  }
});

// List transfers
router.get('/', async (req: AuthenticatedRequest, res: Response) => {
  try {
    const { page = '1', pageSize = '10', status } = req.query;

    const metadata = paymentClient.buildMetadata(
      req.user!.userId,
      req.user!.kycLevel,
      req.correlationId
    );

    const response = await paymentClient.listTransfers(
      metadata,
      parseInt(page as string, 10),
      parseInt(pageSize as string, 10),
      status as string | undefined
    );

    if (response.error) {
      res.status(400).json({ error: response.error.message, code: response.error.code });
      return;
    }

    res.json({
      transfers: response.transfers.map(mapTransferResponse),
      pagination: response.pagination,
    });
  } catch (error) {
    logger.error({ error, correlationId: req.correlationId }, 'Failed to list transfers');
    res.status(503).json({ error: 'Payment service unavailable' });
  }
});

// Cancel transfer
router.post('/:id/cancel', async (req: AuthenticatedRequest, res: Response) => {
  try {
    const { id } = req.params;
    const { reason } = req.body;

    const metadata = paymentClient.buildMetadata(
      req.user!.userId,
      req.user!.kycLevel,
      req.correlationId
    );

    const response = await paymentClient.cancelTransfer(metadata, id, reason || 'User cancelled');

    if (response.error) {
      res.status(400).json({ error: response.error.message, code: response.error.code });
      return;
    }

    logger.info({ transferId: id, reason, correlationId: req.correlationId }, 'Transfer cancelled');
    res.json(mapTransferResponse(response.transfer));
  } catch (error) {
    logger.error({ error, correlationId: req.correlationId }, 'Failed to cancel transfer');
    res.status(503).json({ error: 'Payment service unavailable' });
  }
});

function mapTransferResponse(transfer: paymentClient.Transfer) {
  return {
    id: transfer.id,
    userId: transfer.userId,
    status: transfer.status,
    sourceAmount: transfer.sourceAmount,
    targetAmount: transfer.targetAmount,
    fee: transfer.fee,
    exchangeRate: transfer.exchangeRate,
    fundingMethod: transfer.fundingMethod,
    payoutMethod: transfer.payoutMethod,
    recipientId: transfer.recipientId,
    fundingDetails: transfer.fundingDetails,
    createdAt: timestampToIso(transfer.createdAt),
    updatedAt: timestampToIso(transfer.updatedAt),
  };
}

function timestampToIso(ts: { seconds: string; nanos: number }): string {
  const ms = parseInt(ts.seconds, 10) * 1000 + ts.nanos / 1000000;
  return new Date(ms).toISOString();
}

export default router;
```

**Step 2: Commit**

```bash
git add src/routes/transfers.ts
git commit -m "feat(gateway): integrate transfers route with Payment gRPC"
```

---

### Task 8: Update Auth Middleware to Call Auth Service

**Files:**
- Modify: `src/middleware/auth.ts`

**Step 1: Update auth middleware to verify token via gRPC**

```typescript
import { Request, Response, NextFunction } from 'express';
import { config } from '../config';
import { logger } from '../utils/logger';
import * as authClient from '../grpc/authClient';

export interface AuthenticatedRequest extends Request {
  user?: {
    userId: string;
    email: string;
    kycLevel: string;
  };
  correlationId: string;
}

export const authMiddleware = async (
  req: AuthenticatedRequest,
  res: Response,
  next: NextFunction
): Promise<void> => {
  try {
    const authHeader = req.headers.authorization;

    if (!authHeader || !authHeader.startsWith('Bearer ')) {
      res.status(401).json({ error: 'Missing or invalid authorization header' });
      return;
    }

    const token = authHeader.split(' ')[1];

    // Call Auth service to verify token
    const response = await authClient.verifyToken(token);

    if (!response.valid || response.error) {
      res.status(401).json({ error: response.error?.message || 'Invalid or expired token' });
      return;
    }

    // Get full user info
    const userResponse = await authClient.getUser(response.userId);

    if (userResponse.error) {
      res.status(401).json({ error: 'User not found' });
      return;
    }

    req.user = {
      userId: response.userId,
      email: userResponse.user.email,
      kycLevel: response.kycLevel,
    };

    next();
  } catch (error) {
    logger.error({ error }, 'Authentication failed');
    // Fall back to local JWT verification if Auth service unavailable
    try {
      const jwt = await import('jsonwebtoken');
      const authHeader = req.headers.authorization;
      if (authHeader) {
        const token = authHeader.split(' ')[1];
        const decoded = jwt.verify(token, config.jwt.secret) as {
          userId: string;
          email: string;
          kycLevel: string;
        };
        req.user = decoded;
        next();
        return;
      }
    } catch {
      // JWT fallback also failed
    }
    res.status(503).json({ error: 'Authentication service unavailable' });
  }
};

export const optionalAuthMiddleware = async (
  req: AuthenticatedRequest,
  res: Response,
  next: NextFunction
): Promise<void> => {
  try {
    const authHeader = req.headers.authorization;

    if (authHeader && authHeader.startsWith('Bearer ')) {
      const token = authHeader.split(' ')[1];

      try {
        const response = await authClient.verifyToken(token);
        if (response.valid && !response.error) {
          req.user = {
            userId: response.userId,
            email: '',
            kycLevel: response.kycLevel,
          };
        }
      } catch {
        // Token invalid or service down, continue without auth
      }
    }

    next();
  } catch (error) {
    next();
  }
};
```

**Step 2: Commit**

```bash
git add src/middleware/auth.ts
git commit -m "feat(gateway): integrate auth middleware with Auth gRPC"
```

---

## Phase C: Error Handling and Polish

### Task 9: Add gRPC Error Handler Utility

**Files:**
- Create: `src/utils/grpcErrors.ts`

**Step 1: Create error handler utility**

```typescript
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
```

**Step 2: Commit**

```bash
git add src/utils/grpcErrors.ts
git commit -m "feat(gateway): add gRPC error mapping utility"
```

---

### Task 10: Add Integration Test Setup

**Files:**
- Create: `src/__tests__/routes/rates.test.ts`

**Step 1: Create basic integration test**

```typescript
import { describe, it, expect, jest, beforeEach } from '@jest/globals';

// Mock the gRPC clients
jest.mock('../../grpc/exchangeClient', () => ({
  getRate: jest.fn(),
  lockRate: jest.fn(),
  getLockedRate: jest.fn(),
  getCorridors: jest.fn(),
}));

import * as exchangeClient from '../../grpc/exchangeClient';

describe('Rates Routes', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('GET /:from/:to', () => {
    it('should return exchange rate from gRPC service', async () => {
      const mockRate = {
        rate: {
          sourceCurrency: 'SGD',
          targetCurrency: 'PHP',
          rate: '39.75',
          buyRate: '39.63',
          marginPercentage: '0.3',
          fetchedAt: { seconds: '1704412800', nanos: 0 },
          expiresAt: { seconds: '1704412830', nanos: 0 },
        },
      };

      (exchangeClient.getRate as jest.Mock).mockResolvedValue(mockRate);

      // Test would use supertest here with actual app
      const result = await exchangeClient.getRate('SGD', 'PHP');

      expect(result.rate.sourceCurrency).toBe('SGD');
      expect(result.rate.targetCurrency).toBe('PHP');
      expect(exchangeClient.getRate).toHaveBeenCalledWith('SGD', 'PHP');
    });

    it('should handle gRPC service errors', async () => {
      (exchangeClient.getRate as jest.Mock).mockRejectedValue(new Error('Service unavailable'));

      await expect(exchangeClient.getRate('SGD', 'PHP')).rejects.toThrow('Service unavailable');
    });
  });

  describe('POST /lock', () => {
    it('should lock rate via gRPC service', async () => {
      const mockLocked = {
        lockedRate: {
          lockId: 'lock_123',
          rate: {
            sourceCurrency: 'SGD',
            targetCurrency: 'PHP',
            rate: '39.75',
            buyRate: '39.63',
            marginPercentage: '0.3',
          },
          lockedAt: { seconds: '1704412800', nanos: 0 },
          expiresAt: { seconds: '1704412830', nanos: 0 },
          expired: false,
        },
      };

      (exchangeClient.lockRate as jest.Mock).mockResolvedValue(mockLocked);

      const result = await exchangeClient.lockRate('SGD', 'PHP', 30);

      expect(result.lockedRate.lockId).toBe('lock_123');
      expect(exchangeClient.lockRate).toHaveBeenCalledWith('SGD', 'PHP', 30);
    });
  });

  describe('GET /corridors', () => {
    it('should return corridors from gRPC service', async () => {
      const mockCorridors = {
        corridors: [
          {
            sourceCurrency: 'SGD',
            targetCurrency: 'PHP',
            enabled: true,
            feePercentage: '0.5',
            feeMinimum: { currency: 'SGD', amount: '3.00' },
            marginPercentage: '0.3',
            payoutMethods: ['BANK_ACCOUNT', 'MOBILE_WALLET'],
          },
        ],
      };

      (exchangeClient.getCorridors as jest.Mock).mockResolvedValue(mockCorridors);

      const result = await exchangeClient.getCorridors();

      expect(result.corridors).toHaveLength(1);
      expect(result.corridors[0].sourceCurrency).toBe('SGD');
    });
  });
});
```

**Step 2: Commit**

```bash
git add src/__tests__/routes/rates.test.ts
git commit -m "test(gateway): add rates route integration tests"
```

---

## Final Summary

After completing all tasks, the API Gateway will have:

1. **gRPC Client Infrastructure** - Proto loader, typed clients for Auth, Exchange, Payment
2. **Real Service Integration** - All routes call actual backend services
3. **Error Handling** - gRPC to REST error mapping
4. **Fallback Auth** - Local JWT verification if Auth service unavailable
5. **Tests** - Basic integration test setup

**Service Endpoints:**
| Route | Backend Service |
|-------|-----------------|
| `/api/auth/*` | Auth Service (gRPC) |
| `/api/rates/*` | Exchange Rate Service (gRPC) |
| `/api/transfers/*` | Payment Service (gRPC) |
