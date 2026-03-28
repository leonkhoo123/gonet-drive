import { createContext, useState, useContext, useEffect } from 'react';
import type { ReactNode } from 'react';

interface PreferencesContextType {
  showHidden: boolean;
  setShowHidden: (value: boolean) => void;
}

const PreferencesContext = createContext<PreferencesContextType | undefined>(undefined);

export function PreferencesProvider({ children }: { children: ReactNode }) {
  const [showHidden, setShowHidden] = useState<boolean>(() => {
    const saved = localStorage.getItem('preferences_showHidden');
    // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment
    const parsed = saved ? JSON.parse(saved) : false;
    return Boolean(parsed);
  });

  useEffect(() => {
    localStorage.setItem('preferences_showHidden', JSON.stringify(showHidden));
  }, [showHidden]);

  return (
    <PreferencesContext.Provider value={{ showHidden, setShowHidden }}>
      {children}
    </PreferencesContext.Provider>
  );
}

export function usePreferences() {
  const context = useContext(PreferencesContext);
  if (context === undefined) {
    throw new Error('usePreferences must be used within a PreferencesProvider');
  }
  return context;
}
