import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import AppRoutes from './routes';
import { ConfigProvider } from './ConfigProvider';
import 'bootstrap/dist/css/bootstrap.min.css';

const rootElement = document.getElementById('root');
const root = ReactDOM.createRoot(rootElement);

root.render(
  <ConfigProvider>
    <BrowserRouter>
      <AppRoutes />
    </BrowserRouter>
  </ConfigProvider>
);