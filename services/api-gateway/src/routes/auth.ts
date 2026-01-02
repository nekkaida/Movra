import { Router, Request, Response } from 'express';
import jwt from 'jsonwebtoken';
import { config } from '../config';
import { logger } from '../utils/logger';
import { AuthenticatedRequest, authMiddleware } from '../middleware/auth';

const router = Router();

// Register new user
router.post('/register', async (req: Request, res: Response) => {
  try {
    const { email, password, phone, firstName, lastName } = req.body;

    // Validate required fields
    if (!email || !password || !firstName || !lastName) {
      res.status(400).json({ error: 'Missing required fields' });
      return;
    }

    // In production, call Auth service via gRPC
    // For now, create a mock user
    const userId = `user_${Date.now()}`;

    const accessToken = jwt.sign(
      { userId, email, kycLevel: 'NONE' },
      config.jwt.secret,
      { expiresIn: '1h' }
    );

    const refreshToken = jwt.sign(
      { userId, type: 'refresh' },
      config.jwt.secret,
      { expiresIn: '7d' }
    );

    const user = {
      id: userId,
      email,
      phone,
      firstName,
      lastName,
      kycLevel: 'NONE',
      createdAt: new Date().toISOString(),
    };

    logger.info({ userId }, 'User registered');

    res.status(201).json({
      user,
      accessToken,
      refreshToken,
    });
  } catch (error) {
    logger.error({ error }, 'Registration failed');
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Login
router.post('/login', async (req: Request, res: Response) => {
  try {
    const { email, password } = req.body;

    if (!email || !password) {
      res.status(400).json({ error: 'Email and password required' });
      return;
    }

    // In production, call Auth service via gRPC
    // For now, create mock tokens
    const userId = `user_${Date.now()}`;

    const accessToken = jwt.sign(
      { userId, email, kycLevel: 'BASIC' },
      config.jwt.secret,
      { expiresIn: '1h' }
    );

    const refreshToken = jwt.sign(
      { userId, type: 'refresh' },
      config.jwt.secret,
      { expiresIn: '7d' }
    );

    const user = {
      id: userId,
      email,
      kycLevel: 'BASIC',
    };

    logger.info({ userId }, 'User logged in');

    res.json({
      user,
      accessToken,
      refreshToken,
      requiresMfa: false,
    });
  } catch (error) {
    logger.error({ error }, 'Login failed');
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Refresh token
router.post('/refresh', async (req: Request, res: Response) => {
  try {
    const { refreshToken } = req.body;

    if (!refreshToken) {
      res.status(400).json({ error: 'Refresh token required' });
      return;
    }

    // In production, call Auth service via gRPC
    const decoded = jwt.verify(refreshToken, config.jwt.secret) as {
      userId: string;
      type: string;
    };

    if (decoded.type !== 'refresh') {
      res.status(401).json({ error: 'Invalid refresh token' });
      return;
    }

    const newAccessToken = jwt.sign(
      { userId: decoded.userId, email: 'user@example.com', kycLevel: 'BASIC' },
      config.jwt.secret,
      { expiresIn: '1h' }
    );

    const newRefreshToken = jwt.sign(
      { userId: decoded.userId, type: 'refresh' },
      config.jwt.secret,
      { expiresIn: '7d' }
    );

    res.json({
      accessToken: newAccessToken,
      refreshToken: newRefreshToken,
      expiresAt: new Date(Date.now() + 3600000).toISOString(),
    });
  } catch (error) {
    logger.error({ error }, 'Token refresh failed');
    res.status(401).json({ error: 'Invalid or expired refresh token' });
  }
});

// Get current user
router.get('/me', authMiddleware, async (req: AuthenticatedRequest, res: Response) => {
  try {
    // In production, call Auth service via gRPC
    const user = {
      id: req.user?.userId,
      email: req.user?.email,
      kycLevel: req.user?.kycLevel,
    };

    res.json(user);
  } catch (error) {
    logger.error({ error }, 'Failed to get user');
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Logout
router.post('/logout', authMiddleware, async (req: AuthenticatedRequest, res: Response) => {
  try {
    // In production, invalidate the refresh token via Auth service
    logger.info({ userId: req.user?.userId }, 'User logged out');
    res.json({ success: true });
  } catch (error) {
    logger.error({ error }, 'Logout failed');
    res.status(500).json({ error: 'Internal server error' });
  }
});

export default router;
