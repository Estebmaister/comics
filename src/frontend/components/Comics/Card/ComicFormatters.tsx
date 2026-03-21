import React from 'react';
import { Genres, Publishers } from '../../../util/ComicClasses';
import urlSwitchData from '../../../../scrape/url_switch.json';
import type { ComicLookupValue } from '../types';

const urlSwitch: { [key: string]: string[] } = urlSwitchData;

export const publishersHandler = (publishers: ComicLookupValue[]): React.ReactNode[] =>
  publishers.flatMap((id, index) => {
    const publisherName = Publishers[+id];
    const links = urlSwitch[publisherName];
    const link = links?.[0];
    return [
      <a key={index} href={link} rel="noreferrer" target="_blank">
        {publisherName}
      </a>,
      index < publishers.length - 1 ? ', ' : null,
    ];
  });

export const genresHandler = (genres: ComicLookupValue[]) =>
  genres.map(id => Genres[+id]).join(', ');
