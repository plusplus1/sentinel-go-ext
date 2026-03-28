import React, { createContext, useContext, useState, useEffect } from 'react';
import type { ReactNode } from 'react';
import axios from 'axios';

export interface AppInfo {
  id: string;
  name: string;
  desc?: string;
  env?: string;
  type?: string;
  endpoints?: string[];
  args?: Record<string, string>;
}

export interface BusinessLine {
  id: string;
  name: string;
  description: string;
  apps: AppInfo[];
}

interface AppContextType {
  apps: AppInfo[];
  businessLines: BusinessLine[];
  selectedApp: AppInfo | null;
  loading: boolean;
  selectApp: (appId: string) => void;
  refreshApps: () => Promise<void>;
  getAppFullPath: (appId: string) => string; // 返回 "业务线名 / 应用名 (描述)"
}

const AppContext = createContext<AppContextType | undefined>(undefined);

export const useApp = () => {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error('useApp must be used within an AppProvider');
  }
  return context;
};

interface AppProviderProps {
  children: ReactNode;
}

export const AppProvider: React.FC<AppProviderProps> = ({ children }) => {
  const [apps, setApps] = useState<AppInfo[]>([]);
  const [businessLines, setBusinessLines] = useState<BusinessLine[]>([]);
  const [selectedApp, setSelectedApp] = useState<AppInfo | null>(null);
  const [loading, setLoading] = useState(false);

  const fetchApps = async () => {
    setLoading(true);
    try {
      const response = await axios.get('/api/apps');
      const { code, data, message: msg } = response.data;
      if (code === 0) {
        // Transform API response to BusinessLine array and flat apps array
        // API returns: [{id: line_id, name: line_name, children: [{id: app_id, name: app_name, description, status}]}]
        let businessLinesList: BusinessLine[] = [];
        let appsList: AppInfo[] = [];
        if (data && Array.isArray(data)) {
          data.forEach((line: any) => {
            const businessLine: BusinessLine = {
              id: line.id,
              name: line.name || `业务线 ${line.id}`,
              description: line.description || '',
              apps: [],
            };
            if (line.children && Array.isArray(line.children)) {
              line.children.forEach((app: any) => {
                const appInfo: AppInfo = {
                  id: app.id,
                  name: app.name,
                  desc: app.description,
                  env: app.status === 'active' ? '生产' : '开发',
                  type: 'etcd',
                  endpoints: [],
                  args: {}
                };
                businessLine.apps.push(appInfo);
                appsList.push(appInfo);
              });
            }
            businessLinesList.push(businessLine);
          });
        }
        setBusinessLines(businessLinesList);
        setApps(appsList);
        // Clear stale selection if current app no longer exists in the list
        const stale = selectedApp && !appsList.some(a => a.id === selectedApp.id);
        if (stale) {
          setSelectedApp(appsList.length > 0 ? appsList[0] : null);
        } else if (!selectedApp && appsList.length > 0) {
          setSelectedApp(appsList[0]);
        }
      } else if (code === 401) {
        // Not authenticated, don't show error, just clear apps
        setBusinessLines([]);
        setApps([]);
      } else {
        // For other errors, log to console instead of showing popup
        console.error('获取应用列表失败:', msg || 'Unknown error');
        setBusinessLines([]);
        setApps([]);
      }
    } catch (error: any) {
      if (error.response?.status === 401) {
        // Not authenticated, clear apps
        setBusinessLines([]);
        setApps([]);
      } else {
        // Network error, log to console instead of showing popup
        console.error('请求失败:', error.message);
        setBusinessLines([]);
        setApps([]);
      }
    } finally {
      setLoading(false);
    }
  };

  const selectApp = (appId: string) => {
    const app = apps.find(a => a.id === appId);
    if (app) {
      setSelectedApp(app);
    }
  };

  const refreshApps = async () => {
    await fetchApps();
  };

  const getAppFullPath = (appId: string): string => {
    for (const line of businessLines) {
      const app = line.apps.find(a => a.id === appId);
      if (app) {
        // 格式：业务线名(业务线描述) / 应用名(应用描述)
        const linePart = line.description 
          ? `${line.name}(${line.description})`
          : line.name;
        const appPart = app.desc 
          ? `${app.name}(${app.desc})`
          : app.name;
        return `${linePart} / ${appPart}`;
      }
    }
    return appId;
  };

  useEffect(() => {
    // Check authentication first, then fetch apps if authenticated
    const checkAuthAndFetch = async () => {
      try {
        const response = await axios.get('/api/auth/me');
        const { code } = response.data;
        if (code === 0) {
          // User is authenticated, fetch apps
          await fetchApps();
        } else {
          // Not authenticated, clear apps
          setBusinessLines([]);
          setApps([]);
        }
      } catch (error) {
        // Not authenticated, clear apps
        setBusinessLines([]);
        setApps([]);
      } finally {
        // Auth check complete
      }
    };
    checkAuthAndFetch();
  }, []);

  return (
    <AppContext.Provider value={{ apps, businessLines, selectedApp, loading, selectApp, refreshApps, getAppFullPath }}>
      {children}
    </AppContext.Provider>
  );
};