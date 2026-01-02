import { Router, Response } from 'express';
import { AuthenticatedRequest, authMiddleware } from '../middleware/auth';
import { logger } from '../utils/logger';

const router = Router();

// All transfer routes require authentication
router.use(authMiddleware);

// Create a new transfer
router.post('/', async (req: AuthenticatedRequest, res: Response) => {
  try {
    const { sourceAmount, sourceCurrency, targetCurrency, recipientId, fundingMethod, rateLockId } =
      req.body;

    // Validate required fields
    if (!sourceAmount || !sourceCurrency || !targetCurrency || !recipientId || !fundingMethod) {
      res.status(400).json({ error: 'Missing required fields' });
      return;
    }

    // In production, this would call Payment service via gRPC
    // For now, return a mock response
    const transfer = {
      id: `txn_${Date.now()}`,
      userId: req.user?.userId,
      status: 'CREATED',
      sourceAmount: { currency: sourceCurrency, amount: sourceAmount },
      targetAmount: { currency: targetCurrency, amount: '0' }, // Calculated by service
      fundingMethod,
      recipientId,
      rateLockId,
      createdAt: new Date().toISOString(),
    };

    logger.info({ transferId: transfer.id, correlationId: req.correlationId }, 'Transfer created');

    res.status(201).json(transfer);
  } catch (error) {
    logger.error({ error, correlationId: req.correlationId }, 'Failed to create transfer');
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Get transfer by ID
router.get('/:id', async (req: AuthenticatedRequest, res: Response) => {
  try {
    const { id } = req.params;

    // In production, call Payment service via gRPC
    const transfer = {
      id,
      userId: req.user?.userId,
      status: 'CREATED',
      sourceAmount: { currency: 'SGD', amount: '500.00' },
      targetAmount: { currency: 'PHP', amount: '19850.00' },
      exchangeRate: '39.70',
      fee: { currency: 'SGD', amount: '2.50' },
      createdAt: new Date().toISOString(),
    };

    res.json(transfer);
  } catch (error) {
    logger.error({ error, correlationId: req.correlationId }, 'Failed to get transfer');
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Confirm transfer
router.post('/:id/confirm', async (req: AuthenticatedRequest, res: Response) => {
  try {
    const { id } = req.params;

    // In production, call Payment service via gRPC
    const transfer = {
      id,
      status: 'AWAITING_FUNDS',
      fundingDetails: {
        bankName: 'DBS Bank',
        accountNumber: '1234567890',
        accountName: 'Movra Pte Ltd',
        reference: `MOV${id}`,
      },
    };

    logger.info({ transferId: id, correlationId: req.correlationId }, 'Transfer confirmed');

    res.json(transfer);
  } catch (error) {
    logger.error({ error, correlationId: req.correlationId }, 'Failed to confirm transfer');
    res.status(500).json({ error: 'Internal server error' });
  }
});

// List transfers for user
router.get('/', async (req: AuthenticatedRequest, res: Response) => {
  try {
    const { page = '1', pageSize = '10', status } = req.query;

    // In production, call Payment service via gRPC
    const transfers = [
      {
        id: 'txn_1',
        status: 'COMPLETED',
        sourceAmount: { currency: 'SGD', amount: '500.00' },
        targetAmount: { currency: 'PHP', amount: '19850.00' },
        createdAt: new Date().toISOString(),
      },
    ];

    res.json({
      transfers,
      pagination: {
        page: parseInt(page as string, 10),
        pageSize: parseInt(pageSize as string, 10),
        totalPages: 1,
        totalItems: 1,
      },
    });
  } catch (error) {
    logger.error({ error, correlationId: req.correlationId }, 'Failed to list transfers');
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Cancel transfer
router.post('/:id/cancel', async (req: AuthenticatedRequest, res: Response) => {
  try {
    const { id } = req.params;
    const { reason } = req.body;

    // In production, call Payment service via gRPC
    logger.info({ transferId: id, reason, correlationId: req.correlationId }, 'Transfer cancelled');

    res.json({ id, status: 'CANCELLED' });
  } catch (error) {
    logger.error({ error, correlationId: req.correlationId }, 'Failed to cancel transfer');
    res.status(500).json({ error: 'Internal server error' });
  }
});

export default router;
