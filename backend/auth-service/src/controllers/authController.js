const jwt = require('jsonwebtoken');
const bcrypt = require('bcryptjs');
const { validationResult } = require('express-validator');
const User = require('../models/user');
const { createLogger, format, transports } = require('winston');
const redis = require('redis');

// Initialize Redis client
const redisClient = redis.createClient({
  url: process.env.REDIS_URL || 'redis://localhost:6379'
});

redisClient.on('error', (err) => console.log('Redis Client Error', err));
redisClient.connect();

// Initialize logger
const logger = createLogger({
  format: format.combine(
    format.timestamp(),
    format.json()
  ),
  transports: [
    new transports.Console(),
    new transports.File({ filename: 'auth.log' })
  ]
});

// Generate JWT tokens
const generateTokens = async (user) => {
  const accessToken = jwt.sign(
    { 
      id: user.id,
      email: user.email,
      roles: user.roles 
    },
    process.env.JWT_SECRET,
    { expiresIn: '15m' }
  );

  const refreshToken = jwt.sign(
    { id: user.id },
    process.env.JWT_REFRESH_SECRET,
    { expiresIn: '7d' }
  );

  // Store refresh token in Redis
  await redisClient.set(
    `refresh_token:${user.id}`,
    refreshToken,
    'EX',
    7 * 24 * 60 * 60 // 7 days
  );

  return { accessToken, refreshToken };
};

