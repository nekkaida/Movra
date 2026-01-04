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
