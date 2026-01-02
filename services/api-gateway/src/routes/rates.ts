import { Router, Request, Response } from 'express';
import { AuthenticatedRequest, optionalAuthMiddleware } from '../middleware/auth';
import { logger } from '../utils/logger';

const router = Router();

// Get exchange rate (public, but rate limiting applies)
router.get('/:from/:to', optionalAuthMiddleware, async (req: Request, res: Response) => {
  try {
    const { from, to } = req.params;

    // Validate currency codes (ISO 4217)
    if (!/^[A-Z]{3}$/.test(from) || !/^[A-Z]{3}$/.test(to)) {
      res.status(400).json({ error: 'Invalid currency code format' });
      return;
    }

    // In production, call Exchange Rate service via gRPC
    // Mock response with realistic spread
    const midMarketRate = getMockMidMarketRate(from, to);
    const margin = 0.003; // 0.3% margin
    const buyRate = midMarketRate * (1 - margin);

    const rate = {
      sourceCurrency: from,
      targetCurrency: to,
      rate: midMarketRate.toFixed(6),
      buyRate: buyRate.toFixed(6),
      marginPercentage: (margin * 100).toFixed(2),
      fetchedAt: new Date().toISOString(),
      expiresAt: new Date(Date.now() + 30000).toISOString(), // 30 seconds
    };

    res.json(rate);
  } catch (error) {
    logger.error({ error }, 'Failed to get exchange rate');
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Lock rate for transfer
router.post('/lock', async (req: AuthenticatedRequest, res: Response) => {
  try {
    const { sourceCurrency, targetCurrency, durationSeconds = 30 } = req.body;

    if (!sourceCurrency || !targetCurrency) {
      res.status(400).json({ error: 'Source and target currency required' });
      return;
    }

    // In production, call Exchange Rate service via gRPC
    const midMarketRate = getMockMidMarketRate(sourceCurrency, targetCurrency);
    const margin = 0.003;
    const buyRate = midMarketRate * (1 - margin);

    const lockedRate = {
      lockId: `lock_${Date.now()}`,
      rate: {
        sourceCurrency,
        targetCurrency,
        rate: midMarketRate.toFixed(6),
        buyRate: buyRate.toFixed(6),
        marginPercentage: (margin * 100).toFixed(2),
      },
      lockedAt: new Date().toISOString(),
      expiresAt: new Date(Date.now() + durationSeconds * 1000).toISOString(),
      expired: false,
    };

    logger.info({ lockId: lockedRate.lockId }, 'Rate locked');

    res.json(lockedRate);
  } catch (error) {
    logger.error({ error }, 'Failed to lock rate');
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Get locked rate
router.get('/locked/:lockId', async (req: Request, res: Response) => {
  try {
    const { lockId } = req.params;

    // In production, call Exchange Rate service via gRPC
    // Mock: check if expired based on lockId timestamp
    const lockTimestamp = parseInt(lockId.replace('lock_', ''), 10);
    const expired = Date.now() - lockTimestamp > 30000;

    if (expired) {
      res.status(410).json({ error: 'Rate lock expired', expired: true });
      return;
    }

    const lockedRate = {
      lockId,
      rate: {
        sourceCurrency: 'SGD',
        targetCurrency: 'PHP',
        rate: '39.750000',
        buyRate: '39.630750',
      },
      lockedAt: new Date(lockTimestamp).toISOString(),
      expiresAt: new Date(lockTimestamp + 30000).toISOString(),
      expired: false,
    };

    res.json(lockedRate);
  } catch (error) {
    logger.error({ error }, 'Failed to get locked rate');
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Get available corridors
router.get('/corridors', async (_req: Request, res: Response) => {
  try {
    // In production, call Exchange Rate service via gRPC
    const corridors = [
      {
        sourceCurrency: 'SGD',
        targetCurrency: 'PHP',
        enabled: true,
        feePercentage: '0.5',
        feeMinimum: { currency: 'SGD', amount: '3.00' },
        marginPercentage: '0.3',
        payoutMethods: ['BANK_ACCOUNT', 'MOBILE_WALLET', 'CASH_PICKUP'],
      },
      {
        sourceCurrency: 'SGD',
        targetCurrency: 'INR',
        enabled: true,
        feePercentage: '0.5',
        feeMinimum: { currency: 'SGD', amount: '3.00' },
        marginPercentage: '0.35',
        payoutMethods: ['BANK_ACCOUNT', 'MOBILE_WALLET'],
      },
      {
        sourceCurrency: 'SGD',
        targetCurrency: 'IDR',
        enabled: true,
        feePercentage: '0.5',
        feeMinimum: { currency: 'SGD', amount: '3.00' },
        marginPercentage: '0.3',
        payoutMethods: ['BANK_ACCOUNT', 'MOBILE_WALLET'],
      },
      {
        sourceCurrency: 'USD',
        targetCurrency: 'PHP',
        enabled: true,
        feePercentage: '0.4',
        feeMinimum: { currency: 'USD', amount: '2.00' },
        marginPercentage: '0.25',
        payoutMethods: ['BANK_ACCOUNT', 'MOBILE_WALLET', 'CASH_PICKUP'],
      },
      {
        sourceCurrency: 'SGD',
        targetCurrency: 'USD',
        enabled: true,
        feePercentage: '0.3',
        feeMinimum: { currency: 'SGD', amount: '2.00' },
        marginPercentage: '0.2',
        payoutMethods: ['BANK_ACCOUNT'],
      },
    ];

    res.json({ corridors });
  } catch (error) {
    logger.error({ error }, 'Failed to get corridors');
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Helper function for mock rates
function getMockMidMarketRate(from: string, to: string): number {
  const rates: Record<string, Record<string, number>> = {
    SGD: { PHP: 39.75, INR: 62.5, IDR: 11800, USD: 0.74 },
    USD: { PHP: 53.75, SGD: 1.35 },
    PHP: { SGD: 0.0252 },
  };

  return rates[from]?.[to] || 1;
}

export default router;
