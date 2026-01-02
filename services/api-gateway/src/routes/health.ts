import { Router, Request, Response } from 'express';

const router = Router();

router.get('/health', (_req: Request, res: Response) => {
  res.json({
    status: 'healthy',
    service: 'api-gateway',
    timestamp: new Date().toISOString(),
  });
});

router.get('/ready', (_req: Request, res: Response) => {
  // In production, check connections to downstream services
  res.json({
    status: 'ready',
    service: 'api-gateway',
    timestamp: new Date().toISOString(),
  });
});

export default router;
