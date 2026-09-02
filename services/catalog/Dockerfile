FROM node:22-alpine AS build
RUN apk add --no-cache python3 make g++ openssl
WORKDIR /app
COPY package.json ./
RUN npm install
COPY prisma ./prisma
RUN npx prisma generate
COPY tsconfig.json nest-cli.json ./
COPY src ./src
RUN npm run build

FROM node:22-alpine
RUN apk add --no-cache wget openssl
WORKDIR /app
ENV NODE_ENV=production
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/dist ./dist
COPY --from=build /app/prisma ./prisma
COPY package.json ./
EXPOSE 8002
CMD ["sh", "-c", "npx prisma db push --skip-generate && node dist/main.js"]
