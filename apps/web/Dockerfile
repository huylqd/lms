# syntax=docker/dockerfile:1

# ---- deps: install exactly what package-lock.json says ----------------------
FROM node:24.22.3-alpine3.24 AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --no-audit --no-fund

# ---- build: standalone Next.js output (no env needed at build time) ---------
FROM node:24.22.3-alpine3.24 AS build
ENV NEXT_TELEMETRY_DISABLED=1
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

# ---- test: lint + typecheck (build target fails if either fails) ------------
FROM build AS test
RUN npm run lint && npx tsc --noEmit

# ---- runtime: minimal, non-root, works with a read-only rootfs --------------
# Writable paths needed at runtime: /tmp and /app/.next/cache (tmpfs in compose).
FROM node:24.22.3-alpine3.24 AS runtime
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0
# npm/npx/corepack are not needed at runtime and bundle their own vulnerable deps.
RUN rm -rf /usr/local/lib/node_modules/npm /usr/local/lib/node_modules/corepack \
        /usr/local/bin/npm /usr/local/bin/npx /usr/local/bin/corepack
WORKDIR /app
COPY --from=build --chown=node:node /app/public ./public
COPY --from=build --chown=node:node /app/.next/standalone ./
COPY --from=build --chown=node:node /app/.next/static ./.next/static
USER node
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD ["node", "-e", "fetch('http://127.0.0.1:3000/api/health').then(r=>process.exit(r.ok?0:1)).catch(()=>process.exit(1))"]
CMD ["node", "server.js"]
