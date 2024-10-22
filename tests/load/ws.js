import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

const BASE_URL = 'wss://grid.guliguli.work';
// Replace with a valid JWT token
const TOKEN = "load_test"

export const options = {
  stages: [
    { duration: '30s', target: 100 },  // Ramp up to 10 users over 30 seconds
    { duration: '1m', target: 200 },
    { duration: '30s', target: 0 },
  ],
};

export default function () {
  const url = `${BASE_URL}/ws?token=${TOKEN}`;

  const res = ws.connect(url, {}, function (socket) {
    check(socket, { 'Connected successfully': (socket) => socket.readyState === 1 });

    socket.on('open', () => {
      console.log('WebSocket connection established');

      socket.on('close', () => console.log('WebSocket connection closed'));

      // Close the connection after 1 minute
      setTimeout(() => {
        socket.close();
      }, 60000);
    });
  });

  check(res, { 'status is 101': (r) => r && r.status === 101 });

  sleep(1);
}