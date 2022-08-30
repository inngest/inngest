FROM node:16.15-stretch-slim AS builder

# Create a build stage, which compiles TS to regular ol' JS.
WORKDIR /opt/
COPY package*.json *.lock tsconfig.json ./
RUN npm install
COPY src .
RUN npm run build

# This is the output image, which contains only the prebuilt JS.
FROM node:16.15-stretch-slim
WORKDIR /opt/
COPY package*.json /opt/
RUN npm install --production
COPY --from=builder /opt/build /opt/build
# The flag lets us use imports without .js extensions, making the transition between
# TS and JS easy.
ENTRYPOINT ["node", "./build/_init.js"]
