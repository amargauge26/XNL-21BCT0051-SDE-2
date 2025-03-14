const { DataTypes } = require('sequelize');
const sequelize = require('../config/database');

const User = sequelize.define('User', {
  id: {
    type: DataTypes.UUID,
    defaultValue: DataTypes.UUIDV4,
    primaryKey: true
  },
  email: {
    type: DataTypes.STRING,
    allowNull: false,
    unique: true,
    validate: {
      isEmail: true
    }
  },
  password: {
    type: DataTypes.STRING,
    allowNull: false
  },
  name: {
    type: DataTypes.STRING,
    allowNull: false
  },
  roles: {
    type: DataTypes.ARRAY(DataTypes.STRING),
    defaultValue: ['user']
  },
  isEmailVerified: {
    type: DataTypes.BOOLEAN,
    defaultValue: false
  },
  lastLogin: {
    type: DataTypes.DATE
  },
  provider: {
    type: DataTypes.STRING,
    defaultValue: 'local'
  },
  providerId: {
    type: DataTypes.STRING
  },
  profilePicture: {
    type: DataTypes.STRING
  },
  preferences: {
    type: DataTypes.JSONB,
    defaultValue: {}
  },
  status: {
    type: DataTypes.ENUM('active', 'inactive', 'suspended'),
    defaultValue: 'active'
  }
}, {
  timestamps: true,
  paranoid: true, // Soft deletes
  indexes: [
    {
      unique: true,
      fields: ['email']
    },
    {
      fields: ['provider', 'providerId']
    }
  ]
});

// Instance methods
User.prototype.toJSON = function() {
  const values = { ...this.get() };
  delete values.password;
  return values;
};

// Class methods
User.findByProvider = function(provider, providerId) {
  return this.findOne({
    where: { provider, providerId }
  });
};

module.exports = User; 