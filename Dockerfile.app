# --- Stage 1: Build ---
FROM denoland/deno:latest AS builder

WORKDIR /app

COPY app/package.json app/deno.lock ./

RUN deno install

COPY app/ .

RUN deno task build

# --- Stage 2: Prod ---
FROM nginx:alpine AS final

COPY --from=builder /app/build /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
