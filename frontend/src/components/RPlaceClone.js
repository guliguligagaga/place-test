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

const GridContainer = styled.div`
    background: linear-gradient(135deg, #f8f8f8, #e6e6e6);  
    border-radius: 1px;  
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1), 0 2px 4px rgba(0, 0, 0, 0.06);  
    border: 1px solid rgba(230, 230, 230, 0.8);  
    overflow: hidden;
    transition: transform 0.3s ease, box-shadow 0.3s ease;
`;


const RPlaceClone = () => {
    const [grid, setGrid, updateGrid] = useGrid();
    const [selectedColor, setSelectedColor] = useState(0);
    const [error, setError] = useState(null);
    const [token, setToken] = useState(() => localStorage.getItem('token'));
    const [isLoading, setIsLoading] = useState(false);
    const [wsError, setWsError] = useState(null);
    const [reconnectDelay, setReconnectDelay] = useState(1000);
    const [initialFetchDone, setInitialFetchDone] = useState(false);
    const [connectedClients, setConnectedClients] = useState(0);
    const [isSignedOut, setIsSignedOut] = useState(false);

    const reconnectAttemptsRef = React.useRef(0);
    const wsRef = React.useRef(null);
    const lastUpdateRef = React.useRef(null);

    const debouncedUpdateGrid = useCallback(
        debounce((x, y, color) => {
            updateGrid(x, y, color);
        }, 50),
        [updateGrid])

    const handlePixel = useCallback((update) => {
        const { x, y, color, Time } = update;

        // Check if this update is newer than the last one we processed
        if (!lastUpdateRef.current || Time > lastUpdateRef.current) {
            lastUpdateRef.current = Time;
            debouncedUpdateGrid(x, y, color);

        } else {
            console.log('Skipped duplicate or older update');
        }
    }, [debouncedUpdateGrid]);

    useEffect(() => {
        const storedToken = localStorage.getItem('token');
        if (storedToken) {
            setToken(storedToken);
        }
    }, []);

    const connectWebSocket = useCallback(() => {
        if (isSignedOut) {
            return;
        }
        if (wsRef.current?.readyState === 1) {
            console.log('WebSocket already connected. Skipping connection.');
            return
        }

        const ws = new WebSocket(`${window.location.origin.replace(/^http/, 'ws')}/ws?token=${token}`);

        ws.onopen = () => {
            console.log('WebSocket connected');
        };

        ws.onmessage = async (event) => {
            const data = event.data
            if (event.type === "message" && typeof(data) === "string") {
                const update = JSON.parse(event.data)
                handlePixel(update)
            } else if (data instanceof Blob) {
                // Handle Blob data (grid state)
                const arrayBuffer = await data.arrayBuffer();
                setGrid(arrayBuffer);
                setInitialFetchDone(true)
            } else {
                console.warn('Received unknown message format:', event.data);
            }
        };

        ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        ws.onclose = () => {
            setTimeout(() => {
                if (reconnectAttemptsRef.current < MAX_RECONNECT_ATTEMPTS) {
                    reconnectAttemptsRef.current++;
                    reconnectWebSocket();
                } else {
                    setError('Unable to connect to the server. Please try again later.');
                    setIsSignedOut(true)
                    wsRef.current = null;
                }
            }, 1000 * Math.pow(2, reconnectAttemptsRef.current));
        };

        wsRef.current = ws;
    }, [token, updateGrid, isSignedOut, handlePixel]);

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
            reconnectAttemptsRef.current = 0;
            return connectWebSocket();
        } else if (!token || isSignedOut) {
            localStorage.removeItem('token');
            if (wsRef.current) {
                wsRef.current.close();
            }
            setInitialFetchDone(false);
        }
    }, [token, connectWebSocket, initialFetchDone, setGrid, isSignedOut]);


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
                localStorage.setItem("token", data.token)
            } else {
                throw new Error('Failed to authenticate with the server');
            }
        } catch (err) {
            setError('Authentication failed. Please try again.');
        } finally {
            setIsLoading(false);
        }
    }, [connectWebSocket]);

    useEffect(() => {
        const checkTokenExpiration = () => {
            if (token) {
                // const tokenData = JSON.parse(atob(token.split('.')[1]));
                // const expirationTime = tokenData.exp * 1000; // Convert to milliseconds
                // const currentTime = Date.now();
                // const timeUntilExpiration = expirationTime - currentTime;
                //
                // if (timeUntilExpiration < 300000) { // 5 minutes before expiration
                //     renewToken();
                // }
            }
        };

        const intervalId = setInterval(checkTokenExpiration, 60000); // Check every minute

        return () => clearInterval(intervalId);
    }, [token]);

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