#--------------------------------------------------------------------------
# Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
# All Rights Reserved
# Unauthorized copying of this file, via any medium is strictly prohibited
# Proprietary and confidential
#--------------------------------------------------------------------------

# Build phase.
FROM node:16.14-alpine as build

WORKDIR /app

COPY ./ui/ .

# RUN --mount=type=bind,id=pnpm,source=.pnpm-store,target=/app/.pnpm-store

ARG ARG_REACT_APP_API_SERVER
ENV REACT_APP_API_SERVER=$ARG_REACT_APP_API_SERVER

ARG ARG_PUBLIC_URL
ENV PUBLIC_URL=$ARG_PUBLIC_URL

ARG ARG_REACT_APP_SIGNIN_URL
ENV REACT_APP_SIGNIN_URL=$ARG_REACT_APP_SIGNIN_URL

ARG ARG_REACT_APP_AUTH_URL
ENV REACT_APP_AUTH_URL=$ARG_REACT_APP_AUTH_URL

ARG ARG_REACT_APP_ROOT_URL
ENV REACT_APP_ROOT_URL=$ARG_REACT_APP_ROOT_URL

ARG ARG_REACT_APP_WS_SERVER
ENV REACT_APP_WS_SERVER=$ARG_REACT_APP_WS_SERVER

ARG ARG_REACT_APP_EXPLAIN_DEPESZ_SERVER
ENV REACT_APP_EXPLAIN_DEPESZ_SERVER=$ARG_REACT_APP_EXPLAIN_DEPESZ_SERVER

ARG ARG_REACT_APP_EXPLAIN_PEV2_SERVER
ENV REACT_APP_EXPLAIN_PEV2_SERVER=$ARG_REACT_APP_EXPLAIN_PEV2_SERVER

ARG ARG_REACT_APP_STRIPE_API_KEY
ENV REACT_APP_STRIPE_API_KEY=$ARG_REACT_APP_STRIPE_API_KEY

ARG ARG_REACT_APP_SENTRY_DSN
ENV REACT_APP_SENTRY_DSN=$ARG_REACT_APP_SENTRY_DSN

RUN apk add --no-cache --update git && \
    npm i -g pnpm@7.30.5; \
    pnpm config set store-dir /app/.pnpm-store; \
    pnpm set verify-store-integrity false; \
    pnpm --filter @postgres.ai/platform i; \
    pnpm --filter @postgres.ai/platform build

# Run phase.
FROM nginx:1.20.1-alpine as run

COPY --from=build /app/packages/platform/build /srv/platform
COPY ./ui/packages/platform/nginx.conf /etc/nginx/conf.d/platform.conf
RUN rm -rf /etc/nginx/conf.d/default.conf

CMD ["nginx", "-g", "daemon off;"]
