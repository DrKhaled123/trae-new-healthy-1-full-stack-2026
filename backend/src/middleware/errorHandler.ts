import { Request, Response, NextFunction } from 'express';
import { logger } from '../config/logger';

// Custom error class for application errors
export class AppError extends Error {
  public statusCode: number;
  public isOperational: boolean;
  public code?: string;

  constructor(message: string, statusCode: number, code?: string) {
    super(message);
    this.statusCode = statusCode;
    this.isOperational = true;
    this.code = code;

    Error.captureStackTrace(this, this.constructor);
  }
}

// Error response interface
interface ErrorResponse {
  success: false;
  message: string;
  code?: string;
  errors?: any[];
  stack?: string;
}

/**
 * Global Error Handling Middleware
 * Catches all errors and formats them consistently
 */
export const errorHandler = (
  error: Error | AppError,
  req: Request,
  res: Response,
  next: NextFunction
): void => {
  let err = error as AppError;

  // Set default values for non-operational errors
  if (!err.isOperational) {
    err.statusCode = 500;
    err.message = 'Something went wrong';
  }

  // Log error details
  logger.error('Error occurred:', {
    message: err.message,
    statusCode: err.statusCode,
    code: err.code,
    stack: err.stack,
    url: req.url,
    method: req.method,
    ip: req.ip,
    userAgent: req.get('User-Agent'),
    body: req.body,
    params: req.params,
    query: req.query
  });

  // Handle specific error types
  if (error.name === 'ValidationError') {
    handleValidationError(error, req, res);
  } else if (error.name === 'CastError') {
    handleCastError(error, req, res);
  } else if (error.code === 11000) {
    handleDuplicateFieldsError(error, req, res);
  } else if (error.name === 'JsonWebTokenError') {
    handleJWTError(error, req, res);
  } else if (error.name === 'TokenExpiredError') {
    handleJWTExpiredError(error, req, res);
  } else {
    // Handle general application errors
    sendErrorResponse(err, req, res);
  }
};

/**
 * Handle Mongoose Validation Errors
 */
const handleValidationError = (error: any, req: Request, res: Response): void => {
  const errors = Object.values(error.errors).map((err: any) => ({
    field: err.path,
    message: err.message,
    value: err.value
  }));

  const response: ErrorResponse = {
    success: false,
    message: 'Validation Error',
    code: 'VALIDATION_ERROR',
    errors
  };

  res.status(400).json(response);
};

/**
 * Handle Mongoose Cast Errors (invalid ObjectId)
 */
const handleCastError = (error: any, req: Request, res: Response): void => {
  const response: ErrorResponse = {
    success: false,
    message: `Invalid ${error.path}: ${error.value}`,
    code: 'INVALID_ID'
  };

  res.status(400).json(response);
};

/**
 * Handle Duplicate Field Errors
 */
const handleDuplicateFieldsError = (error: any, req: Request, res: Response): void => {
  const field = Object.keys(error.keyValue)[0];
  const value = error.keyValue[field];

  const response: ErrorResponse = {
    success: false,
    message: `Duplicate field value: ${field} with value: ${value}`,
    code: 'DUPLICATE_FIELD'
  };

  res.status(400).json(response);
};

/**
 * Handle JWT Errors
 */
const handleJWTError = (error: any, req: Request, res: Response): void => {
  const response: ErrorResponse = {
    success: false,
    message: 'Invalid token. Please log in again',
    code: 'INVALID_TOKEN'
  };

  res.status(401).json(response);
};

/**
 * Handle JWT Expired Errors
 */
const handleJWTExpiredError = (error: any, req: Request, res: Response): void => {
  const response: ErrorResponse = {
    success: false,
    message: 'Your token has expired. Please log in again',
    code: 'TOKEN_EXPIRED'
  };

  res.status(401).json(response);
};

/**
 * Send Error Response
 */
const sendErrorResponse = (err: AppError, req: Request, res: Response): void => {
  const response: ErrorResponse = {
    success: false,
    message: err.message,
    code: err.code
  };

  // Add stack trace in development
  if (process.env.NODE_ENV === 'development') {
    response.stack = err.stack;
  }

  // Add validation errors if they exist
  if (err.message.includes('validation') && err.stack) {
    try {
      const parsedErrors = JSON.parse(err.message);
      response.errors = parsedErrors;
    } catch {
      // Ignore parsing errors
    }
  }

  res.status(err.statusCode).json(response);
};

/**
 * Async Error Wrapper
 * Wraps async functions to catch errors automatically
 */
export const catchAsync = (fn: Function) => {
  return (req: Request, res: Response, next: NextFunction) => {
    Promise.resolve(fn(req, res, next)).catch(next);
  };
};

/**
 * 404 Not Found Handler
 */
export const notFoundHandler = (req: Request, res: Response, next: NextFunction): void => {
  const error = new AppError(`Route ${req.originalUrl} not found`, 404, 'NOT_FOUND');
  next(error);
};

/**
 * Unhandled Promise Rejection Handler
 */
export const handleUnhandledRejections = (): void => {
  process.on('unhandledRejection', (reason: Error, promise: Promise<any>) => {
    logger.error('Unhandled Promise Rejection:', {
      reason: reason.message,
      stack: reason.stack,
      promise
    });

    // Close server gracefully
    logger.info('Shutting down due to unhandled promise rejection...');
    process.exit(1);
  });
};

/**
 * Uncaught Exception Handler
 */
export const handleUncaughtExceptions = (): void => {
  process.on('uncaughtException', (error: Error) => {
    logger.error('Uncaught Exception:', {
      message: error.message,
      stack: error.stack
    });

    // Close server gracefully
    logger.info('Shutting down due to uncaught exception...');
    process.exit(1);
  });
};
