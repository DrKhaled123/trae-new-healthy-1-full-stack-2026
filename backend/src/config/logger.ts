import winston from 'winston';
import DailyRotateFile from 'winston-daily-rotate-file';
import path from 'path';

// Define log levels
const levels = {
  error: 0,
  warn: 1,
  info: 2,
  http: 3,
  debug: 4,
};

// Define colors for each level
const colors = {
  error: 'red',
  warn: 'yellow',
  info: 'green',
  http: 'magenta',
  debug: 'white',
};

// Add colors to winston
winston.addColors(colors);

// Define log format
const format = winston.format.combine(
  winston.format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss:ms' }),
  winston.format.colorize({ all: true }),
  winston.format.printf(
    (info) => `${info.timestamp} ${info.level}: ${info.message}`,
  ),
);

// Define transports
const transports: winston.transport[] = [
  // Console transport
  new winston.transports.Console({
    format: winston.format.combine(
      winston.format.colorize(),
      winston.format.simple(),
    ),
  }),
];

// Add file transports if in production
if (process.env.NODE_ENV === 'production') {
  // Ensure logs directory exists
  const logDir = process.env.LOG_FILE || 'logs';
  
  transports.push(
    // Error log file
    new DailyRotateFile({
      filename: path.join(logDir, 'error-%DATE%.log'),
      datePattern: 'YYYY-MM-DD',
      level: 'error',
      maxSize: process.env.LOG_MAX_SIZE || '20m',
      maxFiles: process.env.LOG_MAX_FILES || '14d',
      format: winston.format.combine(
        winston.format.timestamp(),
        winston.format.json(),
      ),
    }),
    
    // Combined log file
    new DailyRotateFile({
      filename: path.join(logDir, 'combined-%DATE%.log'),
      datePattern: 'YYYY-MM-DD',
      maxSize: process.env.LOG_MAX_SIZE || '20m',
      maxFiles: process.env.LOG_MAX_FILES || '14d',
      format: winston.format.combine(
        winston.format.timestamp(),
        winston.format.json(),
      ),
    }),
  );
}

// Create logger instance
const logger = winston.createLogger({
  level: process.env.LOG_LEVEL || 'info',
  levels,
  format,
  transports,
  exitOnError: false,
});

// Create a factory function for creating named loggers
export const createLogger = (service: string) => {
  return {
    error: (message: string, meta?: any) => {
      logger.error(`[${service}] ${message}`, meta);
    },
    warn: (message: string, meta?: any) => {
      logger.warn(`[${service}] ${message}`, meta);
    },
    info: (message: string, meta?: any) => {
      logger.info(`[${service}] ${message}`, meta);
    },
    http: (message: string, meta?: any) => {
      logger.http(`[${service}] ${message}`, meta);
    },
    debug: (message: string, meta?: any) => {
      logger.debug(`[${service}] ${message}`, meta);
    },
  };
};

// Stream for Morgan HTTP logger
export const logStream = {
  write: (message: string) => {
    logger.http(message.trim());
  },
};

export default logger;
