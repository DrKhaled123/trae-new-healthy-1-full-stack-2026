import dotenv from 'dotenv';
import path from 'path';

// Load environment variables from .env file
dotenv.config({ path: path.join(__dirname, '../../.env') });

// Environment configuration interface
interface EnvConfig {
  // Server
  NODE_ENV: string;
  PORT: number;
  API_VERSION: string;
  
  // Database
  DATABASE_URL: string;
  
  // Redis
  REDIS_URL: string;
  REDIS_PASSWORD?: string;
  
  // JWT
  JWT_SECRET: string;
  JWT_REFRESH_SECRET: string;
  JWT_EXPIRES_IN: string;
  JWT_REFRESH_EXPIRES_IN: string;
  
  // CORS
  CORS_ORIGIN: string;
  CORS_CREDENTIALS: boolean;
  
  // Rate Limiting
  RATE_LIMIT_WINDOW_MS: number;
  RATE_LIMIT_MAX_REQUESTS: number;
  
  // File Upload
  UPLOAD_DIR: string;
  MAX_FILE_SIZE: number;
  ALLOWED_FILE_TYPES: string[];
  
  // Email
  SMTP_HOST: string;
  SMTP_PORT: number;
  SMTP_USER: string;
  SMTP_PASS: string;
  FROM_EMAIL: string;
  FROM_NAME: string;
  
  // Logging
  LOG_LEVEL: string;
  LOG_FILE: string;
  LOG_MAX_SIZE: string;
  LOG_MAX_FILES: string;
  
  // Security
  BCRYPT_ROUNDS: number;
  SESSION_SECRET: string;
  
  // API Documentation
  API_DOCS_ENABLED: boolean;
  API_DOCS_PATH: string;
  
  // Monitoring
  HEALTH_CHECK_ENABLED: boolean;
  METRICS_ENABLED: boolean;
  
  // Development
  DEBUG: boolean;
}

// Validate required environment variables
const validateEnv = (): void => {
  const required = [
    'DATABASE_URL',
    'JWT_SECRET',
    'JWT_REFRESH_SECRET',
    'SESSION_SECRET',
  ];

  const missing = required.filter(key => !process.env[key]);

  if (missing.length > 0) {
    throw new Error(`Missing required environment variables: ${missing.join(', ')}`);
  }
};

// Parse environment variables
const parseEnv = (): EnvConfig => {
  // Validate required variables first
  validateEnv();

  return {
    // Server
    NODE_ENV: process.env.NODE_ENV || 'development',
    PORT: parseInt(process.env.PORT || '3001', 10),
    API_VERSION: process.env.API_VERSION || 'v1',
    
    // Database
    DATABASE_URL: process.env.DATABASE_URL!,
    
    // Redis
    REDIS_URL: process.env.REDIS_URL || 'redis://localhost:6379',
    REDIS_PASSWORD: process.env.REDIS_PASSWORD,
    
    // JWT
    JWT_SECRET: process.env.JWT_SECRET!,
    JWT_REFRESH_SECRET: process.env.JWT_REFRESH_SECRET!,
    JWT_EXPIRES_IN: process.env.JWT_EXPIRES_IN || '15m',
    JWT_REFRESH_EXPIRES_IN: process.env.JWT_REFRESH_EXPIRES_IN || '7d',
    
    // CORS
    CORS_ORIGIN: process.env.CORS_ORIGIN || 'http://localhost:3000',
    CORS_CREDENTIALS: process.env.CORS_CREDENTIALS === 'true',
    
    // Rate Limiting
    RATE_LIMIT_WINDOW_MS: parseInt(process.env.RATE_LIMIT_WINDOW_MS || '900000', 10), // 15 minutes
    RATE_LIMIT_MAX_REQUESTS: parseInt(process.env.RATE_LIMIT_MAX_REQUESTS || '100', 10),
    
    // File Upload
    UPLOAD_DIR: process.env.UPLOAD_DIR || 'uploads',
    MAX_FILE_SIZE: parseInt(process.env.MAX_FILE_SIZE || '5242880', 10), // 5MB
    ALLOWED_FILE_TYPES: (process.env.ALLOWED_FILE_TYPES || 'image/jpeg,image/png,image/webp').split(','),
    
    // Email
    SMTP_HOST: process.env.SMTP_HOST || 'smtp.gmail.com',
    SMTP_PORT: parseInt(process.env.SMTP_PORT || '587', 10),
    SMTP_USER: process.env.SMTP_USER || '',
    SMTP_PASS: process.env.SMTP_PASS || '',
    FROM_EMAIL: process.env.FROM_EMAIL || 'noreply@nutritrack.com',
    FROM_NAME: process.env.FROM_NAME || 'NutriTrack',
    
    // Logging
    LOG_LEVEL: process.env.LOG_LEVEL || 'info',
    LOG_FILE: process.env.LOG_FILE || 'logs',
    LOG_MAX_SIZE: process.env.LOG_MAX_SIZE || '20m',
    LOG_MAX_FILES: process.env.LOG_MAX_FILES || '14d',
    
    // Security
    BCRYPT_ROUNDS: parseInt(process.env.BCRYPT_ROUNDS || '12', 10),
    SESSION_SECRET: process.env.SESSION_SECRET!,
    
    // API Documentation
    API_DOCS_ENABLED: process.env.API_DOCS_ENABLED === 'true',
    API_DOCS_PATH: process.env.API_DOCS_PATH || '/api-docs',
    
    // Monitoring
    HEALTH_CHECK_ENABLED: process.env.HEALTH_CHECK_ENABLED === 'true',
    METRICS_ENABLED: process.env.METRICS_ENABLED === 'true',
    
    // Development
    DEBUG: process.env.DEBUG === 'true',
  };
};

// Export parsed configuration
export const config = parseEnv();

// Helper functions
export const isDevelopment = (): boolean => config.NODE_ENV === 'development';
export const isProduction = (): boolean => config.NODE_ENV === 'production';
export const isTest = (): boolean => config.NODE_ENV === 'test';

// Get frontend URL (for CORS, emails, etc.)
export const getFrontendUrl = (): string => {
  return config.CORS_ORIGIN;
};

// Get backend URL (for API calls, webhooks, etc.)
export const getBackendUrl = (): string => {
  const protocol = config.NODE_ENV === 'production' ? 'https' : 'http';
  const host = process.env.HOST || 'localhost';
  return `${protocol}://${host}:${config.PORT}`;
};

export default config;
