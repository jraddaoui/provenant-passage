# syntax=docker/dockerfile:1.4
# SPDX-FileCopyrightText: Artefactual Systems Inc. and Enduro contributors
# SPDX-FileCopyrightText: 2026 Artefactual Systems Limited
# SPDX-License-Identifier: Apache-2.0
FROM node:26.13 AS builder
WORKDIR /app
COPY --link package.json package-lock.json ./
RUN --mount=type=cache,target=/app/.npm npm ci --cache /app/.npm --ignore-scripts --no-audit --no-fund
COPY --link . .
RUN npm run build && node hack/collect-notices.mjs
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0", "--port", "80"]

FROM nginx:alpine AS provenant-ui
RUN apk add --no-cache bash
ENV PROVENANT_UI_DIST=/dist PROVENANT_UI_ROOT=/usr/share/nginx/html
COPY --from=builder --link /app/dist /dist
COPY hack/default.conf.template /etc/nginx/templates/default.conf.template
COPY hack/inject-vite-envs.sh /docker-entrypoint.d/50-inject-vite-envs.sh
RUN chmod +x /docker-entrypoint.d/50-inject-vite-envs.sh
