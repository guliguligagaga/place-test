import React, {useCallback, useEffect, useRef, useState} from 'react';
import {Alert, Button} from 'react-bootstrap';
import {GoogleLogin, googleLogout} from '@react-oauth/google';
import useGrid from '../hooks/useGrid';
import PixelGrid from './PixelGrid';
import ColorPicker from './ColorPicker';

const GRID_SIZE = 100;
const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8081';
const INACTIVITY_TIMEOUT = 5 * 60 * 1000; // 5 minutes

const COLORS = [
    '#FFFFFF', '#E4E4E4', '#888888', '#222222',
    '#FFA7D1', '#E50000', '#E59500', '#A06A42',
    '#E5D900', '#94E044', '#02BE01', '#00D3DD',
    '#0083C7', '#0000EA', '#CF6EE4', '#820080'
];

const RPlaceClone = () => {
    const [grid, setGrid, updateGrid] = useGrid(GRID_SIZE * GRID_SIZE);
    const [selectedColor, setSelectedColor] = useState(0);
    const [error, setError] = useState(null);
    const [user, setUser] = useState(null);
    const [token, setToken] = useState(null);
    const [quadrants, setQuadrants] = useState([]);
    const [visibleQuadrants, setVisibleQuadrants] = useState(new Set());
    const wsRef = useRef(null);
    const lastActivityRef = useRef(Date.now());

    useEffect(() => {
        const storedUser = localStorage.getItem('user');
        const storedToken = localStorage.getItem('token');
        if (storedUser && storedToken) {
            setUser(JSON.parse(storedUser));
            setToken(storedToken);
        }
    }, []);


    const handleGoogleSignIn = useCallback(async (tokenResponse) => {
        try {
            const res = await fetch(`${API_BASE_URL}/api/auth/google`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({token: tokenResponse.credential}),
                credentials: 'include',
            });

            if (res.ok) {
                const data = await res.json();
                setUser(data.user);
                setToken(data.token);
                localStorage.setItem('user', JSON.stringify(data.user));
                localStorage.setItem('token', data.token);
                await fetchGrid();
                connectWebSocket();
            } else {
                throw new Error('Failed to authenticate with the server');
            }
        } catch (err) {
            setError('Authentication failed. Please try again.');
        }
    }, []);


    const fetchGrid = useCallback(async () => {
        if (!token) return;

        try {
            const response = await fetch(`${API_BASE_URL}/api/grid`, {
                headers: {
                    'Authorization': `Bearer ${token}`
                },
            });
            if (!response.ok) {
                throw new Error(`Failed to fetch grid: ${response.status} ${response.statusText}`);
            }
            const data = await response.arrayBuffer();
            const compressedGrid = new Uint8Array(data);
            console.log('Fetched compressed grid data:', compressedGrid);

            if (compressedGrid.length !== GRID_SIZE * GRID_SIZE / 2) {
                console.error(`Invalid grid size: expected ${GRID_SIZE * GRID_SIZE / 2}, got ${compressedGrid.length}`);
                return;
            }

            // Unpack the 4-bit format into 8-bit format
            const unpackedGrid = new Uint8Array(GRID_SIZE * GRID_SIZE);
            for (let i = 0; i < compressedGrid.length; i++) {
                const byte = compressedGrid[i];
                const x1 = i * 2;
                const x2 = x1 + 1;
                const y = Math.floor(i / (GRID_SIZE / 2));
                const color2 = byte & 0x0F;
                const color1 = (byte & 0xF0) >> 4;
                console.log(`Unpacking grid data: byte=${byte}, x1=${x1}, y=${y}, color1=${color1}, x2=${x2}, y=${y}, color2=${color2}`);
                unpackedGrid[y * GRID_SIZE + x1] = color1;
                unpackedGrid[y * GRID_SIZE + x2] = color2;
            }

            console.log('Unpacked grid data:', unpackedGrid);
            setGrid(unpackedGrid);
        } catch (err) {
            console.error('Error fetching grid:', err);
            setError('Failed to fetch grid: ' + err.message);
        }
    }, [token, setGrid]);
    const subscribeToQuadrant = useCallback((quadrantId) => {
        if (!wsRef.current) return;
        wsRef.current.send(JSON.stringify({
            type: 'Subscribe',
            payload: { quadrant_id: quadrantId }
        }));
        console.log(`Subscribed to quadrant ${quadrantId}`);
    }, []);

    const unsubscribeFromQuadrant = useCallback((quadrantId) => {
        if (!wsRef.current) return;
        wsRef.current.send(JSON.stringify({
            type: 'Unsubscribe',
            payload: { quadrant_id: quadrantId }
        }));
        console.log(`Unsubscribed from quadrant ${quadrantId}`);
    }, []);

    const handleVisibilityChange = useCallback((newVisibleQuadrants) => {
        setVisibleQuadrants((prevVisible) => {
            const toSubscribe = newVisibleQuadrants.filter(id => !prevVisible.has(id));
            const toUnsubscribe = Array.from(prevVisible).filter(id => !newVisibleQuadrants.has(id));

            toSubscribe.forEach(quadrantId => subscribeToQuadrant(quadrantId));
            toUnsubscribe.forEach(quadrantId => unsubscribeFromQuadrant(quadrantId));

            return new Set(newVisibleQuadrants);
        });
    }, [subscribeToQuadrant, unsubscribeFromQuadrant]);

    const connectWebSocket = useCallback(() => {
        if (!token) return;

        const wsUrl = `ws:${API_BASE_URL.replace(/^https?:/, '')}/ws`;
        wsRef.current = new WebSocket(wsUrl);

        wsRef.current.onopen = () => {
            console.log('WebSocket connection established');
            // Resubscribe to visible quadrants on reconnection
            visibleQuadrants.forEach(quadrantId => subscribeToQuadrant(quadrantId));
        };

        wsRef.current.onmessage = (event) => {
            const data = JSON.parse(event.data);
            if (data.type === 'configuration') {
                console.log("Received configuration", data);
                setQuadrants(data.quadrants);
                data.quadrants.forEach(quadrant => subscribeToQuadrant(quadrant.id))
            } else if (data.type === 'update') {
                console.log("Received update", data);
                updateGrid(data.x, data.y, data.color);
            }
        };

        wsRef.current.onerror = (error) => {
            console.error('WebSocket error:', error);
            setError('WebSocket connection error');
        };

        wsRef.current.onclose = (event) => {
            console.log('WebSocket connection closed:', event.code, event.reason);
            // Attempt to reconnect after a delay
            setTimeout(connectWebSocket, 5000);
        };

        return () => {
            if (wsRef.current) {
                wsRef.current.close();
            }
        };
    }, [token, updateGrid, visibleQuadrants, subscribeToQuadrant]);

    useEffect(() => {
        if (token) {
            fetchGrid();
            return connectWebSocket();
        }
    }, [token, fetchGrid, connectWebSocket]);

    useEffect(() => {
        const checkActivity = () => {
            const now = Date.now();
            if (now - lastActivityRef.current > INACTIVITY_TIMEOUT) {
                if (wsRef.current) {
                    wsRef.current.close();
                }
            }
        };

        const intervalId = setInterval(checkActivity, 600000); // Check every minute

        return () => clearInterval(intervalId);
    }, []);

    useEffect(() => {
        const handleActivity = () => {
            lastActivityRef.current = Date.now();
            if (wsRef.current && wsRef.current.readyState === WebSocket.CLOSED) {
                connectWebSocket();
            }
        };

        window.addEventListener('mousemove', handleActivity);
        window.addEventListener('keydown', handleActivity);

        return () => {
            window.removeEventListener('mousemove', handleActivity);
            window.removeEventListener('keydown', handleActivity);
        };
    }, [connectWebSocket]);

    const handlePixelUpdate = useCallback(async (x, y) => {
        if (!token) {
            setError('Please sign in to update pixels');
            return;
        }

        const response = await fetch(`${API_BASE_URL}/api/draw`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`
            },
            body: JSON.stringify({x, y, color: selectedColor}),
        });

        if (!response.ok) {
            setError('Failed to update pixel');
            return;
        }

        // Optimistically update the grid
        updateGrid(x, y, selectedColor);
    }, [token, selectedColor, updateGrid]);
    const handleSignOut = useCallback(() => {
        googleLogout();
        setUser(null);
        setToken(null);
        setGrid(new Uint8Array(GRID_SIZE * GRID_SIZE));
        if (wsRef.current) wsRef.current.close();
        localStorage.removeItem('user');
        localStorage.removeItem('token');
    }, [setGrid]);

    return (
        <div className="container d-flex flex-column align-items-center justify-content-center min-vh-100">
            <h1 className="mb-4">r/place Clone</h1>
            {user ? (
                <>
                    <p className="mb-4">Welcome, {user.name}!</p>
                    <Button onClick={handleSignOut} className="mb-4">Sign Out</Button>
                    <ColorPicker selectedColor={selectedColor} onColorSelect={setSelectedColor} colors={COLORS}/>
                    <PixelGrid
                        grid={grid}
                        onPixelClick={handlePixelUpdate}
                        size={GRID_SIZE}
                        colors={COLORS}
                        quadrants={quadrants}
                        onVisibilityChange={handleVisibilityChange}
                    />
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