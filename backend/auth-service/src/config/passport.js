const passport = require('passport');
const { Strategy: JwtStrategy, ExtractJwt } = require('passport-jwt');
const GoogleStrategy = require('passport-google-oauth20').Strategy;
const GitHubStrategy = require('passport-github2').Strategy;
const User = require('../models/user');

// JWT Strategy
const jwtOptions = {
  jwtFromRequest: ExtractJwt.fromAuthHeaderAsBearerToken(),
  secretOrKey: process.env.JWT_SECRET
};

passport.use(new JwtStrategy(jwtOptions, async (payload, done) => {
  try {
    const user = await User.findByPk(payload.id);
    if (user) {
      return done(null, user);
    }
    return done(null, false);
  } catch (error) {
    return done(error, false);
  }
}));

// Google OAuth Strategy
passport.use(new GoogleStrategy({
  clientID: process.env.GOOGLE_CLIENT_ID,
  clientSecret: process.env.GOOGLE_CLIENT_SECRET,
  callbackURL: `${process.env.API_URL}/auth/google/callback`
}, async (accessToken, refreshToken, profile, done) => {
  try {
    // Find or create user
    let user = await User.findByProvider('google', profile.id);
    
    if (!user) {
      user = await User.create({
        email: profile.emails[0].value,
        name: profile.displayName,
        provider: 'google',
        providerId: profile.id,
        profilePicture: profile.photos[0]?.value,
        isEmailVerified: true,
        password: Math.random().toString(36).slice(-8) // Random password for OAuth users
      });
    }

    return done(null, user);
  } catch (error) {
    return done(error, false);
  }
}));

// GitHub OAuth Strategy
passport.use(new GitHubStrategy({
  clientID: process.env.GITHUB_CLIENT_ID,
  clientSecret: process.env.GITHUB_CLIENT_SECRET,
  callbackURL: `${process.env.API_URL}/auth/github/callback`,
  scope: ['user:email']
}, async (accessToken, refreshToken, profile, done) => {
  try {
    // Find or create user
    let user = await User.findByProvider('github', profile.id);
    
    if (!user) {
      user = await User.create({
        email: profile.emails[0].value,
        name: profile.displayName || profile.username,
        provider: 'github',
        providerId: profile.id,
        profilePicture: profile.photos[0]?.value,
        isEmailVerified: true,
        password: Math.random().toString(36).slice(-8) // Random password for OAuth users
      });
    }

    return done(null, user);
  } catch (error) {
    return done(error, false);
  }
}));

// Serialize user for the session
passport.serializeUser((user, done) => {
  done(null, user.id);
});

// Deserialize user from the session
passport.deserializeUser(async (id, done) => {
  try {
    const user = await User.findByPk(id);
    done(null, user);
  } catch (error) {
    done(error, null);
  }
}); 