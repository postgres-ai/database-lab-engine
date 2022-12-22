# Biuld phase.
FROM node:16.13.0-alpine as build

WORKDIR /app

COPY ./ui/ .

# RUN --mount=type=bind,id=pnpm,source=.pnpm-store,target=/app/.pnpm-store

ARG API_URL_PREFIX
ENV REACT_APP_API_URL_PREFIX ${API_URL_PREFIX}

ARG WS_URL_PREFIX
ENV REACT_APP_WS_URL_PREFIX ${WS_URL_PREFIX}

RUN apk add --no-cache --update git && \
    npm i -g pnpm; \
    pnpm config set store-dir /app/.pnpm-store; \
    pnpm set verify-store-integrity false; \
    pnpm --filter @postgres.ai/ce i; \
    pnpm --filter @postgres.ai/ce build

# Run phase.
FROM nginx:1.20.1-alpine as run

COPY --from=build /app/packages/ce/build /srv/ce
COPY ./ui/packages/ce/nginx.conf /etc/nginx/conf.d/ce.conf.template
COPY ./ui/packages/ce/docker-entrypoint.sh /

RUN rm -f /etc/nginx/conf.d/default.conf && chmod +x /docker-entrypoint.sh

EXPOSE 2346

ENTRYPOINT ["/docker-entrypoint.sh"]

CMD ["nginx", "-g", "daemon off;"]
