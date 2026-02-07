# Stage 1: Dependencies
FROM node:20-alpine AS deps
WORKDIR /app
COPY web/package.json web/package-lock.json* ./
RUN npm ci

# Stage 2: Build
FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY web/ .
# Ensure public exists (Next.js may not create it if the project has no public assets)
RUN mkdir -p public

# Build-time env (optional; override in compose for NEXT_PUBLIC_*)
ARG NEXT_PUBLIC_API_URL
ARG NEXT_PUBLIC_ADD_CHANNEL_USERNAME
ENV NEXT_PUBLIC_API_URL=${NEXT_PUBLIC_API_URL}
ENV NEXT_PUBLIC_ADD_CHANNEL_USERNAME=${NEXT_PUBLIC_ADD_CHANNEL_USERNAME}

RUN npm run build

# Stage 3: Runner (standalone output)
FROM node:20-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV PORT=3000
EXPOSE 3000

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs
CMD ["node", "server.js"]
