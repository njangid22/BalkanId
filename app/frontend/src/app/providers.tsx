"use client";

import { cacheExchange, fetchExchange, Provider, createClient } from "urql";
import { useMemo } from "react";
import type { ReactNode } from "react";

const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export function GraphQLProvider({ children }: { children: ReactNode }) {
  const client = useMemo(
    () =>
      createClient({
        url: `${apiUrl}/graphql`,
        fetchOptions: {
          credentials: "include"
        },
        exchanges: [cacheExchange, fetchExchange]
      }),
    []
  );

  return <Provider value={client}>{children}</Provider>;
}
