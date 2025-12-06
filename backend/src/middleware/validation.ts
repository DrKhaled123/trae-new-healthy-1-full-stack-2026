import { Request, Response, NextFunction } from 'express';
import { validationResult, ValidationError } from 'express-validator';
import { AppError } from './errorHandler';
import Joi from 'joi';
import { logger } from '../config/logger';

/**
 * Express Validator Middleware
 * Checks for validation errors and formats them
 */
export const validateRequest = (req: Request, res: Response, next: NextFunction): void => {
  const errors = validationResult(req);
  
  if (!errors.isEmpty()) {
    const formattedErrors = errors.array().map((error: ValidationError) => ({
      field: error.type === 'field' ? error.path : 'unknown',
      message: error.msg,
      value: error.type === 'field' ? error.value : undefined
    }));

    logger.warn('Validation failed', {
      url: req.url,
      method: req.method,
      errors: formattedErrors,
      body: req.body,
      params: req.params,
      query: req.query
    });

    const error = new AppError('Validation failed', 400, 'VALIDATION_ERROR');
    error.message = JSON.stringify(formattedErrors);
    return next(error);
  }

  next();
};

/**
 * Joi Validation Middleware Factory
 * Creates middleware for validating request body/params/query using Joi schemas
 */
export const validateWithJoi = (
  schema: Joi.ObjectSchema,
  target: 'body' | 'params' | 'query' = 'body'
) => {
  return (req: Request, res: Response, next: NextFunction): void => {
    const { error, value } = schema.validate(req[target], {
      abortEarly: false,
      stripUnknown: true,
      convert: true
    });

    if (error) {
      const formattedErrors = error.details.map(detail => ({
        field: detail.path.join('.'),
        message: detail.message,
        value: detail.context?.value
      }));

      logger.warn('Joi validation failed', {
        url: req.url,
        method: req.method,
        target,
        errors: formattedErrors,
        input: req[target]
      });

      const validationError = new AppError('Validation failed', 400, 'VALIDATION_ERROR');
      validationError.message = JSON.stringify(formattedErrors);
      return next(validationError);
    }

    // Replace the request data with the validated and cleaned data
    req[target] = value;
    next();
  };
};

/**
 * Sanitize Input Middleware
 * Removes potentially dangerous characters from inputs
 */
export const sanitizeInput = (req: Request, res: Response, next: NextFunction): void => {
  const sanitizeString = (str: string): string => {
    if (typeof str !== 'string') return str;
    
    // Remove potentially dangerous characters
    return str
      .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
      .replace(/<iframe\b[^<]*(?:(?!<\/iframe>)<[^<]*)*<\/iframe>/gi, '')
      .replace(/javascript:/gi, '')
      .replace(/on\w+\s*=/gi, '')
      .trim();
  };

  const sanitizeObject = (obj: any): any => {
    if (obj === null || obj === undefined) return obj;
    
    if (typeof obj === 'string') {
      return sanitizeString(obj);
    }
    
    if (Array.isArray(obj)) {
      return obj.map(sanitizeObject);
    }
    
    if (typeof obj === 'object') {
      const sanitized: any = {};
      for (const [key, value] of Object.entries(obj)) {
        sanitized[key] = sanitizeObject(value);
      }
      return sanitized;
    }
    
    return obj;
  };

  // Sanitize request body, params, and query
  req.body = sanitizeObject(req.body);
  req.params = sanitizeObject(req.params);
  req.query = sanitizeObject(req.query);

  next();
};

/**
 * Rate Limit Validation Middleware
 * Validates rate limit headers
 */
export const validateRateLimit = (req: Request, res: Response, next: NextFunction): void => {
  const rateLimitHeaders = {
    'X-RateLimit-Limit': req.get('X-RateLimit-Limit'),
    'X-RateLimit-Remaining': req.get('X-RateLimit-Remaining'),
    'X-RateLimit-Reset': req.get('X-RateLimit-Reset')
  };

  // Log rate limit information for monitoring
  if (rateLimitHeaders['X-RateLimit-Limit']) {
    logger.debug('Rate limit info', {
      ip: req.ip,
      url: req.url,
      method: req.method,
      rateLimitHeaders
    });
  }

  next();
};

/**
 * Content Type Validation Middleware
 * Ensures request has appropriate content type
 */
export const validateContentType = (allowedTypes: string[]) => {
  return (req: Request, res: Response, next: NextFunction): void => {
    if (req.method === 'GET' || req.method === 'DELETE') {
      return next();
    }

    const contentType = req.get('Content-Type');
    
    if (!contentType) {
      const error = new AppError('Content-Type header is required', 400, 'MISSING_CONTENT_TYPE');
      return next(error);
    }

    const isAllowed = allowedTypes.some(type => 
      contentType.toLowerCase().includes(type.toLowerCase())
    );

    if (!isAllowed) {
      const error = new AppError(
        `Content-Type ${contentType} is not allowed. Allowed types: ${allowedTypes.join(', ')}`,
        415,
        'UNSUPPORTED_MEDIA_TYPE'
      );
      return next(error);
    }

    next();
  };
};

/**
 * Request Size Validation Middleware
 * Validates request payload size
 */
export const validateRequestSize = (maxSize: number) => {
  return (req: Request, res: Response, next: NextFunction): void => {
    const contentLength = parseInt(req.get('Content-Length') || '0');
    
    if (contentLength > maxSize) {
      const error = new AppError(
        `Request size ${contentLength} exceeds maximum allowed size ${maxSize}`,
        413,
        'PAYLOAD_TOO_LARGE'
      );
      return next(error);
    }

    next();
  };
};

// Common validation schemas
export const commonSchemas = {
  // MongoDB ObjectId validation
  objectId: Joi.string().pattern(/^[0-9a-fA-F]{24}$/).message('Invalid ID format'),

  // Email validation
  email: Joi.string().email().required().messages({
    'string.email': 'Please provide a valid email address',
    'any.required': 'Email is required'
  }),

  // Password validation
  password: Joi.string().min(8).max(128).required().messages({
    'string.min': 'Password must be at least 8 characters long',
    'string.max': 'Password cannot exceed 128 characters',
    'any.required': 'Password is required'
  }),

  // Pagination validation
  pagination: Joi.object({
    page: Joi.number().integer().min(1).default(1),
    limit: Joi.number().integer().min(1).max(100).default(10),
    sort: Joi.string().optional(),
    order: Joi.string().valid('asc', 'desc').default('desc')
  }),

  // Date range validation
  dateRange: Joi.object({
    startDate: Joi.date().iso().optional(),
    endDate: Joi.date().iso().min(Joi.ref('startDate')).optional()
  })
};

export default {
  validateRequest,
  validateWithJoi,
  sanitizeInput,
  validateRateLimit,
  validateContentType,
  validateRequestSize,
  commonSchemas
};
