module.exports = {
  // Database configuration
  database: {
    type: 'sqlite',
    sqlite: {
      database: '/home/node/.n8n/database.sqlite',
      enableWAL: true,
    },
  },

  // Security settings
  security: {
    // Basic auth for n8n UI
    basicAuth: {
      active: true,
      user: process.env.N8N_USER || 'admin',
      password: process.env.N8N_PASSWORD || 'password',
    },
    
    // CORS settings
    cors: {
      origin: process.env.N8N_HOST || 'https://localhost',
      credentials: true,
    },
    
    // SSL/TLS settings
    ssl: {
      cert: '/etc/ssl/certs/n8n.crt',
      key: '/etc/ssl/private/n8n.key',
    },
  },

  // Host and port settings
  host: process.env.N8N_HOST || 'localhost',
  port: process.env.N8N_PORT || 5678,
  protocol: process.env.N8N_PROTOCOL || 'https',
  
  // SSL settings
  ssl: {
    cert: '/etc/ssl/certs/n8n.crt',
    key: '/etc/ssl/private/n8n.key',
  },

  // External URL (for webhooks and callbacks)
  url: process.env.N8N_PROTOCOL + '://' + (process.env.N8N_HOST || 'localhost') + ':' + (process.env.N8N_PORT || 5678),

  // Execution settings
  execution: {
    mode: 'main',
    timeout: 3600000, // 1 hour in milliseconds
    maxTimeout: 21600000, // 6 hours in milliseconds
  },

  // Queue settings
  queue: {
    mode: 'main',
    redis: {
      host: process.env.REDIS_HOST || 'redis',
      port: process.env.REDIS_PORT || 6379,
      db: process.env.REDIS_DB || 0,
    },
  },

  // Binary data settings
  binaryData: {
    mode: 'default',
    uploadFolder: '/home/node/.n8n/binaryData',
    temporaryFolder: '/tmp/n8n',
  },

  // Logging settings
  logging: {
    level: 'info',
    format: 'json',
    file: '/home/node/.n8n/logs/n8n.log',
  },

  // External hooks
  externalHooks: {
    'workflow.afterExecute': [
      'https://your-domain.com/webhooks/n8n-workflow-completed'
    ],
  },

  // Custom settings
  custom: {
    // Disable telemetry
    telemetry: false,
    
    // Disable frontend tracking
    frontendTracking: false,
    
    // Enable webhook testing
    webhookTesting: true,
  },

  // OAuth2 settings (if needed)
  oauth2: {
    google: {
      clientId: process.env.GOOGLE_CLIENT_ID,
      clientSecret: process.env.GOOGLE_CLIENT_SECRET,
      scopes: ['https://www.googleapis.com/auth/calendar'],
    },
  },

  // Email settings (if needed)
  email: {
    smtp: {
      host: process.env.SMTP_HOST,
      port: process.env.SMTP_PORT || 587,
      secure: process.env.SMTP_SECURE === 'true',
      auth: {
        user: process.env.SMTP_USER,
        pass: process.env.SMTP_PASS,
      },
    },
  },
};