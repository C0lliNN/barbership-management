# frontend/Dockerfile — placeholder until Item 005 (Next.js PWA)
# Item 005 will replace server.js with a Next.js app and update this file.

FROM node:20-alpine

WORKDIR /app

# No npm dependencies — server.js uses only Node built-ins.
COPY server.js .

ENV PORT=3000
ENV NEXT_PUBLIC_API_URL=http://localhost:8080

EXPOSE 3000

# Health check so docker compose can gate dependent services
# Use 127.0.0.1, not localhost — busybox wget on Alpine resolves localhost to ::1
# (IPv6) but Node listens on 0.0.0.0 (IPv4 only).
HEALTHCHECK --interval=5s --timeout=5s --retries=10 --start-period=10s \
  CMD wget -qO- http://127.0.0.1:3000/health || exit 1

CMD ["node", "server.js"]
