import React from 'react';
import {Routes, Route, Navigate} from 'react-router-dom';
import RPlaceClone from './components/RPlaceClone';
import {GoogleOAuthProvider} from '@react-oauth/google';

const AppRoutes = () => {
    return (
        <Routes>
            <Route path="/health" element={<h3>ok</h3>}/>
            <Route path="/" element={<React.StrictMode>
                <GoogleOAuthProvider clientId={process.env.REACT_APP_GOOGLE_CLIENT_ID}>
                    <RPlaceClone/>
                </GoogleOAuthProvider>
            </React.StrictMode>}/>
            <Route path="*" element={<Navigate to="/not-found" replace/>}/>
        </Routes>
    );
};

export default AppRoutes;