"""Temporary verification script for the Asura scraper parsing logic."""
import asyncio

from scrape.asura import extract_comics_from_props, _parse_island_props
from scrape.scrapper import scrape_url


async def main():
    url = 'https://asurascans.com/'
    soup = await scrape_url(url)
    islands = soup.find_all('astro-island', attrs={'props': True})
    print(f'islands with props: {len(islands)}')

    seen = set()
    comics = []
    for island in islands:
        props = _parse_island_props(island.get('props', ''))
        if not props:
            continue
        found = extract_comics_from_props(props)
        if found:
            name = island.get('component-url', '?')
            print(f'  island {name}: {len(found)} comics')
        for c in found:
            key = c.title.strip().lower()
            if key in seen:
                continue
            seen.add(key)
            comics.append(c)

    print(f'total unique comics: {len(comics)}')
    for c in comics[:8]:
        print(f'  - {c.title!r} ch={c.chapter!r} type={c.com_type!r} '
              f'status={c.status!r} cover={c.cover_url[:60]!r}')


if __name__ == '__main__':
    asyncio.run(main())
