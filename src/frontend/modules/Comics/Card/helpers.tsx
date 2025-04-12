import React from 'react';
import { Genres, Publishers } from '../../../util/ComicClasses';
import urlSwitchData from '../../../../scrape/url_switch.json';
const urlSwitch: { [key: string]: string[] } = urlSwitchData;

export const publishersHandler = (publishers: string[]) => {
  const elements: React.ReactNode[] = [];

  publishers.forEach((element: string, index: number) => {
    const publisherName = Publishers[+element];
    const links = urlSwitch[publisherName];
    let link;
    if (links && links.length > 0) link = links[0];
    elements.push(
      <a key={index} href={link} rel="noreferrer" target="_blank">
        {publisherName}
      </a>
    );
    if (index < publishers.length - 1) {
      elements.push(', ');
    }
  });
  return elements;
}

export const genresHandler = (genres: string[]) => {
  const return_array: string[] = [];
  genres.forEach((element: string) =>
    return_array.push(Genres[+element]))
  return return_array.join(', ');
}
