import { useState, useEffect, useCallback } from 'react';
import { check } from '@tauri-apps/plugin-updater';
import { relaunch } from '@tauri-apps/plugin-process';

interface UpdateStatus {
  available: boolean;
  version?: string;
  body?: string;
  downloading: boolean;
  progress: number;
  error?: string;
}

const useUpdater = () => {
  // Intake is a separate fork: upstream Xplorer releases must never replace it.
  const updatesDisabled = import.meta.env.VITE_INTAKE_MODE === '1';
  const [status, setStatus] = useState<UpdateStatus>({
    available: false,
    downloading: false,
    progress: 0,
  });

  const checkForUpdate = useCallback(async () => {
    if (updatesDisabled) return null;
    try {
      const update = await check();
      if (update) {
        setStatus((prev) => ({
          ...prev,
          available: true,
          version: update.version,
          body: update.body ?? undefined,
          error: undefined,
        }));
        return update;
      }
    } catch (err) {
      console.warn('Failed to check for updates:', err);
      setStatus((prev) => ({
        ...prev,
        error: err instanceof Error ? err.message : String(err),
      }));
    }
    return null;
  }, [updatesDisabled]);

  const installUpdate = useCallback(async () => {
    if (updatesDisabled) return;
    try {
      const update = await check();
      if (!update) return;

      setStatus((prev) => ({ ...prev, downloading: true, progress: 0 }));

      let downloaded = 0;
      let contentLength = 0;

      await update.downloadAndInstall((event) => {
        if (event.event === 'Started') {
          contentLength = (event.data as { contentLength?: number }).contentLength ?? 0;
        } else if (event.event === 'Progress') {
          downloaded += (event.data as { chunkLength: number }).chunkLength;
          const progress = contentLength > 0 ? (downloaded / contentLength) * 100 : 0;
          setStatus((prev) => ({ ...prev, progress }));
        } else if (event.event === 'Finished') {
          setStatus((prev) => ({ ...prev, progress: 100 }));
        }
      });

      await relaunch();
    } catch (err) {
      console.warn('Failed to install update:', err);
      setStatus((prev) => ({
        ...prev,
        downloading: false,
        error: err instanceof Error ? err.message : String(err),
      }));
    }
  }, [updatesDisabled]);

  const dismissUpdate = useCallback(() => {
    setStatus({
      available: false,
      downloading: false,
      progress: 0,
    });
  }, []);

  useEffect(() => {
    if (updatesDisabled) return;
    const timer = setTimeout(() => checkForUpdate(), 5000);
    const interval = setInterval(() => checkForUpdate(), 4 * 60 * 60 * 1000);
    return () => {
      clearTimeout(timer);
      clearInterval(interval);
    };
  }, [checkForUpdate, updatesDisabled]);

  return { status, checkForUpdate, installUpdate, dismissUpdate };
};

export default useUpdater;
