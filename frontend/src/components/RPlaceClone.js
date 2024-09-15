import React, { useCallback, useEffect, useState, useMemo, useRef } from 'react';
import { Alert, Button } from 'react-bootstrap';
import { GoogleLogin, googleLogout } from '@react-oauth/google';
import useGrid from '../hooks/useGrid';
import PixelGrid from './PixelGrid';
import ColorPicker from './ColorPicker';
import { debounce } from 'lodash';
import styled from 'styled-components';

const GRID_SIZE = 100; // Increased grid size
const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8081';
const INACTIVITY_TIMEOUT = 5 * 60 * 1000; // 5 minutes
const MAX_RECONNECT_ATTEMPTS = 5;

const COLORS = [
    '#FFFFFF', '#E4E4E4', '#888888', '#222222',
    '#FFA7D1', '#E50000', '#E59500', '#A06A42',
    '#E5D900', '#94E044', '#02BE01', '#00D3DD',
    '#0083C7', '#0000EA', '#CF6EE4', '#820080'
];

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
  box-shadow: 0 10px 20px rgba(0,0,0,0.19), 0 6px 6px rgba(0,0,0,0.23);
  overflow: hidden;
`;

const RPlaceClone = () => {
    const [grid, setGrid, updateGrid] = useGrid(GRID_SIZE * GRID_SIZE);
    const [selectedColor, setSelectedColor] = useState(0);
    const [error, setError] = useState(null);
    const [user, setUser] = useState(null);
    const [token, setToken] = useState(() => localStorage.getItem('token'));
    const [isLoading, setIsLoading] = useState(false);
    const [wsError, setWsError] = useState(null);
    const [reconnectDelay, setReconnectDelay] = useState(1000);
    const [initialFetchDone, setInitialFetchDone] = useState(false);
    const [quadrants, setQuadrants] = useState([]);
    const [subscribedQuadrants, setSubscribedQuadrants] = useState(new Set());
    const [connectedClients, setConnectedClients] = useState(0);

    const wsRef = React.useRef(null);
    const lastActivityRef = React.useRef(Date.now());
    const reconnectAttemptsRef = React.useRef(0);

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
        if (!token || wsRef.current) return;

        const wsUrl = `ws:${API_BASE_URL.replace(/^https?:/, '')}/ws`;
        wsRef.current = new WebSocket(`${wsUrl}?token=${encodeURIComponent(token)}`);
        wsRef.current.withCredentials = false;

        wsRef.current.onopen = () => {
            console.log('WebSocket connection established');
            reconnectAttemptsRef.current = 0;
            setReconnectDelay(1000);
            setWsError(null);
        };

        wsRef.current.onmessage = (event) => {
            const data = JSON.parse(event.data);
            
            if (data.type === 'configuration') {
                console.log("Received configuration", data);
                setQuadrants(data.quadrants);
                setConnectedClients(data.connectedClients || 0);
            } else if (data.type === 'update') {
                console.log("Received update", data);
                updateGrid(data.x, data.y, data.color);
            } else if (data.type === 'clientCount') {
                setConnectedClients(data.count);
            }
        };

        wsRef.current.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        wsRef.current.onclose = (event) => {
            console.log('WebSocket connection closed:', event.code, event.reason);
            if (event.code !== 1000) {
                reconnectWebSocket();
            }
        };
    }, [token, updateGrid]);

    const reconnectWebSocket = useCallback(() => {
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
    }, [connectWebSocket, reconnectDelay]);

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
            setInitialFetchDone(false);
        }
    }, [token, fetchGrid, connectWebSocket, initialFetchDone, setGrid]);

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
    }, [fetchGrid, connectWebSocket]);

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
    
    const subscribeToQuadrant = useCallback((quadrantId) => {
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify({ type: 'Subscribe', payload: { quadrant_id: quadrantId } }));
            setSubscribedQuadrants(prev => new Set(prev).add(quadrantId));
        }
    }, []);
    
    const unsubscribeFromQuadrant = useCallback((quadrantId) => {
        if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
            wsRef.current.send(JSON.stringify({ type: 'Unsubscribe', payload: { quadrant_id: quadrantId } }));
            setSubscribedQuadrants(prev => {
                const newSet = new Set(prev);
                newSet.delete(quadrantId);
                return newSet;
            });
        }
    }, []);
    
    return (
        <AppContainer>
            <Header>r/place Clone</Header>
            {isLoading ? (
                <div>Loading...</div>
            ) : user ? (
                <>
                    <SignOutButton onClick={handleSignOut}>Sign Out</SignOutButton>
                    <ColorPickerContainer>
                        <ColorPicker selectedColor={selectedColor} onColorSelect={setSelectedColor} colors={COLORS} />
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