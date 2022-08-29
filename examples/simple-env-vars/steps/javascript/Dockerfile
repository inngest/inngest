FROM node:16.15-stretch-slim
WORKDIR /opt/
COPY package*.json *lock src /opt/
RUN npm install --production
ENTRYPOINT ["node", "--es-module-specifier-resolution=node", "./_init.js"]
