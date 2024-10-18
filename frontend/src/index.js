import React from 'react';
import ReactDOM from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import AppRoutes from './routes';
import { ConfigProvider, ConfigContext } from './ConfigProvider';
import 'bootstrap/dist/css/bootstrap.min.css';

const rootElement = document.getElementById('root');
const root = ReactDOM.createRoot(rootElement);

const App = () => {
    return (
        <ConfigProvider>
            <ConfigContext.Consumer>
                {(config) => (
                    <BrowserRouter>
                        <AppRoutes authEnabled={config.authEnabled} />
                    </BrowserRouter>
                )}
            </ConfigContext.Consumer>
        </ConfigProvider>
    );
};

root.render(<App />);