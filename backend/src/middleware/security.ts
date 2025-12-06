import rateLimit from 'express-rate-limit';
import { Request, Response } from 'express';
import { logger } from '../config/logger';

/**
 * General API Rate Limiter
 * Limits requests to prevent abuse and DDoS attacks
 */
export const apiLimiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 100, // Limit each IP to 100 requests per windowMs
  message: {
    success: false,
    message: 'Too many requests from this IP, please try again later',
    code: 'RATE_LIMIT_EXCEEDED'
  },
  standardHeaders: true, // Return rate limit info in the `RateLimit-*` headers
  legacyHeaders: false, // Disable the `X-RateLimit-*` headers
  handler: (req: Request, res: Response) => {
    logger.warn('Rate limit exceeded', {
      ip: req.ip,
      url: req.url,
      method: req.method,
      userAgent: req.get('User-Agent')
    });
    
    res.status(429).json({
      success: false,
      message: 'Too many requests from this IP, please try again later',
      code: 'RATE_LIMIT_EXCEEDED',
      retryAfter: res.get('Retry-After')
    });
  }
});

/**
 * Strict Rate Limiter for sensitive endpoints
 * Used for authentication, password reset, etc.
 */
export const strictLimiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 5, // Limit each IP to 5 requests per windowMs
  message: {
    success: false,
    message: 'Too many attempts from this IP, please try again later',
    code: 'STRICT_RATE_LIMIT_EXCEEDED'
  },
  standardHeaders: true,
  legacyHeaders: false,
  skipSuccessfulRequests: true,
  handler: (req: Request, res: Response) => {
    logger.warn('Strict rate limit exceeded', {
      ip: req.ip,
      url: req.url,
      method: req.method,
      userAgent: req.get('User-Agent')
    });
    
    res.status(429).json({
      success: false,
      message: 'Too many attempts from this IP, please try again later',
      code: 'STRICT_RATE_LIMIT_EXCEEDED',
      retryAfter: res.get('Retry-After')
    });
  }
});

/**
 * Password Reset Rate Limiter
 * Specifically for password reset requests
 */
export const passwordResetLimiter = rateLimit({
  windowMs: 60 * 60 * 1000, // 1 hour
  max: 3, // Limit each IP to 3 password reset requests per hour
  message: {
    success: false,
    message: 'Too many password reset attempts, please try again later',
    code: 'PASSWORD_RESET_LIMIT_EXCEEDED'
  },
  standardHeaders: true,
  legacyHeaders: false,
  keyGenerator: (req: Request) => {
    // Use email for limiting if available, otherwise use IP
    return req.body?.email || req.ip;
  },
  handler: (req: Request, res: Response) => {
    logger.warn('Password reset rate limit exceeded', {
      ip: req.ip,
      email: req.body?.email,
      userAgent: req.get('User-Agent')
    });
    
    res.status(429).json({
      success: false,
      message: 'Too many password reset attempts, please try again later',
      code: 'PASSWORD_RESET_LIMIT_EXCEEDED',
      retryAfter: res.get('Retry-After')
    });
  }
});

/**
 * File Upload Rate Limiter
 * Limits file upload requests to prevent abuse
 */
export const uploadLimiter = rateLimit({
  windowMs: 60 * 60 * 1000, // 1 hour
  max: 20, // Limit each IP to 20 uploads per hour
  message: {
    success: false,
    message: 'Too many file uploads from this IP, please try again later',
    code: 'UPLOAD_LIMIT_EXCEEDED'
  },
  standardHeaders: true,
  legacyHeaders: false,
  handler: (req: Request, res: Response) => {
    logger.warn('Upload rate limit exceeded', {
      ip: req.ip,
      url: req.url,
      method: req.method,
      userAgent: req.get('User-Agent')
    });
    
    res.status(429).json({
      success: false,
      message: 'Too many file uploads from this IP, please try again later',
      code: 'UPLOAD_LIMIT_EXCEEDED',
      retryAfter: res.get('Retry-After')
    });
  }
});

/**
 * Create User Rate Limiter
 * Limits account creation to prevent spam
 */
export const createUserLimiter = rateLimit({
  windowMs: 60 * 60 * 1000, // 1 hour
  max: 5, // Limit each IP to 5 account creations per hour
  message: {
    success: false,
    message: 'Too many account creation attempts, please try again later',
    code: 'ACCOUNT_CREATION_LIMIT_EXCEEDED'
  },
  standardHeaders: true,
  legacyHeaders: false,
  handler: (req: Request, res: Response) => {
    logger.warn('Account creation rate limit exceeded', {
      ip: req.ip,
      url: req.url,
      method: req.method,
      userAgent: req.get('User-Agent')
    });
    
    res.status(429).json({
      success: false,
      message: 'Too many account creation attempts, please try again later',
      code: 'ACCOUNT_CREATION_LIMIT_EXCEEDED',
      retryAfter: res.get('Retry-After')
    });
  }
});

/**
 * API Key Rate Limiter
 * For endpoints that use API key authentication
 */
export const apiKeyLimiter = rateLimit({
  windowMs: 15 * 60 * 1000, // 15 minutes
  max: 1000, // Higher limit for authenticated API calls
  keyGenerator: (req: Request) => {
    // Use API key if available, otherwise use IP
    return req.headers['x-api-key'] as string || req.ip;
  },
  message: {
    success: false,
    message: 'API rate limit exceeded, please try again later',
    code: 'API_RATE_LIMIT_EXCEEDED'
  },
  standardHeaders: true,
  legacyHeaders: false,
  handler: (req: Request, res: Response) => {
    logger.warn('API rate limit exceeded', {
      ip: req.ip,
      apiKey: req.headers['x-api-key'],
      url: req.url,
      method: req.method
    });
    
    res.status(429).json({
      success: false,
      message: 'API rate limit exceeded, please try again later',
      code: 'API_RATE_LIMIT_EXCEEDED',
      retryAfter: res.get('Retry-After')
    });
  }
});

/**
 * Dynamic Rate Limiter
 * Creates a rate limiter with custom configuration
 */
export const createDynamicLimiter = (options: {
  windowMs?: number;
  max?: number;
  message?: string;
  skipSuccessfulRequests?: boolean;
  skipFailedRequests?: boolean;
  keyGenerator?: (req: Request) => string;
}) => {
  return rateLimit({
    windowMs: options.windowMs || 15 * 60 * 1000,
    max: options.max || 100,
    message: {
      success: false,
      message: options.message || 'Rate limit exceeded',
      code: 'RATE_LIMIT_EXCEEDED'
    },
    standardHeaders: true,
    legacyHeaders: false,
    skipSuccessfulRequests: options.skipSuccessfulRequests || false,
    skipFailedRequests: options.skipFailedRequests || false,
    keyGenerator: options.keyGenerator || ((req: Request) => req.ip),
    handler: (req: Request, res: Response) => {
      logger.warn('Dynamic rate limit exceeded', {
        ip: req.ip,
        url: req.url,
        method: req.method,
        userAgent: req.get('User-Agent')
      });
      
      res.status(429).json({
        success: false,
        message: options.message || 'Rate limit exceeded',
        code: 'RATE_LIMIT_EXCEEDED',
        retryAfter: res.get('Retry-After')
      });
    }
  });
};

export default {
  apiLimiter,
  strictLimiter,
  passwordResetLimiter,
  uploadLimiter,
  createUserLimiter,
  apiKeyLimiter,
  createDynamicLimiter
};
