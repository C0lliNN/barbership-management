// frontend/server.js — placeholder until Item 005 (Next.js PWA)
// Zero npm dependencies; uses only Node's built-in http module.
// Item 005 will replace this file with a full Next.js app.
'use strict';

const http = require('http');

const PORT = parseInt(process.env.PORT || '3000', 10);
const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

const HTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Barbearia — Em breve</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
      display: flex;
      align-items: center;
      justify-content: center;
      min-height: 100vh;
      background: #f5f5f5;
      color: #1a1a1a;
    }
    .card {
      background: #fff;
      border-radius: 16px;
      padding: 2.5rem 3rem;
      text-align: center;
      box-shadow: 0 2px 16px rgba(0,0,0,.08);
      max-width: 400px;
      width: 90%;
    }
    .icon { font-size: 3rem; margin-bottom: 1rem; }
    h1 { font-size: 1.75rem; margin-bottom: 0.5rem; }
    .sub { color: #555; margin-bottom: 1.5rem; }
    .badge {
      display: inline-block;
      background: #f0f0f0;
      border-radius: 8px;
      padding: 0.4rem 0.8rem;
      font-size: 0.78rem;
      color: #777;
      font-family: monospace;
    }
  </style>
</head>
<body>
  <div class="card">
    <div class="icon">💈</div>
    <h1>Barbearia</h1>
    <p class="sub">Plataforma de gerenciamento — em breve.</p>
    <span class="badge">API: ${API_URL}</span>
  </div>
</body>
</html>`;

const server = http.createServer((req, res) => {
  if (req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ status: 'ok' }));
    return;
  }

  res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
  res.end(HTML);
});

server.listen(PORT, '0.0.0.0', () => {
  // JSON structured log line so it blends with the API's zap output in compose logs
  process.stdout.write(
    JSON.stringify({
      level: 'info',
      msg: 'frontend stub listening',
      port: PORT,
      api: API_URL,
      note: 'placeholder — replaced by Next.js in Item 005',
    }) + '\n'
  );
});

server.on('error', (err) => {
  process.stderr.write(JSON.stringify({ level: 'error', msg: err.message }) + '\n');
  process.exit(1);
});
