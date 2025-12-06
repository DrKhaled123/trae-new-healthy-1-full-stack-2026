import { Router, Request, Response } from 'express';
import { catchAsync } from '../middleware/errorHandler';
import { logger } from '../config/logger';

const router = Router();

/**
 * @swagger
 * /api/health:
 *   get:
 *     summary: Health check endpoint
 *     description: Returns the health status of the API server and its dependencies
 *     tags: [Health]
 *     responses:
 *       200:
 *         description: Server is healthy
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                   example: true
 *                 message:
 *                   type: string
 *                   example: Server is healthy
 *                 timestamp:
 *                   type: string
 *                   example: "2023-12-07T10:30:00.000Z"
 *                 uptime:
 *                   type: number
 *                   example: 3600
 *                 version:
 *                   type: string
 *                   example: "1.0.0"
 *                 environment:
 *                   type: string
 *                   example: "production"
 *                 checks:
 *                   type: object
 *                   properties:
 *                     database:
 *                       type: object
 *                       properties:
 *                         status:
 *                           type: string
 *                           example: "connected"
 *                         responseTime:
 *                           type: number
 *                           example: 15
 *                     redis:
 *                       type: object
 *                       properties:
 *                         status:
 *                           type: string
 *                           example: "connected"
 *                         responseTime:
 *                           type: number
 *                           example: 5
 *                     memory:
 *                       type: object
 *                       properties:
 *                         used:
 *                           type: string
 *                           example: "125MB"
 *                         total:
 *                           type: string
 *                           example: "512MB"
 *                         percentage:
 *                           type: number
 *                           example: 24.4
 *       503:
 *         description: Service unavailable
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                   example: false
 *                 message:
 *                   type: string
 *                   example: Service unavailable
 *                 timestamp:
 *                   type: string
 *                   example: "2023-12-07T10:30:00.000Z"
 *                 checks:
 *                   type: object
 *                   properties:
 *                     database:
 *                       type: object
 *                       properties:
 *                         status:
 *                           type: string
 *                           example: "disconnected"
 *                         error:
 *                           type: string
 *                           example: "Connection timeout"
 */

/**
 * Basic health check endpoint
 */
router.get('/', catchAsync(async (req: Request, res: Response) => {
  const startTime = Date.now();
  
  // Basic server info
  const serverInfo = {
    success: true,
    message: 'Server is healthy',
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
    version: process.env.npm_package_version || '1.0.0',
    environment: process.env.NODE_ENV || 'development',
    responseTime: Date.now() - startTime
  };

  logger.info('Health check accessed', {
    ip: req.ip,
    userAgent: req.get('User-Agent'),
    responseTime: serverInfo.responseTime
  });

  res.status(200).json(serverInfo);
}));

/**
 * Detailed health check with dependency status
 */
router.get('/detailed', catchAsync(async (req: Request, res: Response) => {
  const startTime = Date.now();
  
  // Memory usage
  const memoryUsage = process.memoryUsage();
  const memory = {
    used: `${Math.round(memoryUsage.heapUsed / 1024 / 1024)}MB`,
    total: `${Math.round(memoryUsage.heapTotal / 1024 / 1024)}MB`,
    external: `${Math.round(memoryUsage.external / 1024 / 1024)}MB`,
    percentage: Math.round((memoryUsage.heapUsed / memoryUsage.heapTotal) * 100)
  };

  // CPU usage (simplified)
  const cpuUsage = process.cpuUsage();

  // Check dependencies
  const checks = {
    server: {
      status: 'healthy',
      responseTime: Date.now() - startTime
    },
    memory,
    cpu: {
      user: cpuUsage.user,
      system: cpuUsage.system
    }
  };

  // Check database connectivity (placeholder - would need actual DB connection)
  try {
    // This would be replaced with actual database ping
    checks.database = {
      status: 'connected',
      responseTime: 10 // placeholder
    };
  } catch (error) {
    checks.database = {
      status: 'disconnected',
      error: error instanceof Error ? error.message : 'Unknown error'
    };
  }

  // Check Redis connectivity (placeholder - would need actual Redis connection)
  try {
    // This would be replaced with actual Redis ping
    checks.redis = {
      status: 'connected',
      responseTime: 5 // placeholder
    };
  } catch (error) {
    checks.redis = {
      status: 'disconnected',
      error: error instanceof Error ? error.message : 'Unknown error'
    };
  }

  // Determine overall health status
  const allHealthy = Object.values(checks).every(check => 
    typeof check === 'object' && check.status === 'healthy' || check.status === 'connected'
  );

  const response = {
    success: allHealthy,
    message: allHealthy ? 'All systems operational' : 'Some systems are down',
    timestamp: new Date().toISOString(),
    uptime: process.uptime(),
    version: process.env.npm_package_version || '1.0.0',
    environment: process.env.NODE_ENV || 'development',
    responseTime: Date.now() - startTime,
    checks
  };

  const statusCode = allHealthy ? 200 : 503;

  logger.info('Detailed health check accessed', {
    ip: req.ip,
    userAgent: req.get('User-Agent'),
    healthy: allHealthy,
    responseTime: response.responseTime
  });

  res.status(statusCode).json(response);
}));

/**
 * Readiness probe endpoint
 * Used by Kubernetes/OpenShift to determine if the pod is ready to serve traffic
 */
router.get('/ready', catchAsync(async (req: Request, res: Response) => {
  // Check if all critical dependencies are ready
  const isReady = true; // Would check actual dependencies

  if (isReady) {
    res.status(200).json({
      success: true,
      message: 'Server is ready',
      timestamp: new Date().toISOString()
    });
  } else {
    res.status(503).json({
      success: false,
      message: 'Server is not ready',
      timestamp: new Date().toISOString()
    });
  }
}));

/**
 * Liveness probe endpoint
 * Used by Kubernetes/OpenShift to determine if the pod is still alive
 */
router.get('/live', catchAsync(async (req: Request, res: Response) => {
  // Simple liveness check - if we can respond, we're alive
  res.status(200).json({
    success: true,
    message: 'Server is alive',
    timestamp: new Date().toISOString(),
    uptime: process.uptime()
  });
}));

/**
 * Version information endpoint
 */
router.get('/version', catchAsync(async (req: Request, res: Response) => {
  res.status(200).json({
    success: true,
    version: process.env.npm_package_version || '1.0.0',
    build: process.env.BUILD_NUMBER || 'unknown',
    environment: process.env.NODE_ENV || 'development',
    timestamp: new Date().toISOString()
  });
}));

export default router;
