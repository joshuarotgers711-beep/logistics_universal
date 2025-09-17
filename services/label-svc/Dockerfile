# Build stage
FROM node:20-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm ci || npm i
COPY tsconfig.json ./
COPY src ./src
RUN npm run build

# Runtime stage
FROM node:20-alpine
WORKDIR /app
ENV NODE_ENV=production
# Puppeteer deps (alpine minimal)
RUN apk add --no-cache chromium nss
ENV PUPPETEER_EXECUTABLE_PATH=/usr/bin/chromium-browser
COPY --from=build /app/dist ./dist
COPY package*.json ./
RUN npm ci --omit=dev || npm i --omit=dev
EXPOSE 8084
CMD ["node", "-r", "./dist/tracing.js", "dist/main.js"]

