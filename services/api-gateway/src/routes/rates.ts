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
