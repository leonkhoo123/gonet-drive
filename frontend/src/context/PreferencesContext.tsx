import { createContext, useState, use, useEffect } from 'react';
import type { ReactNode } from 'react';

interface PreferencesContextType {
  showHidden: boolean;
  setShowHidden: (value: boolean) => void;
  viewMode: 'list' | 'grid';
  setViewMode: (value: 'list' | 'grid') => void;
}

const PreferencesContext = createContext<PreferencesContextType | undefined>(undefined);

export function PreferencesProvider({ children }: { children: ReactNode }) {
  const [showHidden, setShowHidden] = useState<boolean>(() => {
    const saved = localStorage.getItem('preferences_showHidden');
    const parsed: unknown = saved ? JSON.parse(saved) as unknown : false;
    return Boolean(parsed);
  });

  const [viewMode, setViewMode] = useState<'list' | 'grid'>(() => {
    const saved = localStorage.getItem('preferences_viewMode');
    return saved === 'grid' ? 'grid' : 'list';
  });

  useEffect(() => {
    localStorage.setItem('preferences_showHidden', JSON.stringify(showHidden));
  }, [showHidden]);

  useEffect(() => {
    localStorage.setItem('preferences_viewMode', viewMode);
  }, [viewMode]);

  return (
    <PreferencesContext value={{ showHidden, setShowHidden, viewMode, setViewMode }}>
      {children}
    </PreferencesContext>
  );
}

export function usePreferences() {
  const context = use(PreferencesContext);
  if (context === undefined) {
    throw new Error('usePreferences must be used within a PreferencesProvider');
  }
  return context;
}
