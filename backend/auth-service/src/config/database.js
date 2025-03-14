const { Sequelize } = require('sequelize');

const sequelize = new Sequelize(
  process.env.DB_NAME || 'financeapp',
  process.env.DB_USER || 'root',
  process.env.DB_PASSWORD || '',
  {
    host: process.env.DB_HOST || 'localhost',
    port: process.env.DB_PORT || 26257, // CockroachDB default port
    dialect: 'postgres', // CockroachDB uses PostgreSQL wire protocol
    dialectOptions: {
      ssl: process.env.NODE_ENV === 'production' ? {
        rejectUnauthorized: true,
        ca: process.env.DB_SSL_CA,
      } : false,
    },
    pool: {
      max: 20,
      min: 0,
      acquire: 60000,
      idle: 10000
    },
    logging: process.env.NODE_ENV === 'development' ? console.log : false,
    define: {
      timestamps: true,
      underscored: true
    },
    retry: {
      max: 10,
      match: [
        /40001/, // Retry on serialization failures
        /ECONNREFUSED/,
        /ETIMEDOUT/,
        /ECONNRESET/,
        /ESOCKETTIMEDOUT/
      ]
    }
  }
);

// Test database connection
const testConnection = async () => {
  try {
    await sequelize.authenticate();
    console.log('Database connection has been established successfully.');
  } catch (error) {
    console.error('Unable to connect to the database:', error);
    process.exit(1);
  }
};

// Sync database in development
if (process.env.NODE_ENV === 'development') {
  sequelize.sync({ alter: true }).then(() => {
    console.log('Database synced in development mode');
  });
}

testConnection();

module.exports = sequelize; 