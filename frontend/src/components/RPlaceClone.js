import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Alert, Button } from 'react-bootstrap';
import { GoogleLogin, googleLogout, useGoogleLogin } from '@react-oauth/google';
import useGrid from '../hooks/useGrid';
import PixelGrid from './PixelGrid';
import ColorPicker, {COLORS} from './ColorPicker';

const GRID_SIZE = 100;
const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://api.example.com';

const RPlaceClone = () => {
    const [grid, setGrid, updateGrid] = useGrid();
    const [selectedColor, setSelectedColor] = useState(COLORS[0]);
    const [error, setError] = useState(null);
    const [user, setUser] = useState(null);
    const [token, setToken] = useState(null);
    const wsRef = useRef(null);

    useEffect(() => {
        const loadGoogleScript = () => {
            const script = document.createElement('script');
            script.src = 'https://accounts.google.com/gsi/client';
            script.async = true;
            script.defer = true;
            document.body.appendChild(script);
            return script;
        };

        const script = loadGoogleScript();

        return () => {
            document.body.removeChild(script);
        };
    }, []);

    const handleGoogleSignIn = useCallback(async (tokenResponse) => {
        try {
            const res = await fetch(`${API_BASE_URL}/auth/google`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ token: tokenResponse }),
            });

            if (res.ok) {
                const data = await res.json();
                setUser(data.user);
                setToken(data.token);
                await fetchGrid();
                connectWebSocket();
            } else {
                throw new Error('Failed to authenticate with the server');
            }
        } catch (err) {
            setError('Authentication failed. Please try again.');
        }
    }, []);

    const login = useGoogleLogin({
        onSuccess: handleGoogleSignIn,
        onError: () => setError('Google Sign-In failed. Please try again.'),
    });

    const fetchGrid = useCallback(async () => {
        if (!token) return;

        try {
            const response = await fetch(`${API_BASE_URL}/api/grid`, {
                headers: {
                    'Authorization': `Bearer ${token}`
                }
            });
            if (!response.ok) {
                throw new Error('Failed to fetch grid');
            }
            const data = await response.json();
            setGrid(data);
        } catch (err) {
            setError('Failed to fetch grid');
        }
    }, [token, setGrid]);

    const connectWebSocket = useCallback(() => {
        if (!token) return;

        const wsUrl = `ws:${API_BASE_URL.replace(/^https?:/, '')}/ws`;
        wsRef.current = new WebSocket(wsUrl);
        wsRef.current.onopen = () => {
            wsRef.current?.send(JSON.stringify({ token }));
        };
        wsRef.current.onmessage = (event) => {
            const update = JSON.parse(event.data);
            updateGrid(update.index, update.color);
        };
        wsRef.current.onerror = () => setError('WebSocket connection error');

        return () => {
            if (wsRef.current) {
                wsRef.current.close();
            }
        };
    }, [token, updateGrid]);

    useEffect(() => {
        if (token) {
            fetchGrid();
            return connectWebSocket();
        }
    }, [token, fetchGrid, connectWebSocket]);

    const handlePixelUpdate = useCallback(async (index) => {
        if (!token) {
            setError('Please sign in to update pixels');
            return;
        }

        try {
            const response = await fetch(`${API_BASE_URL}/api/update`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify({ index, color: selectedColor }),
            });

            if (!response.ok) throw new Error('Failed to update pixel');

            // Optimistically update the grid
            updateGrid(index, selectedColor);
        } catch (err) {
            setError('Failed to update pixel');
        }
    }, [token, selectedColor, updateGrid]);

    const handleSignOut = useCallback(() => {
        googleLogout();
        setUser(null);
        setToken(null);
        setGrid(Array(GRID_SIZE * GRID_SIZE).fill('#FFFFFF'));
        if (wsRef.current) wsRef.current.close();
        window.google?.accounts.id.disableAutoSelect();
    }, [setGrid]);

    return (
        <div className="container d-flex flex-column align-items-center justify-content-center min-vh-100">
            <h1 className="mb-4">r/place Clone</h1>
            {user ? (
                <>
                    <p className="mb-4">Welcome, {user.name}!</p>
                    <Button onClick={handleSignOut} className="mb-4">Sign Out</Button>
                    <ColorPicker selectedColor={selectedColor} onColorSelect={setSelectedColor} />
                    <PixelGrid grid={grid} onPixelClick={handlePixelUpdate} />
                </>
            ) : (
                <GoogleLogin
                    onSuccess={handleGoogleSignIn}
                    onError={() => setError('Google Sign-In failed. Please try again.')}
                />
            )}
            {error && (
                <Alert variant="danger" className="mt-4">
                    {error}
                </Alert>
            )}
        </div>
    );
};

export default RPlaceClone;