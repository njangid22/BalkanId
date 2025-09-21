"use client";

import { useQuery } from "urql";

const DASHBOARD_QUERY = /* GraphQL */ `
  query FilesDashboard($filter: FileFilter) {
    viewer {
      id
      email
      name
      role
      quotaBytes
      createdAt
    }
    files(filter: $filter) {
      totalCount
      nodes {
        id
        filenameOriginal
        sizeBytesOriginal
        mimeDeclared
        mimeDetected
        uploadedAt
        downloadCount
        deduped
        tags
      }
    }
    storageStats {
      totalUsageBytes
      originalUsageBytes
      savingsBytes
      savingsPercent
    }
  }
`;

interface FileNode {
  id: string;
  filenameOriginal: string;
  sizeBytesOriginal: number;
  mimeDeclared?: string | null;
  mimeDetected?: string | null;
  uploadedAt: string;
  downloadCount: number;
  deduped: boolean;
  tags: string[];
}

interface FilesDashboardResponse {
  viewer: {
    id: string;
    email: string;
    name?: string | null;
    role: string;
    quotaBytes: number;
    createdAt: string;
  } | null;
  files: {
    totalCount: number;
    nodes: FileNode[];
  };
  storageStats: {
    totalUsageBytes: number;
    originalUsageBytes: number;
    savingsBytes: number;
    savingsPercent: number;
  };
}

export function useFilesDashboard(filter?: Record<string, unknown>) {
  const [result, reexecute] = useQuery<FilesDashboardResponse>({
    query: DASHBOARD_QUERY,
    variables: { filter: filter ?? null },
    requestPolicy: "cache-and-network"
  });

  return {
    data: result.data,
    fetching: result.fetching,
    error: result.error,
    refetch: () => reexecute({ requestPolicy: "network-only" })
  };
}
