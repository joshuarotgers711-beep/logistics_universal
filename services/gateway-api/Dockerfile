# Build stage
FROM node:20-alpine as build
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
COPY --from=build /app/dist ./dist
COPY package*.json ./
RUN npm ci --omit=dev || npm i --omit=dev
EXPOSE 8080
CMD ["node", "-r", "./dist/tracing.js", "dist/main.js"]

