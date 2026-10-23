FROM node:24-alpine

WORKDIR /app

# Copiar configuración de dependencias
COPY tecnologia/boilerplate-clean-arquitecture/package.json ./

# Copiar código fuente
COPY tecnologia/boilerplate-clean-arquitecture/src ./src
COPY tecnologia/boilerplate-clean-arquitecture/tsconfig.json ./

EXPOSE 3000

ENV PORT=3000
ENV NODE_ENV=production

# Ejecutar directamente con Node 24 experimental-strip-types
CMD ["node", "--experimental-strip-types", "src/main.ts"]
