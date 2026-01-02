
import { useState, useEffect, useCallback } from 'react';
import { AppSettings } from '../types.ts';

// Session-only storage for the API key
let sessionApiKey = '';

const DEFAULTS: AppSettings = {
  geminiApiKey: '',
  proxyUrl: '',
  useProxy: false,
  debugMode: false,
  contextKeywords: {},
};

const getStoredSettings = (): Partial<AppSettings> => {
  try {
    const item = window.localStorage.getItem('forensicAppSettings');
    return item ? JSON.parse(item) : {};
  } catch (error) {
    console.error('Error reading settings from localStorage', error);
    return {};
  }
};

export function useSettings() {
  const [settings, setSettings] = useState<AppSettings>(() => ({
    ...DEFAULTS,
    ...getStoredSettings(),
    geminiApiKey: sessionApiKey, // Get from session variable on init
  }));

  useEffect(() => {
    try {
      const settingsToStore = { ...settings };
      // Do not store the API key in localStorage for security reasons
      delete (settingsToStore as Partial<AppSettings>).geminiApiKey;
      window.localStorage.setItem('forensicAppSettings', JSON.stringify(settingsToStore));
    } catch (error) {
      console.error('Error saving settings to localStorage', error);
    }
  }, [settings]);

  const setSetting = useCallback(<K extends keyof AppSettings>(key: K, value: AppSettings[K]) => {
    if (key === 'geminiApiKey') {
      sessionApiKey = value as string;
    }
    setSettings(prev => ({ ...prev, [key]: value }));
  }, []);

  return { settings, setSetting };
}
