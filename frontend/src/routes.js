import React, { useContext } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import RPlaceClone from './components/RPlaceClone';
import { GoogleOAuthProvider } from '@react-oauth/google';
import { ConfigContext } from './ConfigProvider';


const AppRoutes = ({ authEnabled }) => {
    return (
        <Routes>
            <Route path="/health" element={<h3>ok</h3>} />
            <Route path="/" element={
                <React.StrictMode>
                    {authEnabled ? (
                        <GoogleOAuthProvider clientId={config.googleClientId}>
                            <RPlaceClone />
                        </GoogleOAuthProvider>
                    ) : (
                        <RPlaceClone authEnabled={authEnabled} />
                    )}
                </React.StrictMode>
            } />
            <Route path="*" element={<Navigate to="/not-found" replace />} />
        </Routes>
    );
};

export default AppRoutes;