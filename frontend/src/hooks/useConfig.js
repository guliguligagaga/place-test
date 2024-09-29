import { useState, useEffect } from 'react';

const CACHE_KEY = 'app_config';
const CACHE_EXPIRATION = 1000 * 60 * 60; // 1 hour

const useConfig = () => {
  const [config, setConfig] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchConfig = async () => {
      try {
        // Check if we have a cached config
        const cachedConfig = localStorage.getItem(CACHE_KEY);
        if (cachedConfig) {
          const { data, timestamp } = JSON.parse(cachedConfig);
          
          // Check if the cache is still valid
          if (Date.now() - timestamp < CACHE_EXPIRATION) {
            setConfig(data);
            setLoading(false);
            return;
          }
        }

        // If no valid cache, fetch from server
        const response = await fetch('/config');
        if (!response.ok) {
          throw new Error(`HTTP error! status: ${response.status}`);
        }
        const data = await response.json();

        // Cache the new config
        localStorage.setItem(CACHE_KEY, JSON.stringify({
          data,
          timestamp: Date.now()
        }));

        setConfig(data);
      } catch (e) {
        setError(e.message);
      } finally {
        setLoading(false);
      }
    };

    fetchConfig();
  }, []);

  return { config, loading, error };
};

export default useConfig;