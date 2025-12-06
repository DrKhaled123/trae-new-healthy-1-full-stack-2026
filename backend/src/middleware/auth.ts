import { Request, Response, NextFunction } from 'express';
import jwt from 'jsonwebtoken';
import { config } from '../config/env';
import { logger } from '../config/logger';

// Extend Request interface to include user
declare global {
  namespace Express {
    interface Request {
      user?: {
        id: string;
        email: string;
        role: string;
      };
    }
  }
}

export interface JWTPayload {
  id: string;
  email: string;
  role: string;
  iat?: number;
  exp?: number;
}

/**
 * JWT Authentication Middleware
 * Validates JWT tokens and attaches user data to request object
 */
export const authenticateToken = (req: Request, res: Response, next: NextFunction): void => {
  try {
    const authHeader = req.headers.authorization;
    const token = authHeader && authHeader.split(' ')[1]; // Bearer TOKEN

    if (!token) {
      res.status(401).json({
        success: false,
        message: 'Access token required',
        code: 'TOKEN_REQUIRED'
      });
      return;
    }

    jwt.verify(token, config.JWT_SECRET, (err, decoded) => {
      if (err) {
        logger.warn(`Invalid token attempt: ${err.message}`, {
          ip: req.ip,
          userAgent: req.get('User-Agent')
        });
        
        res.status(403).json({
          success: false,
          message: 'Invalid or expired token',
          code: 'TOKEN_INVALID'
        });
        return;
      }

      const payload = decoded as JWTPayload;
      req.user = {
        id: payload.id,
        email: payload.email,
        role: payload.role
      };

      logger.debug('User authenticated successfully', {
        userId: payload.id,
        email: payload.email,
        role: payload.role
      });

      next();
    });
  } catch (error) {
    logger.error('Authentication middleware error:', error);
    res.status(500).json({
      success: false,
      message: 'Authentication error',
      code: 'AUTH_ERROR'
    });
  }
};

/**
 * Role-based Authorization Middleware
 * Checks if user has required role
 */
export const authorize = (roles: string[]) => {
  return (req: Request, res: Response, next: NextFunction): void => {
    if (!req.user) {
      res.status(401).json({
        success: false,
        message: 'Authentication required',
        code: 'AUTH_REQUIRED'
      });
      return;
    }

    if (!roles.includes(req.user.role)) {
      logger.warn('Unauthorized access attempt', {
        userId: req.user.id,
        userRole: req.user.role,
        requiredRoles: roles,
        ip: req.ip,
        path: req.path
      });

      res.status(403).json({
        success: false,
        message: 'Insufficient permissions',
        code: 'INSUFFICIENT_PERMISSIONS'
      });
      return;
    }

    next();
  };
};

/**
 * Optional Authentication Middleware
 * Attaches user data if token is present but doesn't require it
 */
export const optionalAuth = (req: Request, res: Response, next: NextFunction): void => {
  try {
    const authHeader = req.headers.authorization;
    const token = authHeader && authHeader.split(' ')[1];

    if (token) {
      jwt.verify(token, config.JWT_SECRET, (err, decoded) => {
        if (!err) {
          const payload = decoded as JWTPayload;
          req.user = {
            id: payload.id,
            email: payload.email,
            role: payload.role
          };
        }
      });
    }

    next();
  } catch (error) {
    // Continue without authentication for optional auth
    next();
  }
};
