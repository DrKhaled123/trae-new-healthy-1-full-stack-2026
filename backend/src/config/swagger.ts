import swaggerJsdoc from 'swagger-jsdoc';
import swaggerUi from 'swagger-ui-express';
import { Express } from 'express';
import { config } from './env';

// Swagger configuration options
const options = {
  definition: {
    openapi: '3.0.0',
    info: {
      title: 'NutriTrack API',
      version: config.API_VERSION || '1.0.0',
      description: 'Secure nutrition tracking platform API documentation',
      contact: {
        name: 'NutriTrack Team',
        email: 'support@nutritrack.com',
        url: 'https://nutritrack.com'
      },
      license: {
        name: 'MIT',
        url: 'https://opensource.org/licenses/MIT'
      }
    },
    servers: [
      {
        url: config.NODE_ENV === 'production' 
          ? 'https://api.nutritrack.com' 
          : `http://localhost:${config.PORT}`,
        description: config.NODE_ENV === 'production' 
          ? 'Production server' 
          : 'Development server'
      }
    ],
    components: {
      securitySchemes: {
        bearerAuth: {
          type: 'http',
          scheme: 'bearer',
          bearerFormat: 'JWT',
          description: 'JWT access token'
        }
      },
      schemas: {
        Error: {
          type: 'object',
          properties: {
            success: {
              type: 'boolean',
              example: false
            },
            message: {
              type: 'string',
              example: 'Error message'
            },
            code: {
              type: 'string',
              example: 'ERROR_CODE'
            },
            errors: {
              type: 'array',
              items: {
                type: 'object',
                properties: {
                  field: {
                    type: 'string',
                    example: 'email'
                  },
                  message: {
                    type: 'string',
                    example: 'Invalid email format'
                  }
                }
              }
            }
          }
        },
        Success: {
          type: 'object',
          properties: {
            success: {
              type: 'boolean',
              example: true
            },
            message: {
              type: 'string',
              example: 'Operation successful'
            },
            data: {
              type: 'object',
              description: 'Response data'
            }
          }
        },
        Pagination: {
          type: 'object',
          properties: {
            page: {
              type: 'integer',
              example: 1
            },
            limit: {
              type: 'integer',
              example: 10
            },
            total: {
              type: 'integer',
              example: 100
            },
            pages: {
              type: 'integer',
              example: 10
            }
          }
        }
      }
    },
    security: [
      {
        bearerAuth: []
      }
    ],
    tags: [
      {
        name: 'Health',
        description: 'Health check endpoints'
      },
      {
        name: 'Authentication',
        description: 'User authentication endpoints'
      },
      {
        name: 'Users',
        description: 'User management endpoints'
      },
      {
        name: 'Nutrition',
        description: 'Nutrition tracking endpoints'
      }
    ]
  },
  apis: [
    './src/routes/*.ts',
    './src/controllers/*.ts',
    './src/models/*.ts'
  ]
};

// Generate Swagger specification
const specs = swaggerJsdoc(options);

// Custom CSS for Swagger UI
const customCss = `
  .swagger-ui .topbar { display: none }
  .swagger-ui .info { margin: 20px 0 }
  .swagger-ui .scheme-container { margin: 20px 0 }
  .swagger-ui .opblock.opblock-post { border-color: #49cc90; }
  .swagger-ui .opblock.opblock-get { border-color: #61affe; }
  .swagger-ui .opblock.opblock-put { border-color: #fca130; }
  .swagger-ui .opblock.opblock-delete { border-color: #f93e3e; }
  .swagger-ui .opblock.opblock-patch { border-color: #50e3c2; }
`;

// Custom Swagger UI options
const swaggerUiOptions = {
  customCss,
  customSiteTitle: 'NutriTrack API Documentation',
  explorer: true,
  swaggerOptions: {
    persistAuthorization: true,
    displayRequestDuration: true,
    filter: true,
    showExtensions: true,
    showCommonExtensions: true,
    docExpansion: 'none',
    defaultModelsExpandDepth: 2,
    defaultModelExpandDepth: 2,
    tryItOutEnabled: true
  }
};

/**
 * Setup Swagger documentation
 */
export const setupSwagger = (app: Express): void => {
  // Swagger UI route
  app.use('/api/docs', swaggerUi.serve);
  app.get('/api/docs', swaggerUi.setup(specs, swaggerUiOptions));

  // JSON specification route
  app.get('/api/docs.json', (req, res) => {
    res.setHeader('Content-Type', 'application/json');
    res.send(specs);
  });

  // Log that Swagger has been set up
  console.log('📚 Swagger documentation available at /api/docs');
  console.log('📄 Swagger JSON specification available at /api/docs.json');
};

export { specs, swaggerUiOptions };