const authController = {
  // User registration
  register: async (req, res) => {
    try {
      const errors = validationResult(req);
      if (!errors.isEmpty()) {
        return res.status(400).json({ errors: errors.array() });
      }

      const { email, password, name } = req.body;

      // Check if user exists
      const existingUser = await User.findOne({ where: { email } });
      if (existingUser) {
        return res.status(400).json({
          status: 'error',
          message: 'User already exists'
        });
      }

      // Hash password
      const salt = await bcrypt.genSalt(10);
      const hashedPassword = await bcrypt.hash(password, salt);

      // Create user
      const user = await User.create({
        email,
        password: hashedPassword,
        name,
        roles: ['user']
      });

      const tokens = await generateTokens(user);

      res.status(201).json({
        status: 'success',
        data: {
          user: {
            id: user.id,
            email: user.email,
            name: user.name,
            roles: user.roles
          },
          ...tokens
        }
      });
    } catch (error) {
      logger.error('Registration error:', error);
      res.status(500).json({
        status: 'error',
        message: 'Error registering user'
      });
    }
  },

  // User login
  login: async (req, res) => {
    try {
      const errors = validationResult(req);
      if (!errors.isEmpty()) {
        return res.status(400).json({ errors: errors.array() });
      }

      const { email, password } = req.body;

      // Find user
      const user = await User.findOne({ where: { email } });
      if (!user) {
        return res.status(401).json({
          status: 'error',
          message: 'Invalid credentials'
        });
      }

      // Verify password
      const isValidPassword = await bcrypt.compare(password, user.password);
      if (!isValidPassword) {
        return res.status(401).json({
          status: 'error',
          message: 'Invalid credentials'
        });
      }

      const tokens = await generateTokens(user);

      res.json({
        status: 'success',
        data: {
          user: {
            id: user.id,
            email: user.email,
            name: user.name,
            roles: user.roles
          },
          ...tokens
        }
      });
    } catch (error) {
      logger.error('Login error:', error);
      res.status(500).json({
        status: 'error',
        message: 'Error logging in'
      });
    }
  },

  // OAuth callback
  oauthCallback: async (req, res) => {
    try {
      const tokens = await generateTokens(req.user);
      
      // Redirect to frontend with tokens
      res.redirect(`${process.env.FRONTEND_URL}/auth/callback?` + 
        `access_token=${tokens.accessToken}&` +
        `refresh_token=${tokens.refreshToken}`);
    } catch (error) {
      logger.error('OAuth callback error:', error);
      res.redirect(`${process.env.FRONTEND_URL}/auth/error`);
    }
  },

  // Get user profile
  getProfile: async (req, res) => {
    try {
      const user = await User.findByPk(req.user.id);
      if (!user) {
        return res.status(404).json({
          status: 'error',
          message: 'User not found'
        });
      }

      res.json({
        status: 'success',
        data: {
          user: {
            id: user.id,
            email: user.email,
            name: user.name,
            roles: user.roles
          }
        }
      });
    } catch (error) {
      logger.error('Get profile error:', error);
      res.status(500).json({
        status: 'error',
        message: 'Error fetching profile'
      });
    }
  },

  // Logout
  logout: async (req, res) => {
    try {
      const authHeader = req.headers.authorization;
      if (!authHeader) {
        return res.status(401).json({
          status: 'error',
          message: 'No token provided'
        });
      }

      const token = authHeader.split(' ')[1];
      const decoded = jwt.verify(token, process.env.JWT_SECRET);

      // Remove refresh token from Redis
      await redisClient.del(`refresh_token:${decoded.id}`);

      res.json({
        status: 'success',
        message: 'Successfully logged out'
      });
    } catch (error) {
      logger.error('Logout error:', error);
      res.status(500).json({
        status: 'error',
        message: 'Error logging out'
      });
    }
  },

  // Refresh token
  refreshToken: async (req, res) => {
    try {
      const { refresh_token } = req.body;
      if (!refresh_token) {
        return res.status(400).json({
          status: 'error',
          message: 'Refresh token required'
        });
      }

      const decoded = jwt.verify(refresh_token, process.env.JWT_REFRESH_SECRET);
      const storedToken = await redisClient.get(`refresh_token:${decoded.id}`);

      if (!storedToken || storedToken !== refresh_token) {
        return res.status(401).json({
          status: 'error',
          message: 'Invalid refresh token'
        });
      }

      const user = await User.findByPk(decoded.id);
      if (!user) {
        return res.status(404).json({
          status: 'error',
          message: 'User not found'
        });
      }

      const tokens = await generateTokens(user);

      res.json({
        status: 'success',
        data: tokens
      });
    } catch (error) {
      logger.error('Refresh token error:', error);
      res.status(500).json({
        status: 'error',
        message: 'Error refreshing token'
      });
    }
  },

  // Check role middleware
  checkRole: (roles) => {
    return (req, res, next) => {
      if (!req.user.roles) {
        return res.status(403).json({
          status: 'error',
          message: 'No roles defined'
        });
      }

      const hasRole = roles.some(role => req.user.roles.includes(role));
      if (!hasRole) {
        return res.status(403).json({
          status: 'error',
          message: 'Not authorized'
        });
      }

      next();
    };
  },

  // Assign role
  assignRole: async (req, res) => {
    try {
      const { userId, role } = req.body;

      const user = await User.findByPk(userId);
      if (!user) {
        return res.status(404).json({
          status: 'error',
          message: 'User not found'
        });
      }

      // Add role if it doesn't exist
      if (!user.roles.includes(role)) {
        user.roles = [...user.roles, role];
        await user.save();
      }

      res.json({
        status: 'success',
        data: {
          user: {
            id: user.id,
            email: user.email,
            roles: user.roles
          }
        }
      });
    } catch (error) {
      logger.error('Assign role error:', error);
      res.status(500).json({
        status: 'error',
        message: 'Error assigning role'
      });
    }
  },

  // Password reset request
  forgotPassword: async (req, res) => {
    try {
      const { email } = req.body;
      
      const user = await User.findOne({ where: { email } });
      if (!user) {
        return res.status(404).json({
          status: 'error',
          message: 'User not found'
        });
      }

      const resetToken = jwt.sign(
        { id: user.id },
        process.env.JWT_RESET_SECRET,
        { expiresIn: '1h' }
      );

      // Store reset token in Redis
      await redisClient.set(
        `reset_token:${user.id}`,
        resetToken,
        'EX',
        60 * 60 // 1 hour
      );

      // TODO: Send email with reset link
      // For now, just return the token
      res.json({
        status: 'success',
        message: 'Password reset email sent',
        data: { resetToken } // Remove in production
      });
    } catch (error) {
      logger.error('Forgot password error:', error);
      res.status(500).json({
        status: 'error',
        message: 'Error processing password reset'
      });
    }
  },

  // Reset password
  resetPassword: async (req, res) => {
    try {
      const { token } = req.params;
      const { password } = req.body;

      const decoded = jwt.verify(token, process.env.JWT_RESET_SECRET);
      const storedToken = await redisClient.get(`reset_token:${decoded.id}`);

      if (!storedToken || storedToken !== token) {
        return res.status(401).json({
          status: 'error',
          message: 'Invalid or expired reset token'
        });
      }

      const user = await User.findByPk(decoded.id);
      if (!user) {
        return res.status(404).json({
          status: 'error',
          message: 'User not found'
        });
      }

      // Hash new password
      const salt = await bcrypt.genSalt(10);
      const hashedPassword = await bcrypt.hash(password, salt);

      // Update password
      user.password = hashedPassword;
      await user.save();

      // Remove reset token
      await redisClient.del(`reset_token:${decoded.id}`);

      res.json({
        status: 'success',
        message: 'Password successfully reset'
      });
    } catch (error) {
      logger.error('Reset password error:', error);
      res.status(500).json({
        status: 'error',
        message: 'Error resetting password'
      });
    }
  }
};

module.exports = authController; 