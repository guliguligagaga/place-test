import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

const BASE_URL = 'ws://guliguli.work'; // Replace with your WebSocket server URL
const TOKEN = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3Mjc2MDIzNjIsImlzcyI6Imdvb2dsZSIsInN1YiI6IjEwNzQxMzE1NTkwNDk2OTIzNzkzMyJ9.vzyIMRemYBhcLXtHZ_Eic8VgDuzsZnhJ2d947OnCAOc'; // Replace with a valid JWT token

export const options = {
  stages: [
    { duration: '30s', target: 1000 },  // Ramp up to 10 users over 30 seconds
    { duration: '1m', target: 1000 },   // Stay at 10 users for 1 minute
    { duration: '30s', target: 0 },   // Ramp down to 0 users over 30 seconds
  ],
};

export default function () {
  const url = `${BASE_URL}/ws?token=${TOKEN}`;

  const res = ws.connect(url, {}, function (socket) {
    check(socket, { 'Connected successfully': (socket) => socket.readyState === 1 });

    socket.on('open', () => {
      console.log('WebSocket connection established');

      // Subscribe to a random quadrant
      const quadrantId = randomIntBetween(0, 3);
      socket.send(JSON.stringify({ type: 'Subscribe', payload: { quadrant_id: quadrantId } }));

      // Simulate pixel updates
      const intervalId = setInterval(() => {
        const x = randomIntBetween(0, 999);
        const y = randomIntBetween(0, 999);
        const color = randomIntBetween(0, 15);
        socket.send(JSON.stringify({ type: 'PixelUpdate', payload: { x, y, color } }));
      }, 5000); // Send a pixel update every 5 seconds

      socket.on('message', (msg) => {
        const data = JSON.parse(msg);
        check(data, {
          'Received pixel update': (data) => data.type === 'PixelUpdate',
        });
      });

      socket.on('close', () => console.log('WebSocket connection closed'));

      // Close the connection after 1 minute
      setTimeout(() => {
        clearInterval(intervalId);
        socket.close();
      }, 60000);
    });
  });

  check(res, { 'status is 101': (r) => r && r.status === 101 });

  sleep(1);
}