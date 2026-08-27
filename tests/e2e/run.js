const { spawn } = require('child_process');
const fs = require('fs');
const net = require('net');
const os = require('os');
const path = require('path');

const fixtureBinary = path.join(
  os.tmpdir(),
  `ts3news-abyss-e2e-${process.pid}${process.platform === 'win32' ? '.exe' : ''}`,
);

function command(name, args, options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(name, args, {
      cwd: path.resolve(__dirname, '..', '..'),
      stdio: 'inherit',
      windowsHide: true,
      ...options,
    });
    child.once('error', reject);
    child.once('exit', (code, signal) => {
      if (code === 0) resolve();
      else reject(new Error(`${name} exited with ${code ?? signal}`));
    });
  });
}

function reserveFixturePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.unref();
    server.once('error', reject);
    server.listen(0, '127.0.0.1', () => {
      const { port } = server.address();
      server.close(error => error ? reject(error) : resolve(port));
    });
  });
}

async function waitForFixture(child, port) {
  const deadline = Date.now() + 120_000;
  while (Date.now() < deadline) {
    if (child.exitCode !== null) throw new Error(`Abyss fixture exited with ${child.exitCode}`);
    try {
      const response = await fetch(`http://127.0.0.1:${port}/healthz`, {
        signal: AbortSignal.timeout(1_000),
      });
      if (response.status === 204) return;
    } catch (_) {
      // Compilation/startup races are expected until the bounded deadline.
    }
    await new Promise(resolve => setTimeout(resolve, 200));
  }
  throw new Error('Abyss E2E fixture did not become ready within 120 seconds');
}

async function stopFixture(child) {
  if (!child || child.exitCode !== null) return;
  await new Promise(resolve => {
    const deadline = setTimeout(resolve, 5_000);
    deadline.unref();
    child.once('exit', () => {
      clearTimeout(deadline);
      resolve();
    });
    child.kill();
  });
}

async function main() {
  let fixture;
  try {
    const port = await reserveFixturePort();
    const fixtureEnv = { ...process.env, ABYSS_E2E_PORT: String(port) };
    await command('go', ['test', '-c', '-tags=e2e', '-o', fixtureBinary, './internal/bot']);
    fixture = spawn(fixtureBinary, [
      '-test.run=^TestAbyssE2EServer$',
      '-test.v',
      '-test.timeout=30m',
    ], {
      cwd: path.resolve(__dirname, '..', '..'),
      stdio: 'inherit',
      windowsHide: true,
      env: fixtureEnv,
    });
    fixture.on('error', error => console.error('Abyss fixture error:', error));
    await waitForFixture(fixture, port);
    await command(process.execPath, [require.resolve('@playwright/test/cli'), 'test', ...process.argv.slice(2)], {
      env: { ...fixtureEnv, ABYSS_E2E_EXTERNAL_SERVER: '1' },
    });
  } finally {
    await stopFixture(fixture);
    fs.rmSync(fixtureBinary, { force: true });
  }
}

main().catch(error => {
  console.error(error);
  process.exitCode = 1;
});
