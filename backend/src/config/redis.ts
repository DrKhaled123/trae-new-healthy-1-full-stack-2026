import { createClient, RedisClientType } from 'redis';
import { createLogger } from './logger';

const logger = createLogger('redis');

// Redis client instance
let redisClient: RedisClientType;

// Redis configuration
const redisConfig = {
  url: process.env.REDIS_URL || 'redis://localhost:6379',
  password: process.env.REDIS_PASSWORD || undefined,
  socket: {
    reconnectStrategy: (retries: number) => {
      if (retries > 10) {
        logger.error('Redis reconnection failed after 10 attempts');
        return new Error('Redis reconnection failed');
      }
      return Math.min(retries * 50, 500);
    },
  },
};

// Connect to Redis
export const connectRedis = async (): Promise<void> => {
  try {
    redisClient = createClient(redisConfig);

    redisClient.on('error', (error) => {
      logger.error('Redis client error:', error);
    });

    redisClient.on('connect', () => {
      logger.info('Redis client connected');
    });

    redisClient.on('ready', () => {
      logger.info('Redis client ready');
    });

    redisClient.on('end', () => {
      logger.info('Redis client disconnected');
    });

    await redisClient.connect();
    logger.info('Redis connected successfully');
  } catch (error) {
    logger.error('Redis connection failed:', error);
    // Don't exit process, just continue without Redis
    logger.warn('Application will continue without Redis caching');
  }
};

// Get Redis client
export const getRedisClient = (): RedisClientType | null => {
  return redisClient || null;
};

// Health check for Redis
export const checkRedisHealth = async (): Promise<boolean> => {
  try {
    if (!redisClient) return false;
    await redisClient.ping();
    return true;
  } catch (error) {
    logger.error('Redis health check failed:', error);
    return false;
  }
};

// Cache service with Redis fallback to memory
class CacheService {
  private memoryCache: Map<string, { data: any; expiry: number }> = new Map();

  // Set cache with TTL (in seconds)
  async set(key: string, value: any, ttl: number = 3600): Promise<void> {
    try {
      if (redisClient?.isOpen) {
        await redisClient.setEx(key, ttl, JSON.stringify(value));
      } else {
        // Fallback to memory cache
        const expiry = Date.now() + ttl * 1000;
        this.memoryCache.set(key, { data: value, expiry });
      }
    } catch (error) {
      logger.error('Cache set error:', error);
      // Fallback to memory cache
      const expiry = Date.now() + ttl * 1000;
      this.memoryCache.set(key, { data: value, expiry });
    }
  }

  // Get cache value
  async get<T>(key: string): Promise<T | null> {
    try {
      if (redisClient?.isOpen) {
        const value = await redisClient.get(key);
        return value ? JSON.parse(value) : null;
      } else {
        // Fallback to memory cache
        const cached = this.memoryCache.get(key);
        if (!cached) return null;
        
        if (Date.now() > cached.expiry) {
          this.memoryCache.delete(key);
          return null;
        }
        
        return cached.data;
      }
    } catch (error) {
      logger.error('Cache get error:', error);
      return null;
    }
  }

  // Delete cache key
  async del(key: string): Promise<void> {
    try {
      if (redisClient?.isOpen) {
        await redisClient.del(key);
      } else {
        this.memoryCache.delete(key);
      }
    } catch (error) {
      logger.error('Cache delete error:', error);
    }
  }

  // Clear all cache
  async clear(): Promise<void> {
    try {
      if (redisClient?.isOpen) {
        await redisClient.flushAll();
      } else {
        this.memoryCache.clear();
      }
    } catch (error) {
      logger.error('Cache clear error:', error);
    }
  }

  // Check if key exists
  async exists(key: string): Promise<boolean> {
    try {
      if (redisClient?.isOpen) {
        return (await redisClient.exists(key)) === 1;
      } else {
        const cached = this.memoryCache.get(key);
        if (!cached) return false;
        
        if (Date.now() > cached.expiry) {
          this.memoryCache.delete(key);
          return false;
        }
        
        return true;
      }
    } catch (error) {
      logger.error('Cache exists error:', error);
      return false;
    }
  }

  // Set cache with pattern (for invalidating multiple keys)
  async setWithPattern(pattern: string, key: string, value: any, ttl: number = 3600): Promise<void> {
    try {
      await this.set(key, value, ttl);
      
      // Store pattern-key mapping for invalidation
      if (redisClient?.isOpen) {
        await redisClient.sAdd(`pattern:${pattern}`, key);
        await redisClient.expire(`pattern:${pattern}`, ttl);
      }
    } catch (error) {
      logger.error('Cache set with pattern error:', error);
    }
  }

  // Invalidate cache by pattern
  async invalidateByPattern(pattern: string): Promise<void> {
    try {
      if (redisClient?.isOpen) {
        const keys = await redisClient.sMembers(`pattern:${pattern}`);
        if (keys.length > 0) {
          await redisClient.del(...keys);
          await redisClient.del(`pattern:${pattern}`);
        }
      }
    } catch (error) {
      logger.error('Cache invalidate by pattern error:', error);
    }
  }
}

// Export cache service instance
export const cache = new CacheService();

// Graceful shutdown
process.on('beforeExit', async () => {
  if (redisClient?.isOpen) {
    logger.info('Closing Redis connection...');
    await redisClient.quit();
    logger.info('Redis connection closed.');
  }
});

export default redisClient;
