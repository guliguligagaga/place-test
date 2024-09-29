import React, {useCallback, useEffect, useState, useMemo} from 'react';
import {Alert, Button} from 'react-bootstrap';
import {GoogleLogin, googleLogout} from '@react-oauth/google';
import useGrid from '../hooks/useGrid';
import PixelGrid from './PixelGrid';
import ColorPicker from './ColorPicker';
import {debounce} from 'lodash';
import styled from 'styled-components';
import {GRID_SIZE, INACTIVITY_TIMEOUT, MAX_RECONNECT_ATTEMPTS, COLORS} from '../utils/constants';

const AppContainer = styled.div`
    background: linear-gradient(to bottom right, #f0f0f0, #e0e0e0);
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 20px;
`;

const Header = styled.h1`
    color: #333;
    margin-bottom: 20px;
`;

const SignOutButton = styled.button`
    background-color: #f44336;
    color: white;
    border: none;
    padding: 10px 20px;
    border-radius: 5px;
    cursor: pointer;
    transition: background-color 0.3s;

    &:hover {
        background-color: #d32f2f;
    }
`;

const ColorPickerContainer = styled.div`
    display: flex;
    justify-content: center;
    margin-bottom: 20px;
`;

const ColorOption = styled.div`
    width: 30px;
    height: 30px;
    border-radius: 50%;
    margin: 0 5px;
    cursor: pointer;
    border: 2px solid ${props => props.selected ? '#333' : 'transparent'};
    transition: transform 0.2s;

    &:hover {
        transform: scale(1.1);
    }
`;

const GridContainer = styled.div`
    background: white;
    border-radius: 10px;
    box-shadow: 0 10px 20px rgba(0, 0, 0, 0.19), 0 6px 6px rgba(0, 0, 0, 0.23);
    overflow: hidden;
`;

