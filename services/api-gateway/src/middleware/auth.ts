import { Request, Response, NextFunction } from 'express';
import jwt from 'jsonwebtoken';
import { config } from '../config';
import { logger } from '../utils/logger';

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

    // In production, this would call the Auth service via gRPC to verify the token
    // For now, we verify locally
    const decoded = jwt.verify(token, config.jwt.secret) as {
      userId: string;
      email: string;
      kycLevel: string;
    };

    req.user = decoded;
    next();
  } catch (error) {
    logger.error({ error }, 'Authentication failed');
    res.status(401).json({ error: 'Invalid or expired token' });
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
      const decoded = jwt.verify(token, config.jwt.secret) as {
        userId: string;
        email: string;
        kycLevel: string;
      };
      req.user = decoded;
    }

    next();
  } catch (error) {
    // Token invalid, but that's okay for optional auth
    next();
  }
};
