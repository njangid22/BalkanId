# Use Node LTS for building
FROM node:20-bookworm-slim AS deps
WORKDIR /app

# Install dependencies (respect lockfile if present)
COPY package.json package-lock.json* pnpm-lock.yaml* yarn.lock* ./
RUN \
    if [ -f package-lock.json ]; then npm ci; \
    elif [ -f pnpm-lock.yaml ]; then npm i -g pnpm && pnpm i --frozen-lockfile; \
    elif [ -f yarn.lock ]; then yarn install --frozen-lockfile; \
    else npm i; fi

FROM node:20-bookworm-slim AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Build Next.js in standalone mode
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

# Production runtime (standalone output)
FROM node:20-bookworm-slim AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# User for security
RUN addgroup -g 1001 -S nodejs && adduser -S nextjs -u 1001

# Copy standalone build
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

# Expose and run
EXPOSE 3000
USER nextjs
ENV PORT=3000
CMD ["node", "server.js"]