const RPlaceClone = () => {
    const [grid, setGrid, updateGrid] = useGrid(GRID_SIZE * GRID_SIZE);
    const [selectedColor, setSelectedColor] = useState(0);
    const [error, setError] = useState(null);
    const [token, setToken] = useState(() => localStorage.getItem('token'));
    const [isLoading, setIsLoading] = useState(false);
    const [wsError, setWsError] = useState(null);
    const [reconnectDelay, setReconnectDelay] = useState(1000);
    const [initialFetchDone, setInitialFetchDone] = useState(false);
    const [quadrants, setQuadrants] = useState([]);
    const [subscribedQuadrants, setSubscribedQuadrants] = useState(new Set());
    const [connectedClients, setConnectedClients] = useState(0);
    const [isSignedOut, setIsSignedOut] = useState(false);

    const wsRef = React.useRef(null);
    const lastActivityRef = React.useRef(Date.now());
    const reconnectAttemptsRef = React.useRef(0);

    useEffect(() => {
        const storedToken = localStorage.getItem('token');
        if (storedToken) {
            setToken(storedToken);
        }
    }, []);

    const fetchGrid = useCallback(async () => {
        if (!token || initialFetchDone) return;
        console.log('Fetching grid...');

        try {
            setIsLoading(true);
            const response = await fetch(`${window.location.origin}/api/grid`, {
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
            setInitialFetchDone(true);
        } catch (err) {
            console.error('Error fetching grid:', err);
            setError('Failed to fetch grid: ' + err.message);
        } finally {
            console.log('Grid fetch completed');
            setIsLoading(false);
        }
    }, [token, setGrid, initialFetchDone]);

    const connectWebSocket = useCallback(() => {
        if (isSignedOut) {
            console.log('User is signed out. Skipping WebSocket connection.');
            return;
        }
        if (wsRef.current) {
            wsRef.current.close();
        }

        const ws = new WebSocket(`${window.location.origin.replace(/^http/, 'ws')}/ws?token=${token}`);

        ws.onopen = () => {
            console.log('WebSocket connected');
        };

        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            updateGrid(data.x, data.y, data.color);
        };

        ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        ws.onclose = () => {
            console.log('WebSocket disconnected');
            setTimeout(() => {
                if (reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
                    reconnectAttemptsRef.current++;
                    reconnectWebSocket();
                } else {
                    setError('Unable to connect to the server. Please try again later.');
                }
            }, 1000 * Math.pow(2, reconnectAttemptsRef.current));
        };

        wsRef.current = ws;
    }, [token, updateGrid, isSignedOut]);

    const reconnectWebSocket = useCallback(() => {
        if (isSignedOut) {
            console.log('User is signed out. Skipping WebSocket reconnection.');
            return;
        }
        if (reconnectAttemptsRef.current >= MAX_RECONNECT_ATTEMPTS) {
            console.log('Max reconnection attempts reached. Stopping reconnection.');
            setWsError('Unable to establish WebSocket connection. Please refresh the page.');
            return;
        }

        reconnectAttemptsRef.current++;
        const nextDelay = Math.min(30000, reconnectDelay * 2);
        console.log(`Reconnecting in ${nextDelay}ms (attempt ${reconnectAttemptsRef.current} of ${MAX_RECONNECT_ATTEMPTS})`);

        setTimeout(() => {
            connectWebSocket();
            setReconnectDelay(nextDelay);
        }, nextDelay);
    }, [connectWebSocket, reconnectDelay, isSignedOut]);

    useEffect(() => {
        if (token && !initialFetchDone && !isSignedOut) {
            console.log('Token available, fetching grid and connecting WebSocket');
            fetchGrid();
            reconnectAttemptsRef.current = 0;
            return connectWebSocket();
        } else if (!token || isSignedOut) {
            localStorage.removeItem('token');
            if (wsRef.current) {
                wsRef.current.close();
            }
            setGrid(new Uint8Array(GRID_SIZE * GRID_SIZE));
            setInitialFetchDone(false);
        }
    }, [token, fetchGrid, connectWebSocket, initialFetchDone, setGrid, isSignedOut]);


    const handleGoogleSignIn = useCallback(async (tokenResponse) => {
        try {
            setIsLoading(true);
            setIsSignedOut(false);
            const res = await fetch(`${window.location.origin}/api/auth/signIn`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    provider: "google",
                    token: tokenResponse.credential
                }),
                credentials: 'include',
            });

            if (res.ok) {
                const data = await res.json();
                setToken(data.token);
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
    }, [fetchGrid, connectWebSocket]);

    useEffect(() => {
        const checkTokenExpiration = () => {
            if (token) {
                const tokenData = JSON.parse(atob(token.split('.')[1]));
                const expirationTime = tokenData.exp * 1000; // Convert to milliseconds
                const currentTime = Date.now();
                const timeUntilExpiration = expirationTime - currentTime;

                if (timeUntilExpiration < 300000) { // 5 minutes before expiration
                    renewToken();
                }
            }
        };

        const intervalId = setInterval(checkTokenExpiration, 60000); // Check every minute

        return () => clearInterval(intervalId);
    }, [token]);

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
            const response = await fetch(`${window.location.origin}/api/draw`, {
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
        setToken(null);
        setGrid(new Uint8Array(GRID_SIZE * GRID_SIZE));
        if (wsRef.current) wsRef.current.close();
        setIsSignedOut(true);
        localStorage.removeItem('token');
    }, [setGrid]);

    const subscribeToQuadrant = useCallback((quadrantId) => {
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify({type: 'Subscribe', payload: {quadrant_id: quadrantId}}));
            setSubscribedQuadrants(prev => new Set(prev).add(quadrantId));
        }
    }, []);

    const unsubscribeFromQuadrant = useCallback((quadrantId) => {
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify({type: 'Unsubscribe', payload: {quadrant_id: quadrantId}}));
            setSubscribedQuadrants(prev => {
                const newSet = new Set(prev);
                newSet.delete(quadrantId);
                return newSet;
            });
        }
    }, []);

    const renewToken = async () => {
        try {
            const res = await fetch(`${window.location.origin}/api/auth/renew`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                },
            });

            if (res.ok) {
                const data = await res.json();
                setToken(data.token);
                localStorage.setItem('token', data.token);
            } else {
                throw new Error('Failed to renew token');
            }
        } catch (err) {
            console.error('Error renewing token:', err);
            // If token renewal fails, log out the user
            handleSignOut();
        }
    };

    return (
        <AppContainer>
            <Header>r/place Clone</Header>
            {isLoading ? (
                <div>Loading...</div>
            ) : token ? (
                <>
                    <SignOutButton onClick={handleSignOut}>Sign Out</SignOutButton>
                    <ColorPickerContainer>
                        <ColorPicker selectedColor={selectedColor} onColorSelect={setSelectedColor} colors={COLORS}/>
                    </ColorPickerContainer>
                    <GridContainer>
                        <PixelGrid
                            grid={grid}
                            onPixelClick={handlePixelUpdate}
                            size={GRID_SIZE}
                            colors={COLORS}
                            quadrants={quadrants}
                            subscribedQuadrants={subscribedQuadrants}
                            onSubscribe={subscribeToQuadrant}
                            onUnsubscribe={unsubscribeFromQuadrant}
                            connectedClients={connectedClients}
                        />
                    </GridContainer>
                </>
            ) : (
                <GoogleLogin
                    onSuccess={handleGoogleSignIn}
                    onError={() => setError('Login Failed')}
                />
            )}

            {error && <Alert variant="danger" className="mt-3">{error}</Alert>}
            {wsError && <Alert variant="warning" className="mt-3">{wsError}</Alert>}
        </AppContainer>
    );
};

export default RPlaceClone;