#!/usr/bin/env node
import { spawn } from 'child_process';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

async function testCoolifyServer() {
  console.log('🚀 Starting Coolify MCP Server Test...\n');

  // Start the MCP server
  const serverProcess = spawn('node', ['dist/index.js'], {
    cwd: __dirname,
    stdio: ['pipe', 'pipe', 'inherit'],
    env: {
      ...process.env,
      COOLIFY_API_BASE_URL: 'https://api.doctorhealthy1.com/api/v1',
      COOLIFY_API_TOKEN: '6|uJSYhIJQIypx4UuxbQkaHkidEyiQshLR6U1QNxEQab344fda'
    }
  });

  // Handle server output
  if (serverProcess.stdout) {
    serverProcess.stdout.on('data', (data) => {
      const output = data.toString();
      console.log('📤 Server:', output.trim());

      // If server is ready, we can test it
      if (output.includes('Coolify MCP server running on stdio')) {
        console.log('✅ Server started successfully!');
        setTimeout(() => {
          console.log('🧪 Server is ready for testing via MCP tools');
          serverProcess.kill();
          process.exit(0);
        }, 2000);
      }
    });
  }

  serverProcess.stderr.on('data', (data) => {
    console.error('❌ Server Error:', data.toString().trim());
  });

  serverProcess.on('close', (code) => {
    if (code === 0) {
      console.log('✅ Server test completed successfully');
    } else {
      console.error(`❌ Server exited with code ${code}`);
    }
  });

  // Timeout after 10 seconds
  setTimeout(() => {
    console.log('⏰ Test timeout reached');
    serverProcess.kill();
    process.exit(1);
  }, 10000);
}

// Run the test
testCoolifyServer().catch(console.error);