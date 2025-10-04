const express = require('express');
const app = express();
const port = 3001;

app.get('/', (req, res) => {
  const currentTime = new Date().toLocaleString();
  res.send(`L'heure actuelle est : ${currentTime}`);
});

app.get('/stats', (req, res) => {
  res.json({ status: 'ok', uptime: process.uptime() });
});

app.listen(port, () => {
  console.log(`Backend Node.js démarré sur le port ${port}`);
});