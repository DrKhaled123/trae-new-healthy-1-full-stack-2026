import { Router, Request, Response } from 'express';
import { body } from 'express-validator';
import { catchAsync } from '../middleware/errorHandler';
import { validateRequest } from '../middleware/validation';
import { authenticateToken } from '../middleware/auth';
import { strictLimiter, passwordResetLimiter, createUserLimiter } from '../middleware/security';
import { logger } from '../config/logger';
import {
  createUser,
  authenticateUser,
  getUserById,
  generateTokens,
  verifyRefreshToken
} from '../services/userService';

const router = Router();

/**
 * @swagger
 * components:
 *   schemas:
 *     LoginRequest:
 *       type: object
 *       required:
 *         - email
 *         - password
 *       properties:
 *         email:
 *           type: string
 *           format: email
 *           description: User email address
 *         password:
 *           type: string
 *           minLength: 8
 *           description: User password
 *     
 *     RegisterRequest:
 *       type: object
 *       required:
 *         - email
 *         - password
 *         - firstName
 *         - lastName
 *       properties:
 *         email:
 *           type: string
 *           format: email
 *           description: User email address
 *         password:
 *           type: string
 *           minLength: 8
 *           description: User password
 *         firstName:
 *           type: string
 *           description: User first name
 *         lastName:
 *           type: string
 *           description: User last name
 *     
 *     AuthResponse:
 *       type: object
 *       properties:
 *         success:
 *           type: boolean
 *           example: true
 *         message:
 *           type: string
 *           example: "Login successful"
 *         data:
 *           type: object
 *           properties:
 *             user:
 *               type: object
 *               properties:
 *                 id:
 *                   type: string
 *                 email:
 *                   type: string
 *                 firstName:
 *                   type: string
 *                 lastName:
 *                   type: string
 *                 role:
 *                   type: string
 *             tokens:
 *               type: object
 *               properties:
 *                 accessToken:
 *                   type: string
 *                 refreshToken:
 *                   type: string
 */

/**
 * @swagger
 * /api/auth/register:
 *   post:
 *     summary: Register a new user
 *     tags: [Authentication]
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/RegisterRequest'
 *     responses:
 *       201:
 *         description: User registered successfully
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/AuthResponse'
 *       400:
 *         description: Validation error or user already exists
 *       429:
 *         description: Too many registration attempts
 */
router.post('/register',
  createUserLimiter,
  [
    body('email')
      .isEmail()
      .normalizeEmail()
      .withMessage('Please provide a valid email address'),
    body('password')
      .isLength({ min: 6 })
      .withMessage('Password must be at least 6 characters long'),
    body('firstName')
      .trim()
      .isLength({ min: 2, max: 50 })
      .withMessage('First name must be between 2 and 50 characters'),
    body('lastName')
      .trim()
      .isLength({ min: 2, max: 50 })
      .withMessage('Last name must be between 2 and 50 characters')
  ],
  validateRequest,
  catchAsync(async (req: Request, res: Response) => {
    const { email, password, firstName, lastName } = req.body;
    const fullName = `${firstName} ${lastName}`;

    logger.info('User registration attempt', {
      email,
      ip: req.ip,
      userAgent: req.get('User-Agent')
    });

    try {
      // Create user
      const user = await createUser(fullName, email, password);
      
      // Generate tokens
      const tokens = generateTokens(user);

      logger.info('User registered successfully', {
        userId: user.id,
        email: user.email
      });

      res.status(201).json({
        success: true,
        message: 'User registered successfully',
        data: {
          user: {
            id: user.id,
            name: user.name,
            email: user.email,
            role: user.role,
            createdAt: user.createdAt
          },
          tokens
        }
      });
    } catch (error) {
      logger.error('User registration failed', {
        email,
        error: error instanceof Error ? error.message : 'Unknown error'
      });

      res.status(400).json({
        success: false,
        message: error instanceof Error ? error.message : 'Registration failed'
      });
    }
  })
);

