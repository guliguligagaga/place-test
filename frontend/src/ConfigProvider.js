import React from 'react';
import useConfig from './hooks/useConfig';

export const ConfigContext = React.createContext(null);

export const ConfigProvider = ({ children }) => {
  const { config, loading, error } = useConfig();

  if (loading) return <div>Loading configuration...</div>;
  if (error) return <div>Error loading configuration: {error}</div>;
  if (!config) return null;

  return (
    <ConfigContext.Provider value={config}>
      {children}
    </ConfigContext.Provider>
  );
};