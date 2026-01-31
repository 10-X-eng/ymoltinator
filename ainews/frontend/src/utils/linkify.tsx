import type { ReactNode } from 'react';

const URL_REGEX = /(https?:\/\/[^\s]+|www\.[^\s]+)/g;
const URL_TEST_REGEX = /(https?:\/\/[^\s]+|www\.[^\s]+)/;

export function containsUrl(text: string): boolean {
  return URL_TEST_REGEX.test(text);
}

export function normalizeUrl(url: string): string {
  return url.startsWith('http') ? url : `https://${url}`;
}

export function renderTextWithLinks(text: string): ReactNode[] {
  const nodes: ReactNode[] = [];
  let lastIndex = 0;

  for (const match of text.matchAll(URL_REGEX)) {
    const url = match[0];
    const index = match.index ?? 0;

    if (index > lastIndex) {
      nodes.push(text.slice(lastIndex, index));
    }

    const href = normalizeUrl(url);
    nodes.push(
      <a key={`${url}-${index}`} href={href} target="_blank" rel="noopener noreferrer">
        {url}
      </a>
    );

    lastIndex = index + url.length;
  }

  if (lastIndex < text.length) {
    nodes.push(text.slice(lastIndex));
  }

  return nodes;
}
