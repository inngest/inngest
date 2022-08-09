FROM node:16.15-stretch-slim as root
# Install openssl in slim, allowing modules which require this to work.
# Without this, Prisma etc. will not be able to connect to databases.
RUN apt-get update && apt-get install -y openssl && rm -rf /var/lib/apt/lists/*

FROM root AS builder

# Create a build stage, which compiles TS to regular ol' JS.
WORKDIR /opt/
COPY package*.json *.lock tsconfig.json .
RUN npm install
COPY . .
RUN npm run build

# This is the output image, which contains only the prebuilt JS.
FROM root
WORKDIR /opt/
COPY package*.json /opt/
RUN npm install --production
COPY --from=builder /opt/build /opt/build
# Some modules (ie. prisma...) need the original source to work.
COPY --from=builder /opt/src /opt/src
ENTRYPOINT ["node", "--no-warnings", "--enable-source-maps", "./build/_init.js"]
