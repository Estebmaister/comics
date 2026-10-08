import React, { SetStateAction, useCallback, useEffect, useState } from 'react';
import config from '../../../util/Config';
import { useToast } from '../../Toast/ToastProvider';
import { RailActionButton } from '../Actions/FloatingActionRail';

const SERVER = config.SERVER;

export type ScrapeStatusResponse = {
  running: boolean;
  source?: string;
  last_completed_at?: string;
  last_completed_source?: string;
};

export const fetchScrapeStatus = async (server = SERVER): Promise<ScrapeStatusResponse | null> => {
  try {
    const response = await fetch(`${server}/scrape/status`, {
      method: 'GET',
      headers: { 'Content-Type': 'application/json' },
    });
    if (!response.ok) {
      return null;
    }
    return await response.json() as ScrapeStatusResponse;
  } catch {
    return null;
  }
};

export const scrape = async (
  setShowLoader: { (value: SetStateAction<boolean>): void; },
  server = SERVER
) => {
  setShowLoader(true);
  try {
    const response = await fetch(`${server}/scrape`, {
      method: 'GET',
      headers: { 'Content-Type': 'application/json' },
    });

    const data = await response.json().catch(() => null);
    if (response.status === 409) {
      return { ok: false as const, busy: true, source: data?.source as string | undefined };
    }
    const ok = response.ok && data?.message !== 'Internal Server Error';
    return { ok, busy: false as const };
  } catch (err) {
    return { ok: false, busy: false as const };
  } finally {
    setShowLoader(false);
  }
};

interface ScrapeButtonProps {
  onSuccess?: () => void;
}

const ScrapeButton = ({ onSuccess }: ScrapeButtonProps) => {
  const [showLoader, setShowLoader] = useState(false);
  const [remoteRunning, setRemoteRunning] = useState(false);
  const toast = useToast();

  const refreshRemoteStatus = useCallback(async () => {
    const status = await fetchScrapeStatus();
    if (status?.running) {
      setRemoteRunning(true);
      setShowLoader(true);
      return true;
    }
    setRemoteRunning(false);
    setShowLoader(false);
    return false;
  }, []);

  useEffect(() => {
    void refreshRemoteStatus();
  }, [refreshRemoteStatus]);

  useEffect(() => {
    if (!showLoader && !remoteRunning) {
      return undefined;
    }
    const interval = window.setInterval(() => {
      void fetchScrapeStatus().then((status) => {
        if (!status?.running) {
          setRemoteRunning(false);
          setShowLoader(false);
        }
      });
    }, 2500);
    return () => window.clearInterval(interval);
  }, [showLoader, remoteRunning]);

  const handleOpenScrapeButtonModal = async () => {
    const status = await fetchScrapeStatus();
    if (status?.running) {
      setRemoteRunning(true);
      setShowLoader(true);
      toast.info({
        title: 'Scrape already running',
        description: status.source
          ? `A ${status.source} scrape is in progress.`
          : 'Another scrape is in progress.',
      });
      return;
    }

    const result = await scrape(setShowLoader);
    if (result.ok) {
      toast.success({
        title: 'Catalog refreshed',
        description: 'Scraping finished and the comics list was refreshed.',
      });
      onSuccess?.();
      return;
    }

    if (result.busy) {
      setRemoteRunning(true);
      setShowLoader(true);
      toast.info({
        title: 'Scrape already running',
        description: result.source
          ? `A ${result.source} scrape is in progress. Try again later.`
          : 'Another scrape is in progress. Try again later.',
      });
      return;
    }

    toast.error({
      title: 'Scrape failed',
      description: 'The backend could not finish the scrape request.',
    });
  };

  const busy = showLoader || remoteRunning;

  return (
    <RailActionButton
      eyebrow="Sync"
      title={busy ? 'Scraping' : 'Scrape'}
      description={busy ? 'Refreshing catalog data...' : 'Refresh scraped sources'}
      tone="neutral"
      onClick={handleOpenScrapeButtonModal}
      disabled={busy}
      aria-label="Scrape sources"
    />
  );
};

export default ScrapeButton;
