import React, { useCallback, useEffect, useState, useMemo } from 'react';
import { Alert, Button } from 'react-bootstrap';
import { GoogleLogin, googleLogout } from '@react-oauth/google';
import useGrid from '../hooks/useGrid';
import PixelGrid from './PixelGrid';
import ColorPicker from './ColorPicker';
import { debounce } from 'lodash';

const GRID_SIZE = 100;
const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8081';
const INACTIVITY_TIMEOUT = 5 * 60 * 1000; // 5 minutes
const MAX_RECONNECT_ATTEMPTS = 5;

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
    const [token, setToken] = useState(() => localStorage.getItem('token'));
    const [quadrants, setQuadrants] = useState([]);
    const [visibleQuadrants, setVisibleQuadrants] = useState(new Set());
    const [isLoading, setIsLoading] = useState(false);
    const [wsError, setWsError] = useState(null);
    const [reconnectDelay, setReconnectDelay] = useState(1000);
    const [initialFetchDone, setInitialFetchDone] = useState(false);

    const wsRef = React.useRef(null);
    const lastActivityRef = React.useRef(Date.now());
    const reconnectAttemptsRef = React.useRef(0);
    const reconnectTimeoutRef = React.useRef(null);

    useEffect(() => {
        const storedUser = localStorage.getItem('user');
        const storedToken = localStorage.getItem('token');
        if (storedUser && storedToken) {
            setUser(JSON.parse(storedUser));
            setToken(storedToken);
        }
    }, []);

    const fetchGrid = useCallback(async () => {
        if (!token || initialFetchDone) return;
        console.log('Fetching grid...');

        try {
            setIsLoading(true);
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
                unpackedGrid[y * GRID_SIZE + x1] = color1;
                unpackedGrid[y * GRID_SIZE + x2] = color2;
            }
            console.log('Unpacked grid data:', unpackedGrid);
            setGrid(unpackedGrid);
            setInitialFetchDone(true);  // Set this to true after successful fetch
        } catch (err) {
            console.error('Error fetching grid:', err);
            setError('Failed to fetch grid: ' + err.message);
        } finally {
            console.log('Grid fetch completed');
            setIsLoading(false);
        }
    }, [token, setGrid, initialFetchDone]);

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

    const connectWebSocket = useCallback(() => {
        if (!token || wsRef.current) return; // Add this line to prevent multiple connections

        if (reconnectAttemptsRef.current >= MAX_RECONNECT_ATTEMPTS) {
            console.log('Max reconnection attempts reached. Stopping reconnection.');
            setWsError('Unable to establish WebSocket connection. Please refresh the page.');
            return;
        }

        const wsUrl = `ws:${API_BASE_URL.replace(/^https?:/, '')}/ws`;
        wsRef.current = new WebSocket(`${wsUrl}?token=${encodeURIComponent(token)}`);
        wsRef.current.withCredentials = false;

        wsRef.current.onopen = () => {
            console.log('WebSocket connection established');
            reconnectAttemptsRef.current = 0;
            setReconnectDelay(1000);
            setWsError(null);
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
            reconnectAttemptsRef.current++;
            const nextDelay = Math.min(30000, reconnectDelay * 2);
            console.log(`Reconnecting in ${nextDelay}ms (attempt ${reconnectAttemptsRef.current} of ${MAX_RECONNECT_ATTEMPTS})`);
            
            if (reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
                reconnectTimeoutRef.current = setTimeout(() => {
                    connectWebSocket();
                    setReconnectDelay(nextDelay);
                }, nextDelay);
            } else {
                setWsError('Max reconnection attempts reached. Please refresh the page.');
            }
        };

        wsRef.current.onclose = (event) => {
            console.log('WebSocket connection closed:', event.code, event.reason);
            reconnectAttemptsRef.current++;
            const nextDelay = Math.min(30000, reconnectDelay * 2);
            console.log(`Reconnecting in ${nextDelay}ms (attempt ${reconnectAttemptsRef.current} of ${MAX_RECONNECT_ATTEMPTS})`);
            
            if (reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
                reconnectTimeoutRef.current = setTimeout(() => {
                    connectWebSocket();
                    setReconnectDelay(nextDelay);
                }, nextDelay);
            } else {
                setWsError('Max reconnection attempts reached. Please refresh the page.');
            }
        };

        return () => {
            if (wsRef.current) {
                wsRef.current.close();
            }
            if (reconnectTimeoutRef.current) {
                clearTimeout(reconnectTimeoutRef.current);
            }
            
        };
    }, [token, updateGrid, visibleQuadrants, subscribeToQuadrant, reconnectDelay]);

    const handleVisibilityChange = useCallback((newVisibleQuadrants) => {
        setVisibleQuadrants((prevVisible) => {
            const toSubscribe = newVisibleQuadrants.filter(id => !prevVisible.has(id));
            const toUnsubscribe = Array.from(prevVisible).filter(id => !newVisibleQuadrants.has(id));

            toSubscribe.forEach(quadrantId => subscribeToQuadrant(quadrantId));
            toUnsubscribe.forEach(quadrantId => unsubscribeFromQuadrant(quadrantId));

            return new Set(newVisibleQuadrants);
        });
    }, [subscribeToQuadrant, unsubscribeFromQuadrant]);

    useEffect(() => {
        if (token && !initialFetchDone) {
            console.log('Token available, fetching grid and connecting WebSocket');
            fetchGrid();
            reconnectAttemptsRef.current = 0;
            return connectWebSocket();
        } else if (!token) {
            localStorage.removeItem('token');
            if (wsRef.current) {
                wsRef.current.close();
            }
            setGrid(new Uint8Array(GRID_SIZE * GRID_SIZE));
            setInitialFetchDone(false);  // Reset this when logging out
        }
    }, [token, fetchGrid, connectWebSocket, initialFetchDone]);

    const handleGoogleSignIn = useCallback(async (tokenResponse) => {
        try {
            setIsLoading(true);
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
        } finally {
            setIsLoading(false);
        }
    }, []);

    useEffect(() => {
        const checkActivity = () => {
            const now = Date.now();
            if (now - lastActivityRef.current > INACTIVITY_TIMEOUT) {
                if (wsRef.current) {
                    wsRef.current.close();
                }
            }
        };

        const intervalId = setInterval(checkActivity, 60000); // Check every minute

        return () => clearInterval(intervalId);
    }, []);

    const handleActivity = useMemo(() => debounce(() => {
        lastActivityRef.current = Date.now();
        if (wsRef.current && wsRef.current.readyState === WebSocket.CLOSED) {
            connectWebSocket();
        }
    }, 200), [connectWebSocket]);

    useEffect(() => {
        window.addEventListener('mousemove', handleActivity);
        window.addEventListener('keydown', handleActivity);

        return () => {
            window.removeEventListener('mousemove', handleActivity);
            window.removeEventListener('keydown', handleActivity);
        };
    }, [handleActivity]);

    const handlePixelUpdate = useCallback(async (x, y) => {
        if (!token) {
            setError('Please sign in to update pixels');
            return;
        }

        try {
            const response = await fetch(`${API_BASE_URL}/api/draw`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${token}`
                },
                body: JSON.stringify({x, y, color: selectedColor}),
            });

            if (!response.ok) {
                throw new Error('Failed to update pixel');
            }

            // Optimistically update the grid
            updateGrid(x, y, selectedColor);
        } catch (err) {
            setError(err.message);
        }
    }, [token, selectedColor, updateGrid]);

    const handleSignOut = useCallback(() => {
        googleLogout();
        setUser(null);
        setToken(null);
        setGrid(new Uint8Array(GRID_SIZE * GRID_SIZE));
        if (wsRef.current) wsRef.current.close();
    }, [setGrid]);

    return (
        <div className="container d-flex flex-column align-items-center justify-content-center min-vh-100">
            <h1 className="mb-4">r/place Clone</h1>
            {isLoading ? (
                <div>Loading...</div>
            ) : user ? (
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
                    onError={() => setError('Login Failed')}
                />
            )}
        
            {
                user ? (
                    <>
                    <Button onClick={handleSignOut} className="mb-4">Log Out</Button>
                    </>
             
                ) : (<></>)
            }
            {error && <Alert variant="danger" className="mt-3">{error}</Alert>}
            {wsError && <Alert variant="warning" className="mt-3">{wsError}</Alert>}
        </div>
    );
};

export default RPlaceClone;