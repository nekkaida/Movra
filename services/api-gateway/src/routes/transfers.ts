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
      req.correlationId || ''
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
      req.correlationId || ''
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
      req.correlationId || ''
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
      req.correlationId || ''
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
      req.correlationId || ''
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
