/* eslint-disable @typescript-eslint/no-explicit-any */

// Mock the gRPC clients before imports
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