/**
 * @swagger
 * /api/auth/login:
 *   post:
 *     summary: Login user
 *     tags: [Authentication]
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             $ref: '#/components/schemas/LoginRequest'
 *     responses:
 *       200:
 *         description: Login successful
 *         content:
 *           application/json:
 *             schema:
 *               $ref: '#/components/schemas/AuthResponse'
 *       401:
 *         description: Invalid credentials
 *       429:
 *         description: Too many login attempts
 */
router.post('/login',
  strictLimiter,
  [
    body('email')
      .isEmail()
      .normalizeEmail()
      .withMessage('Please provide a valid email address'),
    body('password')
      .notEmpty()
      .withMessage('Password is required')
  ],
  validateRequest,
  catchAsync(async (req: Request, res: Response) => {
    const { email, password } = req.body;

    logger.info('User login attempt', {
      email,
      ip: req.ip,
      userAgent: req.get('User-Agent')
    });

    try {
      // Authenticate user
      const user = await authenticateUser(email, password);
      
      // Generate tokens
      const tokens = generateTokens(user);

      logger.info('User logged in successfully', {
        userId: user.id,
        email: user.email
      });

      res.status(200).json({
        success: true,
        message: 'Login successful',
        data: {
          user: {
            id: user.id,
            name: user.name,
            email: user.email,
            role: user.role,
            lastLogin: user.lastLogin
          },
          tokens
        }
      });
    } catch (error) {
      logger.error('User login failed', {
        email,
        error: error instanceof Error ? error.message : 'Unknown error'
      });

      res.status(401).json({
        success: false,
        message: error instanceof Error ? error.message : 'Login failed'
      });
    }
  })
);

/**
 * @swagger
 * /api/auth/logout:
 *   post:
 *     summary: Logout user
 *     tags: [Authentication]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: Logout successful
 *       401:
 *         description: Unauthorized
 */
router.post('/logout',
  authenticateToken,
  catchAsync(async (req: Request, res: Response) => {
    // This is a placeholder - in a real implementation, you would:
    // 1. Invalidate the refresh token
    // 2. Add the token to a blacklist (optional)
    // 3. Log the logout event

    logger.info('User logout', {
      userId: req.user?.id,
      ip: req.ip
    });

    res.status(200).json({
      success: true,
      message: 'Logout successful'
    });
  })
);

/**
 * @swagger
 * /api/auth/refresh:
 *   post:
 *     summary: Refresh access token
 *     tags: [Authentication]
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - refreshToken
 *             properties:
 *               refreshToken:
 *                 type: string
 *                 description: Refresh token
 *     responses:
 *       200:
 *         description: Token refreshed successfully
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
 *                   example: "Token refreshed successfully"
 *                 data:
 *                   type: object
 *                   properties:
 *                     accessToken:
 *                       type: string
 *                     refreshToken:
 *                       type: string
 *       401:
 *         description: Invalid refresh token
 */
router.post('/refresh',
  [
    body('refreshToken')
      .notEmpty()
      .withMessage('Refresh token is required')
  ],
  validateRequest,
  catchAsync(async (req: Request, res: Response) => {
    const { refreshToken } = req.body;

    logger.info('Token refresh attempt', {
      ip: req.ip,
      userAgent: req.get('User-Agent')
    });

    try {
      // Verify refresh token and get user data
      const decoded = verifyRefreshToken(refreshToken);
      
      // Get user from database to ensure they still exist
      const user = getUserById(decoded.id);
      
      // Generate new tokens
      const tokens = generateTokens(user);

      logger.info('Token refreshed successfully', {
        userId: user.id,
        email: user.email
      });

      res.status(200).json({
        success: true,
        message: 'Token refreshed successfully',
        data: tokens
      });
    } catch (error) {
      logger.error('Token refresh failed', {
        error: error instanceof Error ? error.message : 'Unknown error'
      });

      res.status(401).json({
        success: false,
        message: error instanceof Error ? error.message : 'Token refresh failed'
      });
    }
  })
);

