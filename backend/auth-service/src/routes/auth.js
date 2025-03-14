const express = require('express');
const passport = require('passport');
const jwt = require('jsonwebtoken');
const { body, validationResult } = require('express-validator');
const router = express.Router();

// Import auth controller (we'll create this next)
const authController = require('../controllers/authController');

// Import passport configuration
require('../config/passport');

// Validation middleware
const validateLoginInput = [
  body('email').isEmail().normalizeEmail(),
  body('password').isLength({ min: 6 })
];

// Routes for local authentication
router.post('/register', [
  body('email').isEmail().normalizeEmail(),
  body('password').isLength({ min: 6 }),
  body('name').trim().notEmpty()
], authController.register);

router.post('/login', validateLoginInput, authController.login);

// OAuth routes
router.get('/google',
  passport.authenticate('google', { 
    scope: ['profile', 'email']
  })
);

router.get('/google/callback',
  passport.authenticate('google', { 
    failureRedirect: '/login',
    session: false 
  }),
  authController.oauthCallback
);

router.get('/github',
  passport.authenticate('github', { 
    scope: ['user:email']
  })
);

router.get('/github/callback',
  passport.authenticate('github', { 
    failureRedirect: '/login',
    session: false
  }),
  authController.oauthCallback
);

// Protected route example
router.get('/profile',
  passport.authenticate('jwt', { session: false }),
  authController.getProfile
);

// Logout route
router.post('/logout', authController.logout);

// Password reset routes
router.post('/forgot-password',
  body('email').isEmail().normalizeEmail(),
  authController.forgotPassword
);

router.post('/reset-password/:token',
  body('password').isLength({ min: 6 }),
  authController.resetPassword
);

// Refresh token route
router.post('/refresh-token', authController.refreshToken);

// Role management routes
router.post('/roles',
  passport.authenticate('jwt', { session: false }),
  authController.checkRole(['admin']),
  authController.assignRole
);

module.exports = router; 