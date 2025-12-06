import bcrypt from 'bcryptjs';
import { config } from '../config/env';
import { logger } from '../config/logger';

// In-memory user storage (in production, use a proper database)
export interface User {
  id: string;
  name: string;
  email: string;
  password: string;
  role: string;
  createdAt: string;
  lastLogin?: string;
}

// In-memory user store
let users: User[] = [];

// Helper functions
const generateId = (): string => {
  return Date.now().toString(36) + Math.random().toString(36).substr(2);
};

const findUserByEmail = (email: string): User | undefined => {
  return users.find(user => user.email.toLowerCase() === email.toLowerCase());
};

const findUserById = (id: string): User | undefined => {
  return users.find(user => user.id === id);
};

// User service functions
export const createUser = async (name: string, email: string, password: string) => {
  // Check if user already exists
  const existingUser = findUserByEmail(email);
  if (existingUser) {
    throw new Error('User with this email already exists');
  }

  // Hash password
  const hashedPassword = await bcrypt.hash(password, config.BCRYPT_ROUNDS);

  // Create new user
  const newUser: User = {
    id: generateId(),
    name: name.trim(),
    email: email.toLowerCase().trim(),
    password: hashedPassword,
    role: 'user',
    createdAt: new Date().toISOString()
  };

  // Store user
  users.push(newUser);

  logger.info('User created successfully', {
    userId: newUser.id,
    email: newUser.email
  });

  // Return user without password
  const { password: _, ...userWithoutPassword } = newUser;
  return userWithoutPassword;
};

export const authenticateUser = async (email: string, password: string) => {
  // Find user by email
  const user = findUserByEmail(email);
  if (!user) {
    throw new Error('Invalid email or password');
  }

  // Check password
  const isPasswordValid = await bcrypt.compare(password, user.password);
  if (!isPasswordValid) {
    throw new Error('Invalid email or password');
  }

  // Update last login
  user.lastLogin = new Date().toISOString();

  logger.info('User authenticated successfully', {
    userId: user.id,
    email: user.email
  });

  // Return user without password
  const { password: _, ...userWithoutPassword } = user;
  return userWithoutPassword;
};

export const getUserById = (id: string) => {
  const user = findUserById(id);
  if (!user) {
    throw new Error('User not found');
  }

  // Return user without password
  const { password: _, ...userWithoutPassword } = user;
  return userWithoutPassword;
};

export const generateTokens = (user: { id: string; email: string; role: string }) => {
  const payload = {
    id: user.id,
    email: user.email,
    role: user.role
  };

  const accessToken = jwt.sign(payload, config.JWT_SECRET, {
    expiresIn: config.JWT_EXPIRES_IN
  });

  const refreshToken = jwt.sign(payload, config.JWT_REFRESH_SECRET, {
    expiresIn: config.JWT_REFRESH_EXPIRES_IN
  });

  return {
    accessToken,
    refreshToken
  };
};

export const verifyRefreshToken = (refreshToken: string) => {
  try {
    const decoded = jwt.verify(refreshToken, config.JWT_REFRESH_SECRET) as any;
    return decoded;
  } catch (error) {
    throw new Error('Invalid or expired refresh token');
  }
};

export const createDemoUser = async () => {
  try {
    // Create a demo user for testing
    const demoUser = await createUser('Demo User', 'demo@nutrition.com', 'demo123');
    logger.info('Demo user created', { email: demoUser.email });
    return demoUser;
  } catch (error) {
    // Demo user might already exist
    logger.info('Demo user already exists or creation failed');
  }
};

// Initialize demo user on service start
createDemoUser();
