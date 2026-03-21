import React, { SetStateAction, useState } from 'react';
import config from '../../../util/Config';
import { useToast } from '../../Toast/ToastProvider';
import { RailActionButton } from '../Actions/FloatingActionRail';

const SERVER = config.SERVER;

const scrape = async (
  setShowLoader: { (value: SetStateAction<boolean>): void; },
  server = SERVER
) => {
  let success = true;
  setShowLoader(true);
  await fetch(`${server}/scrape`, {
    method: 'GET',
    headers: { 'Content-Type': 'application/json' },
  })
    .then((response) => response.json())
    .then((data) => {
      console.debug(data);
      if (data?.message === 'Internal Server Error') success = false;
    })
    .catch((err) => {
      console.debug(err.message);
      success = false;
    });
  setShowLoader(false);
  return success;
};

interface ScrapeButtonProps {
  onSuccess?: () => void;
}

const ScrapeButton = ({ onSuccess }: ScrapeButtonProps) => {
  const [showLoader, setShowLoader] = useState(false);
  const toast = useToast();

  const handleOpenScrapeButtonModal = async () => {
    if (await scrape(setShowLoader)) {
      toast.success({
        title: 'Catalog refreshed',
        description: 'Scraping finished and the comics list was refreshed.',
      });
      onSuccess?.();
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
