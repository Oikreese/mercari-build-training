FROM node:20-alpine
WORKDIR /app

RUN addgroup -S mercari && adduser -S trainee -G mercari

COPY . .

RUN chown -R trainee:mercari /app && chmod -R 755 /app

USER trainee

RUN npm install
RUN npm run build

EXPOSE 3000

CMD ["npm", "start", "--", "--host", "0.0.0.0", "--port", "3000"]