#!/usr/bin/env node
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { z } from "zod";
import axios from 'axios';

const BASE_URL = process.env.COOLIFY_API_BASE_URL || 'https://api.doctorhealthy1.com/api/v1';
const API_TOKEN = process.env.COOLIFY_API_TOKEN;

if (!API_TOKEN) {
  throw new Error('COOLIFY_API_TOKEN environment variable is required');
}

// Create axios instance for Coolify API
const coolifyApi = axios.create({
  baseURL: BASE_URL,
  headers: {
    'Authorization': `Bearer ${API_TOKEN}`,
    'Content-Type': 'application/json',
    'Accept': 'application/json'
  },
});

// Create an MCP server
const server = new McpServer({
  name: "coolify-server",
  version: "0.1.0"
});

// Add tool to list projects
server.tool(
  "list_projects",
  {
    limit: z.number().min(1).max(100).optional().describe("Number of projects to return (default: 50)"),
    page: z.number().min(1).optional().describe("Page number (default: 1)"),
  },
  async ({ limit = 50, page = 1 }) => {
    try {
      const response = await coolifyApi.get('/projects', {
        params: { limit, page }
      });

      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(response.data, null, 2),
          },
        ],
      };
    } catch (error) {
      if (axios.isAxiosError(error)) {
        return {
          content: [
            {
              type: "text",
              text: `Coolify API error: ${
                error.response?.data?.message || error.response?.data || error.message
              }`,
            },
          ],
          isError: true,
        };
      }
      throw error;
    }
  }
);

// Add tool to get project details
server.tool(
  "get_project",
  {
    project_id: z.string().describe("Project ID or UUID"),
  },
  async ({ project_id }) => {
    try {
      const response = await coolifyApi.get(`/projects/${project_id}`);

      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(response.data, null, 2),
          },
        ],
      };
    } catch (error) {
      if (axios.isAxiosError(error)) {
        return {
          content: [
            {
              type: "text",
              text: `Coolify API error: ${
                error.response?.data?.message || error.response?.data || error.message
              }`,
            },
          ],
          isError: true,
        };
      }
      throw error;
    }
  }
);

// Add tool to deploy project
server.tool(
  "deploy_project",
  {
    project_id: z.string().describe("Project ID or UUID"),
    environment_name: z.string().optional().describe("Environment name (default: production)"),
    branch: z.string().optional().describe("Branch to deploy (default: main)"),
  },
  async ({ project_id, environment_name = "production", branch = "main" }) => {
    try {
      const response = await coolifyApi.post(`/projects/${project_id}/deploy`, {
        environment_name,
        branch
      });

      return {
        content: [
          {
            type: "text",
            text: `Deployment initiated successfully: ${JSON.stringify(response.data, null, 2)}`,
          },
        ],
      };
    } catch (error) {
      if (axios.isAxiosError(error)) {
        return {
          content: [
            {
              type: "text",
              text: `Coolify API error: ${
                error.response?.data?.message || error.response?.data || error.message
              }`,
            },
          ],
          isError: true,
        };
      }
      throw error;
    }
  }
);

// Add tool to list deployments
server.tool(
  "list_deployments",
  {
    project_id: z.string().optional().describe("Filter by project ID"),
    limit: z.number().min(1).max(100).optional().describe("Number of deployments to return (default: 50)"),
    page: z.number().min(1).optional().describe("Page number (default: 1)"),
  },
  async ({ project_id, limit = 50, page = 1 }) => {
    try {
      const params: any = { limit, page };
      if (project_id) params.project_id = project_id;

      const response = await coolifyApi.get('/deployments', { params });

      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(response.data, null, 2),
          },
        ],
      };
    } catch (error) {
      if (axios.isAxiosError(error)) {
        return {
          content: [
            {
              type: "text",
              text: `Coolify API error: ${
                error.response?.data?.message || error.response?.data || error.message
              }`,
            },
          ],
          isError: true,
        };
      }
      throw error;
    }
  }
);

// Add tool to get deployment details
server.tool(
  "get_deployment",
  {
    deployment_id: z.string().describe("Deployment ID or UUID"),
  },
  async ({ deployment_id }) => {
    try {
      const response = await coolifyApi.get(`/deployments/${deployment_id}`);

      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(response.data, null, 2),
          },
        ],
      };
    } catch (error) {
      if (axios.isAxiosError(error)) {
        return {
          content: [
            {
              type: "text",
              text: `Coolify API error: ${
                error.response?.data?.message || error.response?.data || error.message
              }`,
            },
          ],
          isError: true,
        };
      }
      throw error;
    }
  }
);

// Add tool to list servers
server.tool(
  "list_servers",
  {
    limit: z.number().min(1).max(100).optional().describe("Number of servers to return (default: 50)"),
    page: z.number().min(1).optional().describe("Page number (default: 1)"),
  },
  async ({ limit = 50, page = 1 }) => {
    try {
      const response = await coolifyApi.get('/servers', {
        params: { limit, page }
      });

      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(response.data, null, 2),
          },
        ],
      };
    } catch (error) {
      if (axios.isAxiosError(error)) {
        return {
          content: [
            {
              type: "text",
              text: `Coolify API error: ${
                error.response?.data?.message || error.response?.data || error.message
              }`,
            },
          ],
          isError: true,
        };
      }
      throw error;
    }
  }
);

// Add tool to get server details
server.tool(
  "get_server",
  {
    server_id: z.string().describe("Server ID or UUID"),
  },
  async ({ server_id }) => {
    try {
      const response = await coolifyApi.get(`/servers/${server_id}`);

      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(response.data, null, 2),
          },
        ],
      };
    } catch (error) {
      if (axios.isAxiosError(error)) {
        return {
          content: [
            {
              type: "text",
              text: `Coolify API error: ${
                error.response?.data?.message || error.response?.data || error.message
              }`,
            },
          ],
          isError: true,
        };
      }
      throw error;
    }
  }
);

// Start receiving messages on stdin and sending messages on stdout
const transport = new StdioServerTransport();
await server.connect(transport);
console.error('Coolify MCP server running on stdio');