FROM node:24-alpine AS build

WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.29-alpine

COPY docker/default.conf /etc/nginx/conf.d/default.conf
COPY docker/40-argyll-env.sh /docker-entrypoint.d/40-argyll-env.sh
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 3001
