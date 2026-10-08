import React, { SetStateAction, useState } from 'react';
import config from '../../../util/Config';
import { useToast } from '../../Toast/ToastProvider';
import { RailActionButton } from '../Actions/FloatingActionRail';

const SERVER = config.SERVER;

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
  const toast = useToast();

  const handleOpenScrapeButtonModal = async () => {
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

  return (
    <RailActionButton
      eyebrow="Sync"
      title={showLoader ? 'Scraping' : 'Scrape'}
      description={showLoader ? 'Refreshing catalog data...' : 'Refresh scraped sources'}
      tone="neutral"
      onClick={handleOpenScrapeButtonModal}
      disabled={showLoader}
      aria-label="Scrape sources"
    />
  );
};

export default ScrapeButton;
