-- Database initialization script for Docker
-- This script runs when the PostgreSQL container starts for the first time

-- Create the application database if it doesn't exist
SELECT 'CREATE DATABASE nutrition_platform'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'nutrition_platform')\gexec

-- Connect to the application database
\c nutrition_platform;

-- Create application user if it doesn't exist
DO
$do$
BEGIN
   IF NOT EXISTS (
      SELECT FROM pg_catalog.pg_roles
      WHERE  rolname = 'nutrition_user') THEN

      CREATE ROLE nutrition_user LOGIN PASSWORD 'nutrition_password';
   END IF;
END
$do$;

-- Grant privileges to the application user
GRANT ALL PRIVILEGES ON DATABASE nutrition_platform TO nutrition_user;
GRANT ALL ON SCHEMA public TO nutrition_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO nutrition_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO nutrition_user;

-- Set default privileges for future tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO nutrition_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO nutrition_user;