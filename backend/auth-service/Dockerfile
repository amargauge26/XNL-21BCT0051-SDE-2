# Use Node.js LTS version
FROM node:18-alpine

# Set working directory
WORKDIR /usr/src/app

# Install dependencies first (for better caching)
COPY package*.json ./
RUN npm ci --only=production

# Copy source code
COPY . .

# Set environment variables
ENV NODE_ENV=production
ENV PORT=3001

# Expose port
EXPOSE 3001

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3001/health || exit 1

# Start application
CMD ["npm", "start"] 