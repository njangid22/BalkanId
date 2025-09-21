"use client";

import { useQuery } from "urql";

const VIEWER_QUERY = /* GraphQL */ `
  query ViewerSummary {
    viewer {
      id
      email
      name
      role
      quotaBytes
      createdAt
    }
  }
`;

export function useViewer() {
  const [result] = useQuery({ query: VIEWER_QUERY });
  const { data, fetching, error } = result;
  return {
    viewer: data?.viewer ?? null,
    fetching,
    error
  };
}
