# Build phase.
FROM node:25-alpine as build

WORKDIR /app

COPY ./ui/ .

ARG API_URL_PREFIX
ENV VITE_API_URL_PREFIX ${API_URL_PREFIX}

ARG WS_URL_PREFIX
ENV VITE_WS_URL_PREFIX ${WS_URL_PREFIX}

RUN apk add --no-cache --update git && \
    npm i -g pnpm@8.15.9; \
    pnpm config set store-dir /app/.pnpm-store; \
    pnpm --filter @postgres.ai/ce i; \
    pnpm --filter @postgres.ai/ce build; \
    find /app/packages/ce/build -name '*.map' -delete

# Run phase.
FROM nginx:1.29.7-alpine as run

COPY --from=build /app/packages/ce/build /srv/ce
COPY ./ui/packages/ce/nginx.conf /etc/nginx/conf.d/ce.conf.template
COPY ./ui/packages/ce/docker-entrypoint.sh /

RUN rm -f /etc/nginx/conf.d/default.conf && chmod +x /docker-entrypoint.sh

EXPOSE 2346

ENTRYPOINT ["/docker-entrypoint.sh"]

CMD ["nginx", "-g", "daemon off;"]