/**
 * @swagger
 * /api/auth/forgot-password:
 *   post:
 *     summary: Request password reset
 *     tags: [Authentication]
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - email
 *             properties:
 *               email:
 *                 type: string
 *                 format: email
 *                 description: User email address
 *     responses:
 *       200:
 *         description: Password reset email sent
 *       404:
 *         description: User not found
 *       429:
 *         description: Too many password reset attempts
 */
router.post('/forgot-password',
  passwordResetLimiter,
  [
    body('email')
      .isEmail()
      .normalizeEmail()
      .withMessage('Please provide a valid email address')
  ],
  validateRequest,
  catchAsync(async (req: Request, res: Response) => {
    const { email } = req.body;

    // This is a placeholder - in a real implementation, you would:
    // 1. Find user by email
    // 2. Generate password reset token
    // 3. Send reset email
    // 4. Store token with expiration

    logger.info('Password reset request', {
      email,
      ip: req.ip,
      userAgent: req.get('User-Agent')
    });

    res.status(200).json({
      success: true,
      message: 'Password reset email sent'
    });
  })
);

/**
 * @swagger
 * /api/auth/reset-password:
 *   post:
 *     summary: Reset password with token
 *     tags: [Authentication]
 *     requestBody:
 *       required: true
 *       content:
 *         application/json:
 *           schema:
 *             type: object
 *             required:
 *               - token
 *               - password
 *             properties:
 *               token:
 *                 type: string
 *                 description: Password reset token
 *               password:
 *                 type: string
 *                 minLength: 8
 *                 description: New password
 *     responses:
 *       200:
 *         description: Password reset successful
 *       400:
 *         description: Invalid or expired token
 */
router.post('/reset-password',
  [
    body('token')
      .notEmpty()
      .withMessage('Reset token is required'),
    body('password')
      .isLength({ min: 8 })
      .withMessage('Password must be at least 8 characters long')
      .matches(/^(?=.*[a-z])(?=.*[A-Z])(?=.*\d)/)
      .withMessage('Password must contain at least one uppercase letter, one lowercase letter, and one number')
  ],
  validateRequest,
  catchAsync(async (req: Request, res: Response) => {
    const { token, password } = req.body;

    // This is a placeholder - in a real implementation, you would:
    // 1. Verify the reset token
    // 2. Check if token is not expired
    // 3. Hash the new password
    // 4. Update user password
    // 5. Invalidate all existing tokens
    // 6. Delete the reset token

    logger.info('Password reset attempt', {
      ip: req.ip,
      userAgent: req.get('User-Agent')
    });

    res.status(200).json({
      success: true,
      message: 'Password reset successful'
    });
  })
);

/**
 * @swagger
 * /api/auth/me:
 *   get:
 *     summary: Get current user profile
 *     tags: [Authentication]
 *     security:
 *       - bearerAuth: []
 *     responses:
 *       200:
 *         description: User profile retrieved successfully
 *         content:
 *           application/json:
 *             schema:
 *               type: object
 *               properties:
 *                 success:
 *                   type: boolean
 *                   example: true
 *                 data:
 *                   type: object
 *                   properties:
 *                     user:
 *                       type: object
 *                       properties:
 *                         id:
 *                           type: string
 *                         email:
 *                           type: string
 *                         firstName:
 *                           type: string
 *                         lastName:
 *                           type: string
 *                         role:
 *                           type: string
 *       401:
 *         description: Unauthorized
 */
router.get('/me',
  authenticateToken,
  catchAsync(async (req: Request, res: Response) => {
    logger.debug('Get user profile', {
      userId: req.user?.id,
      ip: req.ip
    });

    try {
      // Get user from database
      const user = getUserById(req.user!.id);

      res.status(200).json({
        success: true,
        data: { 
          user: {
            id: user.id,
            name: user.name,
            email: user.email,
            role: user.role,
            createdAt: user.createdAt,
            lastLogin: user.lastLogin
          }
        }
      });
    } catch (error) {
      logger.error('Get user profile failed', {
        userId: req.user?.id,
        error: error instanceof Error ? error.message : 'Unknown error'
      });

      res.status(404).json({
        success: false,
        message: error instanceof Error ? error.message : 'User not found'
      });
    }
  })
);

export default router;
