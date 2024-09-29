import React, { useContext } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import RPlaceClone from './components/RPlaceClone';
import { GoogleOAuthProvider } from '@react-oauth/google';
import { ConfigContext } from './ConfigProvider';

const AppRoutes = () => {
    const config = useContext(ConfigContext);

    return (
        <Routes>
            <Route path="/health" element={<h3>ok</h3>} />
            <Route path="/" element={
                <React.StrictMode>
                    <GoogleOAuthProvider clientId="4569410916-mf7l68sh509mrlpu3ih7op7b6dgg4tqh.apps.googleusercontent.com">
                        <RPlaceClone />
                    </GoogleOAuthProvider>
                </React.StrictMode>
            } />
            <Route path="*" element={<Navigate to="/not-found" replace />} />
        </Routes>
    );
};

export default AppRoutes;